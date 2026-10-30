package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type groupProfileClient interface {
	GetGroupInfo(context.Context, types.JID) (*types.GroupInfo, error)
	GetProfilePictureInfo(context.Context, types.JID, *whatsmeow.GetProfilePictureParams) (*types.ProfilePictureInfo, error)
}

// A stored human message anchors the lookup; callers never supply a target phone number.
func groupProfileHandler(client groupProfileClient, store *MessageStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ChatJID   string `json:"chat_jid"`
			MessageID string `json:"message_id"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		dec.DisallowUnknownFields()
		if dec.Decode(&req) != nil || req.MessageID == "" {
			http.Error(w, "Invalid group message", http.StatusBadRequest)
			return
		}
		group, err := types.ParseJID(req.ChatJID)
		if err != nil || group.Server != types.GroupServer {
			http.Error(w, "A group is required", http.StatusBadRequest)
			return
		}
		var sender string
		var fromMe bool
		err = store.db.QueryRow("SELECT sender,is_from_me FROM messages WHERE chat_jid=? AND id=?", req.ChatJID, req.MessageID).Scan(&sender, &fromMe)
		if err == sql.ErrNoRows || fromMe {
			http.Error(w, "Human group message required", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Message unavailable", http.StatusServiceUnavailable)
			return
		}
		member, err := parseStoredSender(sender)
		if err != nil || (member.Server != types.DefaultUserServer && member.Server != types.HiddenUserServer) {
			http.Error(w, "Member unavailable", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		info, err := client.GetGroupInfo(ctx, group)
		if err != nil || info == nil {
			http.Error(w, "Group unavailable", http.StatusServiceUnavailable)
			return
		}
		present := false
		for _, p := range info.Participants {
			if member.ToNonAD() == p.JID.ToNonAD() || member.ToNonAD() == p.PhoneNumber.ToNonAD() || member.ToNonAD() == p.LID.ToNonAD() {
				present = true
				break
			}
		}
		if !present {
			http.Error(w, "Current group member required", http.StatusForbidden)
			return
		}
		picture, pictureErr := client.GetProfilePictureInfo(ctx, member, &whatsmeow.GetProfilePictureParams{CommonGID: group})
		photoURL := ""
		if pictureErr == nil && picture != nil {
			photoURL = picture.URL
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"picture_url": photoURL, "photo_available": photoURL != ""})
	}
}

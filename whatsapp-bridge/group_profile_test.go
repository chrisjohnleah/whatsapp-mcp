package main

import (
	"context"
	"errors"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"net/http/httptest"
	"strings"
	"testing"
)

type profileFixture struct {
	members []types.GroupParticipant
	calls   int
	hidden  bool
}

func (f *profileFixture) GetGroupInfo(context.Context, types.JID) (*types.GroupInfo, error) {
	return &types.GroupInfo{Participants: f.members}, nil
}
func (f *profileFixture) GetProfilePictureInfo(_ context.Context, _ types.JID, p *whatsmeow.GetProfilePictureParams) (*types.ProfilePictureInfo, error) {
	f.calls++
	if p.CommonGID.Server != types.GroupServer {
		return nil, errors.New("no common group")
	}
	if f.hidden {
		return nil, whatsmeow.ErrProfilePictureUnauthorized
	}
	return &types.ProfilePictureInfo{URL: "https://pps.whatsapp.net/photo.jpg"}, nil
}
func TestGroupProfileScope(t *testing.T) {
	store := newTestMessageStore(t)
	defer store.db.Close()
	_, err := store.db.Exec("INSERT INTO messages(id,chat_jid,sender,content,is_from_me,timestamp) VALUES ('ask','123@g.us','447700900001','hello',0,'2026-10-30'),('self','123@g.us','447700900002','hello',1,'2026-10-30'),('private','other@g.us','447700900003','secret',0,'2026-10-30')")
	if err != nil {
		t.Fatal(err)
	}
	member := types.NewJID("447700900001", types.DefaultUserServer)
	f := &profileFixture{members: []types.GroupParticipant{{JID: types.NewJID("9988", types.HiddenUserServer), PhoneNumber: member}}}
	handler := groupProfileHandler(f, store)
	for _, tc := range []struct {
		body string
		code int
	}{
		{`{"chat_jid":"123@g.us","message_id":"ask"}`, 200},
		{`{"chat_jid":"123@g.us","message_id":"private"}`, 404},
		{`{"chat_jid":"123@g.us","message_id":"self"}`, 404},
		{`{"chat_jid":"447700900001@s.whatsapp.net","message_id":"ask"}`, 400},
		{`{"chat_jid":"123@g.us","message_id":"ask","jid":"private"}`, 400},
	} {
		rec := httptest.NewRecorder()
		handler(rec, httptest.NewRequest("POST", "/api/group-profile", strings.NewReader(tc.body)))
		if rec.Code != tc.code {
			t.Fatalf("status %d want %d: %s", rec.Code, tc.code, rec.Body.String())
		}
	}
	if f.calls != 1 {
		t.Fatalf("out-of-scope lookup reached profile API: %d", f.calls)
	}
	f.members = nil
	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest("POST", "/api/group-profile", strings.NewReader(`{"chat_jid":"123@g.us","message_id":"ask"}`)))
	if rec.Code != 403 || f.calls != 1 {
		t.Fatal("former member accepted")
	}
	f.members = []types.GroupParticipant{{JID: member}}
	f.hidden = true
	rec = httptest.NewRecorder()
	handler(rec, httptest.NewRequest("POST", "/api/group-profile", strings.NewReader(`{"chat_jid":"123@g.us","message_id":"ask"}`)))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"photo_available":false`) {
		t.Fatal("hidden photo not respected", rec.Body.String())
	}
}

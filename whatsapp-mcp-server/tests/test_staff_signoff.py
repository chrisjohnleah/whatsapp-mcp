import json

import staff_signoff


def test_replaces_legacy_face_signoff():
    out = staff_signoff.ensure_signed_message(
        "File is on the line.\n\n- Webby (Billy)",
        "- Webby (Luna - high)",
    )
    assert out.endswith("- Webby (Luna - high)")
    assert "- Webby (Billy)" not in out


def test_appends_when_unsigned():
    out = staff_signoff.ensure_signed_message("Done.", "- Hex (Grok 4.6 - low)")
    assert out == "Done.\n\n- Hex (Grok 4.6 - low)"


def test_leaves_empty_message_alone():
    assert staff_signoff.ensure_signed_message("", "- Hex (Grok 4.6 - low)") == ""
    assert staff_signoff.ensure_signed_message("   ", "- Hex (Grok 4.6 - low)") == "   "


def test_env_wins(monkeypatch):
    monkeypatch.setenv("HW_STAFF_SIGNOFF", "- Halo (Gemma 4 e4b)")
    assert staff_signoff.lookup_staff_signoff("120363@g.us") == "- Halo (Gemma 4 e4b)"
    out = staff_signoff.apply_staff_signoff("On it.", "120363@g.us")
    assert out.endswith("- Halo (Gemma 4 e4b)")


def test_state_file_by_jid(monkeypatch, tmp_path):
    path = tmp_path / "staff-signoff.json"
    path.write_text(
        json.dumps(
            {
                "120363412265017299@g.us": {
                    "signoff": "- Webby (Luna - high)",
                    "ts": 9999999999,
                }
            }
        ),
        encoding="utf-8",
    )
    monkeypatch.delenv("HW_STAFF_SIGNOFF", raising=False)
    monkeypatch.setenv("HW_STAFF_SIGNOFF_FILE", str(path))
    assert (
        staff_signoff.lookup_staff_signoff("120363412265017299@g.us")
        == "- Webby (Luna - high)"
    )
    assert staff_signoff.lookup_staff_signoff("other@g.us") == ""


def test_never_borrows_another_chats_signoff(monkeypatch, tmp_path):
    """A chat with no row of its own gets no stamp, never a neighbour's."""
    path = tmp_path / "staff-signoff.json"
    path.write_text(
        json.dumps(
            {
                "hex-chat@g.us": {"signoff": "- Hex (Sol 5.6 - high)", "ts": 9999999999},
                "webby-chat@g.us": {"signoff": "- Webby (Luna - high)", "ts": 9999999999},
            }
        ),
        encoding="utf-8",
    )
    monkeypatch.delenv("HW_STAFF_SIGNOFF", raising=False)
    monkeypatch.setenv("HW_STAFF_SIGNOFF_FILE", str(path))
    assert staff_signoff.lookup_staff_signoff("bbs-chat@g.us") == ""
    assert staff_signoff.apply_staff_signoff("On it.\n\n- Chris", "bbs-chat@g.us") == (
        "On it.\n\n- Chris"
    )
    assert staff_signoff.lookup_staff_signoff("hex-chat@g.us") == "- Hex (Sol 5.6 - high)"

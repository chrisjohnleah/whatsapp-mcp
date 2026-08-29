"""Stamp Happy Webs staff sign-off (face + model + effort) onto outbound text.

Chris 2026-08-28: every agent, every chat. Employee wakes export HW_STAFF_SIGNOFF
and write staff-signoff.json. send_message / send_file captions must apply it so
a model cannot ship `- Webby (Billy)` without the brain that actually wrote it.
"""
from __future__ import annotations

import json
import os
import re
import time
from pathlib import Path

_SIGN_LINE = re.compile(r"^[-—–]\s+\S")
_STATE_DEFAULT = (
    Path.home()
    / "Library/Application Support/happywebs-employees-runtime/staff-signoff.json"
)
_MAX_AGE_S = 6 * 3600


def _state_path() -> Path:
    override = os.getenv("HW_STAFF_SIGNOFF_FILE", "").strip()
    return Path(override) if override else _STATE_DEFAULT


def lookup_staff_signoff(recipient: str = "") -> str:
    env = os.getenv("HW_STAFF_SIGNOFF", "").strip()
    if env:
        return env
    path = _state_path()
    if not path.is_file():
        return ""
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return ""
    if not isinstance(data, dict):
        return ""
    now = time.time()
    recipient = (recipient or "").strip()
    row = data.get(recipient) if recipient else None
    if isinstance(row, dict):
        ts = float(row.get("ts") or 0)
        sign = str(row.get("signoff") or "").strip()
        if sign and now - ts <= _MAX_AGE_S:
            return sign
    # No row for this chat means no stamp. Borrowing another chat's sign-off
    # put "- Hex (Sol 5.6 - high)" on a BBS client message on 29 Aug 2026.
    return ""


def ensure_signed_message(message: str, signoff: str) -> str:
    msg = message or ""
    sign = (signoff or "").strip()
    if not sign or not msg.strip():
        return msg
    if sign in msg:
        return msg
    lines = msg.splitlines()
    while lines and lines[-1].strip() == "":
        lines.pop()
    if lines and _SIGN_LINE.match(lines[-1]):
        lines[-1] = sign
        return "\n".join(lines)
    body = "\n".join(lines) if lines else msg.strip()
    return f"{body}\n\n{sign}"


def apply_staff_signoff(message: str, recipient: str = "") -> str:
    return ensure_signed_message(message, lookup_staff_signoff(recipient))

#!/usr/bin/env python3
"""Minimal Python client that drives an AgentAPI++ server.

Usage::

    python examples/client-python.py http://localhost:3284 'summarise this repo'

Prints the agent's status, sends the user message, then prints the
last assistant message once the agent returns to ``stable``. Only
depends on the standard library.
"""

from __future__ import annotations

import json
import sys
import time
import urllib.error
import urllib.request
from typing import Any


def _request(method: str, url: str, body: dict[str, Any] | None = None) -> Any:
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(
        url,
        data=data,
        method=method,
        headers={"Content-Type": "application/json"},
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        raw = resp.read()
        return json.loads(raw) if raw else None


def main() -> int:
    if len(sys.argv) < 3:
        print("usage: client-python.py <base-url> <message>", file=sys.stderr)
        return 2

    base, prompt = sys.argv[1], sys.argv[2]
    status_url = f"{base.rstrip('/')}/status"
    message_url = f"{base.rstrip('/')}/message"
    messages_url = f"{base.rstrip('/')}/messages"

    status = _request("GET", status_url)
    print(f"agent={status.get('agentType')} status={status.get('status')}")

    try:
        _request("POST", message_url, {"type": "user", "content": prompt})
    except urllib.error.HTTPError as exc:
        print(f"send failed: {exc.code} {exc.reason}", file=sys.stderr)
        return 1

    deadline = time.time() + 120
    while time.time() < deadline:
        s = _request("GET", status_url)
        if s.get("status") == "stable":
            break
        time.sleep(0.5)
    else:
        print("timed out waiting for agent", file=sys.stderr)
        return 1

    messages = _request("GET", messages_url).get("messages", [])
    if messages:
        print("--- last message ---")
        print(messages[-1]["content"])
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

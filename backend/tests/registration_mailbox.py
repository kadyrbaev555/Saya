"""Complete signup against a disposable SMTP capture mailbox (Mailpit HTTP API)."""

import json
import os
import re
import time
import urllib.parse
import urllib.request


def register(post, credentials):
    email = credentials["email"]
    assert email.endswith("@example.test"), "Use a disposable test recipient"
    status, challenge = post("/auth/register", {
        "full_name": "Demo User", "phone": "+77000000001", **credentials,
    })
    assert status == 202, status
    mailbox = os.environ.get("SAYA_TEST_MAILPIT_URL", "http://127.0.0.1:8025")
    query = urllib.parse.urlencode({"query": "to:" + email})
    for _ in range(20):
        with urllib.request.urlopen(mailbox + "/api/v1/search?" + query, timeout=5) as response:
            messages = json.load(response)["messages"]
        if messages:
            with urllib.request.urlopen(mailbox + "/api/v1/message/" + messages[0]["ID"], timeout=5) as response:
                body = json.load(response)["Text"]
            match = re.search(r"(?<!\d)\d{6}(?!\d)", body)
            assert match, "Missing verification code in test email"
            return post("/auth/register/confirm", {
                "registration_id": challenge["registration_id"], "code": match.group(),
            })
        time.sleep(0.25)
    raise AssertionError("No registration email received in test mailbox")

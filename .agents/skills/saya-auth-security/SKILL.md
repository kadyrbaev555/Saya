---
name: saya-auth-security
description: Modify SAYA credentials, JWT, sessions and authorization.
---

Refresh rotation must lock session rows, preserve consumed hashes and revoke on replay. Never return ADMIN from public registration. Validate JWT algorithm, issuer, audience and expiry. Test concurrent refresh, replay, logout and invalid signatures.

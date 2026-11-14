---
name: saya-go-hexagonal
description: Implement Go business services within SAYA service boundaries.
---

Application owns ports; adapters implement them. Own database per service, parameterized sqlx, context deadlines. Test business invariants independently and persistence against PostgreSQL. Do not move use cases into Gateway.

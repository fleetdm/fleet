# 📦 `contract` Package

This package contains the **request and response structs** used by the HTTP API.

Keeping these in a separate package makes the code:
- **Easier to maintain** — the shape of API data is defined in one place
- **Clearer** — shows exactly what the API expects and returns
- **Reusable** — the same types can be used by handlers, tests, or clients

> This package should only define data structures — no business logic.

🔄 **Note:** Some request/response structs may still live in the server/service packages. Move them here as needed to keep API contracts organized and consistent.

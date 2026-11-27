# Security guidance for WorkTrack

- Every HTTP handler that changes data must go through `internal/application/authz`; never trust role or user ID from the request body — take it from the verified JWT in context.
- Project data isolation: any query on task, time_log, comment, board_column must filter by project membership of the current user.
- SQL only through sqlc-generated queries. No string concatenation into SQL.
- Passwords: bcrypt or argon2id. Activation and refresh tokens are stored hashed; compare with constant-time comparison.
- JWT: verify algorithm explicitly (reject `none` and unexpected algs), check `exp`. Refresh token only in HttpOnly + Secure + SameSite=Strict cookie. Access token never in localStorage.
- WebSocket: authenticate and check project membership before `websocket.Accept`; enforce message size limit.
- Never log passwords, tokens, API keys, or full request bodies of auth endpoints.
- Do not send email, phone, or passwords to the Gemini API. Treat Gemini output as untrusted input: validate JSON schema and numeric ranges.
- File uploads (avatar ≤ 5MB jpg/png, Excel/CSV import): check size and content type server-side; never use the client filename as a path.
- Secrets only from environment variables; `.env*` is never committed.

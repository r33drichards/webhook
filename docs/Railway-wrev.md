# wrev on Railway

Deploy this repository using its Dockerfile. Set `WREV_REVISION` to a wrev
commit that includes the WASI Preview 2 SQLite backend. Add a Railway volume
mounted at `/data` and run one replica. Set the health check to `/`.

| Railway variable | Value |
| --- | --- |
| `WREV_REVISION` | Build-time wrev commit |
| `WREV_DATA_DIR` | `/data` |
| `WREV_REPOSITORY_OWNER` | GitHub user or organization login |
| `WREV_LABEL` | Optional, defaults to `wrev` |
| `WREV_WEBHOOK_SECRET` | Random signing secret shared with GitHub |
| `WREV_CREDENTIAL_ADMIN_TOKEN` | Separate random bearer token, minimum 32 characters |
| `WREV_CREDENTIAL_ENCRYPTION_KEY` | Separate key of 32 random bytes encoded as 64 hex characters |
| `WREV_GITHUB_TOKEN` | GitHub token with read access to Contents and Pull requests |
| `OPENAI_MODEL` | Model slug from `wrev models` |

Configure a GitHub App webhook installed on all owned repositories, or add
repository webhooks individually. Send JSON **Pull requests** events to
`https://<domain>/github/webhook` with the same signing secret. Applying
`wrev` creates a queued session. Removing it cancels the queued or running
session. Reapplying it creates a new session. Identical GitHub deliveries are
deduplicated. Each review can run for up to ten minutes.

Use `PUT /admin/chatgpt-credentials` with `Authorization: Bearer <admin-token>`
to store a grant. The JSON body contains `access_token` and `expires_at` in Unix
seconds. Supply both `refresh_token` and the issued `client_id` to enable renewal.
A grant submitted without a refresh token remains usable until its expiry.
For initial testing, an environment `CHATGPT_ACCESS_TOKEN` works before a grant
is stored. `GET /admin/reviews` with the same authorization lists recent
sessions and their results.

The volume holds a durable queue in `queue.sqlite`, review checkpoints in
`sessions/<id>/review.sqlite`, and the encrypted ChatGPT grant in
`credentials.sqlite`. On restart, interrupted sessions resume from their last
checkpoint when the PR is still eligible. A checkpoint that records an
unfinished tool call requires a new session. Keep the encryption key available
across deployments and store backups of the volume privately.

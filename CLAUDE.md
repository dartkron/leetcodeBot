# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

- Run all tests: `go test -v ./...`
- Run a single package's tests: `go test -v ./internal/bot`
- Run a single test: `go test -v ./internal/bot -run TestProcessRequestBody`
- Coverage: `go test -coverprofile=c.out ./...`
- Static checks: `go vet ./...` and `staticcheck ./...` (CI pins staticcheck `2023.1.7`)
- Build bot entrypoint locally: `go build ./cmd/bot`
- Build reminder entrypoint locally: `go build ./cmd/reminder`

Go version note: `go.mod` declares `go 1.17`, but CI lint runs on 1.19 and the devcontainer uses 1.21. Don't bump the `go` directive without checking CI.

## Deployment

Both binaries are deployed as Yandex.Cloud serverless Functions by `.github/workflows/go.yml` on push to `main`. The workflow zips the repo per function (including shared `internal/` and `pkg/`) and runs `deploy/deploy.py`, which uses the Yandex.Cloud SDK to create a new function version and preserves the previous version's runtime, entrypoint, resources, and env vars. There is no separate staging environment — a push to `main` that passes `lint` and `tests` deploys to production.

Required secrets at deploy time: `SERVICE_ACCOUNT_KEY_ID`, `SERVICE_ACCOUNT_ID`, `SERVICE_ACCOUNT_PRIVATE_KEY`, `TARGET_FUNCTION_ID` (bot), `REMINDER_FUNCTION_ID` (reminder).

## Runtime environment variables

- `YDB_DATABASE`, `YDB_ENDPOINT` — Yandex YDB connection. If absent or unreachable, the bot falls back to the LeetCode GraphQL API for tasks; user subscribe/unsubscribe operations return `ErrNoActiveUsersStorage` because there is no user-storage fallback.
- `SENDING_TOKEN` — Telegram bot token used by the reminder function when POSTing to `api.telegram.org`.

## Architecture

Two serverless entrypoints share one application core:

- [cmd/bot/bot.go](cmd/bot/bot.go) — HTTP handler for Telegram webhooks. Reads the request body, hands it to `bot.Application.ProcessRequestBody`, writes the JSON response.
- [cmd/reminder/reminder.go](cmd/reminder/reminder.go) — Cron-triggered handler that calls `bot.Application.SendDailyTaskToSubscribedUsers` once per hour; it fans out Telegram sends in goroutines.

Both construct `bot.NewApplication(nil)`, which wires a `storage.YDBandFileCacheController` and a `leetcodeclient.LeetCodeGraphQlClient`.

### Package layout

- [internal/bot](internal/bot) — Telegram request/response types, command routing (`/getDailyTask`, `/Subscribe`, `/Unsubscribe`, hour-picker like `9:00`), keyboard generation, and the `Application` orchestrator. `GetTodayTaskFromAllPossibleSources` is the single read path for today's task: tries storage first, falls back to the LeetCode API, writes the result back through the storage controller.
- [internal/storage](internal/storage) — `Controller` interface plus `YDBandFileCacheController`, which composes two `tasksStorekeeper` layers (file cache → YDB) and one `usersStorekeeper` (YDB only). Task reads check the cache first and backfill it on miss; task writes go to both layers. User writes always go straight to YDB; there is no user cache. Sentinel errors (`ErrNoSuchTask`, `ErrNoSuchUser`, `ErrUserAlreadySubscribed`, `ErrUserAlreadyUnsubscribed`, `ErrNoActiveUsersStorage`, `ErrNoActiveTasksStorage`) are part of the public contract — callers in `internal/bot` switch on them.
- [internal/common](internal/common) — Domain types (`BotLeetCodeTask`, `User`, `CallbackData`) and `DateID` helpers. A `DateID` is a `uint64` in `YYYYMMDD` form produced by `GetDateID(time.Time)`; it is the primary key everywhere (file cache filename, YDB `dailyQuestion.id`, Telegram inline-keyboard callback payloads). The package also owns Telegram HTML sanitization (`FixTagsAndImages`, `RemoveUnsupportedTags`, `ReplaceImgTagWithA`) — LeetCode content contains tags Telegram rejects, so every task loaded from the API must be run through `FixTagsAndImages` before storage or display.
- [pkg/leetcodeclient](pkg/leetcodeclient) — GraphQL client for leetcode.com. `LeetCodeGraphQlClient` is split into a `graphQlRequester` transport (swappable for tests) and the two GraphQL queries (`dailyCodingQuestionRecords`, `GetQuestion`). `GetDailyTask` = slug lookup + question fetch.
- [tests/mocks](tests/mocks), [pkg/leetcodeclient/mocks](pkg/leetcodeclient/mocks) — testify-style mocks for the HTTP transport and LeetCode client. `tests.ErrBypassTest` is the conventional sentinel for forcing an error path through a mock.

### File-cache layer

`fileCache` writes to `/tmp/task_<dateID>.cache`. On Yandex.Cloud Functions `/tmp` persists only for the lifetime of a single warm container, so the cache is effectively per-instance. This is intentional — YDB is the shared store; the file cache just avoids re-querying YDB on hot instances.

### YDB schema

Two tables, `dailyQuestion` and `users`, are expected to exist (see README.md). `users.sendingHour` is the hour-of-day (UTC) the reminder function uses to select recipients; `GetSubscribedUsers` filters by `subscribed = true AND sendingHour = $hour`.

## Telegram contract notes

- All outgoing messages use `parse_mode: HTML`, so content must be HTML-safe and use only Telegram's supported tag subset — go through `common.RemoveUnsupportedTags` / `FixTagsAndImages` rather than emitting tags directly.
- Callback payloads are JSON-encoded `common.CallbackData`; the keyboard builder in `common.GetInlineKeyboard` and the callback router in `Application.processCallback` must stay in sync on the `CallbackType` values (`HintReuqest`, `DifficultyRequest`).

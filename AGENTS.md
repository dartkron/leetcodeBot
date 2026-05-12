# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## Commands

- Run all tests: `go test -v ./...`
- Run a single package's tests: `go test -v ./internal/bot`
- Run a single test: `go test -v ./internal/bot -run TestProcessRequestBody`
- Coverage: `go test -coverprofile=c.out ./...`
- Static checks: `go vet ./...` and `staticcheck ./...` (CI pins staticcheck `2024.1.1`)
- Build bot entrypoint locally: `go build ./cmd/bot`
- Build reminder entrypoint locally: `go build ./cmd/reminder`

Go version note: `go.mod`, CI, the Yandex runtime, and the devcontainer are currently aligned on Go 1.23 (`golang123` in Yandex.Cloud). Do not change the `go` directive or Yandex runtime without checking `.github/workflows/go.yml`, `.devcontainer/devcontainer.json`, and `deploy/deploy.py` together.

## Deployment

Both binaries are deployed as Yandex.Cloud serverless Functions by `.github/workflows/go.yml` on push to `main`. The workflow builds two zip archives, puts the selected entrypoint (`cmd/bot/bot.go` or `cmd/reminder/reminder.go`) at the archive root, includes shared `internal/` and `pkg/`, excludes `*_test.go` files, and runs `deploy/deploy.py`.

`deploy/deploy.py` uses the Yandex.Cloud SDK to create a new function version. It preserves the previous version's entrypoint, resources, execution timeout, service account, and environment variables, sets runtime `golang123`, and adds/updates `GIT_VERSION` from `GITHUB_REF` and `GITHUB_SHA`. There is no separate staging environment — a push to `main` that passes `lint` and `tests` deploys to production. The `sonarcloud` job is separate and is not a deploy dependency.

Required secrets at deploy time: `SERVICE_ACCOUNT_KEY_ID`, `SERVICE_ACCOUNT_ID`, `SERVICE_ACCOUNT_PRIVATE_KEY`, `TARGET_FUNCTION_ID` (bot), `REMINDER_FUNCTION_ID` (reminder). SonarCloud also needs `SONAR_TOKEN` for the quality job.

## Runtime environment variables

- `YDB_DATABASE`, `YDB_ENDPOINT` — Yandex YDB connection. If absent or unreachable, task reads can fall back to the LeetCode GraphQL API; user subscribe/unsubscribe operations return `ErrNoActiveUsersStorage` because there is no user-storage fallback.
- `SENDING_TOKEN` — Telegram bot token used by the reminder function when POSTing to `api.telegram.org`. If it is empty, the send URL becomes `https://api.telegram.org/bot/sendMessage`, which is useful in tests but not in production.

## Architecture

Two serverless entrypoints share one application core:

- [cmd/bot/bot.go](cmd/bot/bot.go) — HTTP handler for Telegram webhooks. Reads the request body, creates a 5-second context, hands it to `bot.Application.ProcessRequestBody`, and writes a JSON Telegram response.
- [cmd/reminder/reminder.go](cmd/reminder/reminder.go) — Cron-triggered handler that calls `bot.Application.SendDailyTaskToSubscribedUsers`; it returns a small Yandex.Function response object and propagates errors as status 500.

Both construct `bot.NewApplication(nil)`, which wires a `storage.YDBandFileCacheController`, a `leetcodeclient.LeetCodeGraphQlClient`, and a default `http.Client`. Tests often construct `Application` directly inside the same package to inject mocks into its private fields.

### Package layout

- [internal/bot](internal/bot) — Telegram request/response types, command routing (`/getDailyTask`, `/Subscribe`, `/Unsubscribe`, hour-picker text like `9:00`), keyboard generation, and the `Application` orchestrator. `GetTodayTaskFromAllPossibleSources` is the single read path for today's task: tries storage first, falls back to the LeetCode API, sanitizes API content, writes the result through the storage controller, and returns it.
- [internal/storage](internal/storage) — `Controller` interface plus `YDBandFileCacheController`, which composes two `tasksStorekeeper` layers (file cache → YDB) and one `usersStorekeeper` (YDB only). Task reads check the cache first and backfill it on miss; task writes go to cache and DB, but cache write failures are logged and tolerated. User writes always go straight to YDB; there is no user cache. Sentinel errors (`ErrNoSuchTask`, `ErrNoSuchUser`, `ErrUserAlreadySubscribed`, `ErrUserAlreadyUnsubscribed`, `ErrNoActiveUsersStorage`, `ErrNoActiveTasksStorage`) are part of the public contract — callers in `internal/bot` and tests switch on them.
- [internal/common](internal/common) — Domain types (`BotLeetCodeTask`, `User`, `CallbackData`) and `DateID` helpers. A `DateID` is a `uint64` in `YYYYMMDD` form produced by `GetDateID(time.Time)`; it is the primary key everywhere (file cache filename, YDB `dailyQuestion.id`, Telegram inline-keyboard callback payloads). The package also owns Telegram HTML sanitization (`FixTagsAndImages`, `RemoveUnsupportedTags`, `ReplaceImgTagWithA`) — LeetCode content contains tags Telegram rejects, so every task loaded from the API must be run through `FixTagsAndImages` before storage or display.
- [pkg/leetcodeclient](pkg/leetcodeclient) — GraphQL client for leetcode.com. `LeetCodeGraphQlClient` is split into a `graphQlRequester` transport (swappable for tests) and two GraphQL queries (`dailyCodingQuestionRecords`, `GetQuestion`). `GetDailyTask` = slug lookup + question fetch.
- [tests/mocks](tests/mocks), [pkg/leetcodeclient/mocks](pkg/leetcodeclient/mocks) — testify-style mocks for the HTTP transport and LeetCode client. `tests.ErrBypassTest` is the conventional sentinel for forcing an error path through a mock.

### File-cache layer

`fileCache` writes to `/tmp/task_<dateID>.cache`. On Yandex.Cloud Functions `/tmp` persists only for the lifetime of a single warm container, so the cache is effectively per-instance. This is intentional — YDB is the shared store; the file cache just avoids re-querying YDB on hot instances.

### YDB schema

Two tables, `dailyQuestion` and `users`, are expected to exist (see README.md). `users.sendingHour` is the hour-of-day (UTC) the reminder function uses to select recipients; `GetSubscribedUsers` filters by `subscribed = true AND sendingHour = $hour`.

## Telegram contract notes

- All outgoing messages use `parse_mode: HTML`, so content must be HTML-safe and use only Telegram's supported tag subset — go through `common.RemoveUnsupportedTags` / `FixTagsAndImages` rather than emitting tags directly.
- User-visible commands are exact string matches. `/Subscribe 7` is not supported; `/Subscribe` only opens the hour keyboard, and the subsequent `H:00` text triggers subscription.
- Main keyboard replies are JSON-encoded into `TelegramResponse.ReplyMarkup`; task hint/difficulty buttons use Telegram inline keyboards.
- Callback payloads are JSON-encoded `common.CallbackData`; the keyboard builder in `common.GetInlineKeyboard` and the callback router in `Application.processCallback` must stay in sync on the `CallbackType` values (`HintRequest`, `DifficultyRequest`).
- Callback handling intentionally reads only from storage and does not fall back to the LeetCode API. That prevents arbitrary callback payloads from making the app fetch many historical tasks.
- `SendDailyTaskToSubscribedUsers` selects users by the current UTC hour and sends messages concurrently. Individual send failures are logged after three attempts and do not fail the whole reminder run.

### Time and IDs

`common.GetDateInRightTimeZone` currently returns `time.Now().UTC()`, and reminders also use `time.Now().UTC()` for the subscription hour. The LeetCode JSON date parser uses `America/Los_Angeles` for API date fields, but the app's daily task key is UTC-based. Be careful when changing date logic: `DateID` must remain consistent across storage, cache filenames, callbacks, and tests.

### YDB layer

The YDB query executer is a package-level singleton initialized with `sync.Once`, and the connection is also initialized once. It reads `YDB_ENDPOINT` and `YDB_DATABASE` on first connection creation, so tests that stub connection behavior should do it before the first real connection attempt.

Task persistence stores hints as a JSON string in YDB and maps difficulty through numeric values (`Easy` = 0, `Medium` = 1, `Hard` = 2, unknown = 3). Changing these mappings requires a storage migration or compatibility layer.

### LeetCode API notes

The GraphQL transport posts to `https://leetcode.com/graphql` with a browser-like `user-agent` and a currently hardcoded `x-csrftoken` (marked `FIXME` in code). If LeetCode starts rejecting requests, check [pkg/leetcodeclient/http.go](pkg/leetcodeclient/http.go) first.

The HTTP requester currently reads the response body without checking non-2xx status codes or closing the response body explicitly. Existing tests focus on transport errors and JSON parsing; add coverage before changing this behavior.

### Testing notes

- The test suite relies heavily on call journals and exact JSON strings. Small text, keyboard, or field-order changes can require test updates even when behavior is intentionally changed.
- `tests.ErrBypassTest` is used to force error branches through mocks. Keep using it for new tests to match the existing style.
- Tests live beside packages (`internal/bot`, `internal/storage`, `pkg/leetcodeclient`) and sometimes access unexported fields by using the same package name instead of `_test`.

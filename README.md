# Telegram bot for Leetcode daily tasks
I found it useful for me to see actual daily task in Telegram. Also want to have notificator about daily task =)

Currently status:
## Bot running in Yandex.Cloud Functions (serverless)
Since there is generous [free tier](https://cloud.yandex.com/en/docs/billing/concepts/serverless-free-tier) and it's stable enough.

## Can use YDB for caching.
Expect following tables structure:
```(sql)
CREATE TABLE `dailyQuestion`
(
    `id` Int64,
    `questionId` Int64,
    `text` String,
    `title` String,
    PRIMARY KEY (`id`)
);

CREATE TABLE `users`
(
    `id` Uint64,
    `chat_id` Uint64,
    `firstName` String,
    `lastName` String,
    `subscribed` Bool,
    `username` String,
    PRIMARY KEY (`id`)
);
```

Awaits `YDB_DATABASE` and `YDB_ENDPOINT` environment variables.
__If database connection will fail or variables will not set, bot will connect LeetCode API for question data.__

## Features
1. Can send you today task in response on any request.
2. Can send task hints if they isset.
3. Subscribe/Unsubscribe buttons/commands.
4. Once per day reminder serverless function send new task to all users who subscribed. Reminder require `SENDING_TOKEN` environment variable with Telegram API token.
And it's all on the current stage.

Plan to add:
1. Possibly get authorization token from subscriber, to show progress, may be hints.

## Using Leetcode graphQl api
Only recieving of daily tasks coded.

## Continuous delivery
Here you can see very basic example of continuous delivery to two Yandex.Cloud functions.
See `.github/workflows/deploy.yml` for details.
# Telegram bot for Leetcode daily tasks
I found it useful for me to see actual daily task in Telegram. Also want to have notificator about daily task =)

Currently status:
## Bot running in Yandex.Cloud Functions (serverless)
Since there is generous [free tier](https://cloud.yandex.com/en/docs/billing/concepts/serverless-free-tier) and it's stable enough.

## Can use YDB for caching.
Expect table with following structure:
```(sql)
CREATE TABLE `dailyQuestion`
(
    `id` Int64,
    `questionId` Int64,
    `text` String,
    `title` String,
    PRIMARY KEY (`id`)
);
```

Awaits `YDB_DATABASE` and `YDB_ENDPOINT` environment variables.
__If database connection will fail or variables will not set, bot will connect LeetCode API for question data.__

## Can send you today task in response on any request
And it's all on current stage.
Plan to add:
1. Notification when new task is appeared.
2. Possible hints show via button.
3. Buttons for subscribe/unsubscribe.
4. Possibly get authorization token from subscriber, to show progress, may be hints.

## Using Leetcode graphQl api
Only recieving of daily tasks coded.
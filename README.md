[![Maintainability](https://api.codeclimate.com/v1/badges/371cc101c86797eb2e70/maintainability)](https://codeclimate.com/github/dartkron/leetcodeBot/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/371cc101c86797eb2e70/test_coverage)](https://codeclimate.com/github/dartkron/leetcodeBot/test_coverage)
# Telegram bot for Leetcode daily tasks
I've found it useful for me to see actual daily task in Telegram. Also want to have notificator about daily task =)

Currently status:
## Bot running in Yandex.Cloud Functions (serverless)
Since there is generous [free tier](https://cloud.yandex.com/en/docs/billing/concepts/serverless-free-tier) and it's stable enough.

## Can use YDB for caching.
Expect following tables structure:
```sql
CREATE TABLE `dailyQuestion`
(
    `id` Uint64,
    `content` String,
    `difficulty` Uint8,
    `hints` String,
    `questionId` Uint64,
    `title` String,
    `titleSlug` String,
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
1. Can reply with today task.
2. Can send task hints if they are set.
3. Can send task difficulty.
3. Subscribe/Unsubscribe user buttons/commands.
4. Once per day reminder serverless function send new task to all users who subscribed. Reminder require `SENDING_TOKEN` environment variable with Telegram API token.
And it's all on the current stage.

Plan to add:
1. Possibly get authorization token from subscriber, to show progress, may be show suggestions =).

## Using Leetcode graphQl API
Most likely it's a proof of concept. Only recieving of tasks if coded for now.
Example:
```go
import (
	"context"
	"fmt"
	"time"

	"github.com/dartkron/leetcodeBot/v2/pkg/leetcodeclient"
)

func main() {
	leetcodeClient := leetcodeclient.NewLeetCodeGraphQlClient()
	loc, _ := time.LoadLocation("US/Pacific")
	now := time.Date(2021, time.September, 26, 0, 0, 0, 0, loc)
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*time.Duration(5))
	defer cancelFunc()
	task, err := leetcodeClient.GetDailyTask(ctx, now)
	if err != nil {
		fmt.Println("Got error from LeetcodeClient", err)
	}
	fmt.Println(task)
}
```

## Continuous delivery
Here you can see very basic example of continuous delivery to two Yandex.Cloud functions.
See `.github/workflows/deploy.yml` for details.

## Python version
Version 1.0 was made with Python, you can find it in versions history.
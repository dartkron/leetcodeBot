import json

from datetime import datetime
from pytz import timezone

from lib import leetcode_api
from lib import bot_ydb
from lib.common import BotLeetCodeTask, addTaskLinkToContent, fixTagsAndImages, getTaskId


def getFixedTaskForDate(targetDate: datetime) -> BotLeetCodeTask:
    task = BotLeetCodeTask(getTaskId(targetDate))
    task.fromLeetCodeTask(leetcode_api.getTaskOfTheDay(targetDate))
    task = fixTagsAndImages(task)
    task = addTaskLinkToContent(task)
    return task

def main() -> None:
    # Uncomment for today task
    #targetDate = datetime.now(tz=timezone('US/Pacific'))
    targetDate = datetime(year=2021, month=2, day=14)
    print(f'Going to retrieve and print daily task for {targetDate}')
    print(getFixedTaskForDate(targetDate))


def handler(event, _):
    body = json.loads(event['body']) if event['body'] else {'message': {'chat': {'id': 0}}}
    targetDate = datetime.now(tz=timezone('US/Pacific'))
    storedTask = bot_ydb.getTaskOfTheDay(targetDate)
    if storedTask[0]:
        task = storedTask[1]
    else:
        task = getFixedTaskForDate(targetDate)
        bot_ydb.saveTaskOfTheDay(task)

    return {
        'statusCode': 200,
        'headers': {
            'Content-Type': 'application/json'
        },
        'body': json.dumps({
            'method': 'sendMessage',
            'chat_id': body['message']['chat']['id'],
            'text':  f'<strong>{task.Title}</strong>\n\n{task.Content}',
            'parse_mode': 'HTML',
        }),
        'isBase64Encoded': False
    }


if __name__ == '__main__':
    main()

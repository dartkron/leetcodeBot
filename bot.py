import json

from datetime import datetime
from pytz import timezone

from lib import leetcode_api
from lib import bot_ydb
from lib.user_actions import startAction, stopAction
from lib.common import BotLeetCodeTask, addTaskLinkToContent, fixTagsAndImages, getTaskId


HELP_MESAGE = '''
You command "{command}" isn't recognized =(
List of available commands:
/dailyTask — get actual dailyTask
/start — start automatically sending of daily tasks
/stop — stop automatically sending of daily tasks
'''


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


def getDailyTask() -> BotLeetCodeTask:
    targetDate = datetime.now(tz=timezone('US/Pacific'))
    storedTask = bot_ydb.getTaskOfTheDay(targetDate)
    if storedTask[0]:
        task = storedTask[1]
    else:
        task = getFixedTaskForDate(targetDate)
        bot_ydb.saveTaskOfTheDay(task)
    return task


def handler(event, _):
    if event['body']:
        body = json.loads(event['body'])
    else:
        # For test purposes ?
        body = {'message': {'chat': {'id': 0}, 'text': 'dailyTask'}}

    command = body['message']['text']

    response = HELP_MESAGE.format(command=command)

    if command == '/dailyTask':
        task = getDailyTask()
        response = f'<strong>{task.Title}</strong>\n\n{task.Content}'
    elif command == '/start':
        response = startAction(body)
    elif command == '/stop':
        response = stopAction(body)

    return {
        'statusCode': 200,
        'headers': {
            'Content-Type': 'application/json'
        },
        'body': json.dumps({
            'method': 'sendMessage',
            'chat_id': body['message']['chat']['id'],
            'text':  response,
            'parse_mode': 'HTML',
        }),
        'isBase64Encoded': False
    }


if __name__ == '__main__':
    main()

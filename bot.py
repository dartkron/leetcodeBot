import json
from typing import Dict, Any

from datetime import datetime
from pytz import timezone

from lib import leetcode_api
from lib import bot_ydb
from lib.user_actions import startAction, stopAction
from lib.common import BotLeetCodeTask, fixTagsAndImages, getTaskId


HELP_MESAGE = '''
You command "{command}" isn't recognized =(
List of available commands:
/getDailyTask — get actual dailyTask
/Subscribe — start automatically sending of daily tasks
/Unsubscribe — stop automatically sending of daily tasks
'''


def getFixedTaskForDate(targetDate: datetime) -> BotLeetCodeTask:
    task = BotLeetCodeTask(getTaskId(targetDate))
    task.fromLeetCodeTask(leetcode_api.getTaskOfTheDay(targetDate))
    task = fixTagsAndImages(task)
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


def serveCallback(responseBody: Dict[str, Any]) -> Dict[str,str]:
    request = json.loads(responseBody['callback_query']['data'])
    response = {
        'chat_id': responseBody['callback_query']['from']['id'],
    }
    exists, task = bot_ydb.getTaskById(request['dateId'])
    if not exists:
        response['text'] = 'There is not such dailyTask. Try another breach ;)'
        return response

    if request['hint'] > len(task.Hints) - 1:
        response['text'] = f'There is no such hint for task {task.DateId}'
        return response
    response['text'] = 'Hint #' + str(request['hint'] + 1) + ': ' + task.Hints[request['hint']]

    return response


def serveMessage(responseBody: Dict[str, Any]) -> Dict[str, str]:
    command = responseBody['message']['text']
    response = {
        'chat_id': responseBody['message']['chat']['id'],
        'text': HELP_MESAGE.format(command=command),
        'reply_markup': json.dumps({
            'keyboard': [
                [{'text': 'getDailyTask'}],
                [{'text': 'Subscribe'}, {'text': 'Unsubscribe'}],
            ],
            'input_field_placeholder': 'Please, use buttons below:',
            'resize_keyboard': True,
        })
    }

    if command in ('getDailyTask', '/getDailyTask'):
        task = getDailyTask()
        response['text'] = f'<strong>{task.Title}</strong>\n\n{task.Content}'
        response['reply_markup'] = task.generateHintsInlineKeyboard()
    elif command in ('Subscribe', '/Subscribe'):
        response['text'] = startAction(responseBody)
    elif command in ('Unsubscribe', '/Unsubscribe'):
        response['text'] = stopAction(responseBody)

    return response

def handler(event, _):
    if event['body']:
        body = json.loads(event['body'])
    else:
        # For test purposes ?
        body = {'message': {'chat': {'id': 0}, 'text': 'dailyTask'}}

    bodyTemplate = {
        'method': 'sendMessage',
        'parse_mode': 'HTML',
    }

    if 'callback_query' in body:
        response = serveCallback(body)
    else:
        response = serveMessage(body)

    for key, value in response.items():
        bodyTemplate[key] = value

    return {
        'statusCode': 200,
        'headers': {
            'Content-Type': 'application/json'
        },
        'body': json.dumps(bodyTemplate),
        'isBase64Encoded': False
    }


if __name__ == '__main__':
    main()

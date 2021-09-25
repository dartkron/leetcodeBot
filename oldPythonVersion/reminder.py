import logging
import json
from lib import bot_ydb
from lib.tasks_actions import getDailyTask
from lib.common import sendMessage


FORMAT = '%(asctime)-15s %(levelname)s %(message)s'
logging.basicConfig(format=FORMAT, level=logging.ERROR)
logging.info('Hello world')


def handler(event, _):
    subscribedUsers = bot_ydb.getSubscribedUsersIds()
    dailyTask = getDailyTask()

    bodyTemplate = {
        'method': 'sendMessage',
        'parse_mode': 'HTML',
    }

    bodyTemplate['text'] = dailyTask.getTaskText()
    bodyTemplate['reply_markup'] = dailyTask.generateHintsInlineKeyboard()
    for userId in subscribedUsers:
        bodyTemplate['chat_id'] = userId
        if not sendMessage(bodyTemplate):
            logging.error('Unable to send message to user %s', userId)

    return {
        'statusCode': 200,
        'body': json.dumps({
            'event': event,
        }),
    }




def main():
    print(json.dumps(handler('','')))


if __name__ == '__main__':
    main()

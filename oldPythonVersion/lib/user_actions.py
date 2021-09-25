from typing import Dict, Any

import lib.bot_ydb as bot_ydb
from lib.common import User


def getOrCreateUser(responseBody: Dict[str, Any]) -> User:
    exist, user = bot_ydb.getUser(responseBody['message']['from']['id'])
    if not exist:
        user.ChatId = responseBody['message']['chat']['id']
        user.Username = responseBody['message']['from']['username']
        user.FirstName = responseBody['message']['from']['first_name']
        user.LastName = responseBody['message']['from']['last_name']
        user.Subscribed = False
        bot_ydb.saveUser(user)
    return user


def startAction(responseBody: Dict[str, Any]) -> str:
    user = getOrCreateUser(responseBody)
    if user.Subscribed:
        return f'{user.FirstName}, you have <strong>already subscribed</strong> for daily updates, nothing to do.'
    bot_ydb.subscribeUser(user)
    return f'{user.FirstName}, you have <strong>sucessfully subscribed</strong>. You\'ll automatically recieve daily tasks when they appear.'


def stopAction(responseBody: Dict[str, Any]) -> str:
    user = getOrCreateUser(responseBody)
    if not user.Subscribed:
        return f'{user.FirstName}, you were <strong>not subscribed</strong>. No additonal actions required.'
    bot_ydb.unsubscribeUser(user)
    return f'''
 {user.FirstName}, you have <strong>sucessfully unsubscribed</strong>. You'll not automatically recieve daily tasks.
If you've found this bot useless and have ideas of possible improvements, please, add them to https://github.com/dartkron/leetcodeBot/issues
    '''

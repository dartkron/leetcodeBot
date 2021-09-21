import logging
from typing import List
from datetime import datetime
import os
import json
import requests

from lib.leetcode_api import LeetCodeTask

SENDING_URL = 'https://api.telegram.org/bot{token}/sendMessage'


class Error(Exception):
    pass


class SendMessageError(Error):
    pass


class BotLeetCodeTask(LeetCodeTask):
    def __init__(self, dateId: int = 0, questionId: int = 0, title: str = '', content: str = '', hints: List[str] = []) -> None:
        self.DateId = dateId
        LeetCodeTask.__init__(self, questionId, title, content, hints)

    def fromLeetCodeTask(self, task: LeetCodeTask):
        LeetCodeTask.__init__(self, task.QuestionId, task.Title, task.Content, task.Hints)

    def __repr__(self) -> str:
        return f'DateId: {self.DateId}\n' + LeetCodeTask.__repr__(self)

    def getTaskText(self) -> str:
        return f'<strong>{self.Title}</strong>\n\n{self.Content}'

    def generateHintsInlineKeyboard(self) -> str:
        listOfHints = []
        level = []
        for i in range(len(self.Hints)):
            level.append({'text': f'Hint {i + 1}', 'callback_data': json.dumps({'dateId': self.DateId, 'hint': i})})
            if len(level) == 5:
                listOfHints.append(level.copy())
                level = []
        if level:
            listOfHints.append(level.copy())
            level = []
        listOfHints = [[{'text': 'See task on LeetCode website', 'url': f'https://leetcode.com/explore/item/{self.QuestionId}'}]] + listOfHints
        return json.dumps({'inline_keyboard': listOfHints})

    def toJsonStr(self) -> str:
        return json.dumps({
            'dateId': self.DateId,
            'questionId': self.QuestionId,
            'title': self.Title,
            'content': self.Content,
            'hints': self.Hints,
        })


class User:
    def __init__(self, userId: int = 0, chatId: int = 0, username: str = '', firstName: str = '', lastName: str = '', subscribed: bool = False):
        self.UserId = userId
        self.ChatId = chatId
        self.Username = username
        self.FirstName = firstName
        self.LastName = lastName
        self.Subscribed = subscribed


def removeUnsupportedTags(s: str) -> str:
    for tag in ('<p>', '</p>','<ul>', '</ul>', '</li>', '</sup>', '<em>', '</em>', '\n\n', '<br>', '</br>'):
        s = s.replace(tag, '')
    s = s.replace('&nbsp;', ' ')
    s = s.replace('<sup>', '**')
    s = s.replace('<sub>', '(')
    s = s.replace('</sub>', ')')
    s = s.replace('<li>', ' — ')
    return s


def replaceImgWithA(s: str) -> str:
    for i in range(s.count('<img')):
        start = s.find('<img')
        end = start + s[start:].find('/>') + 2
        url_start = s[start:end].find('src=') + 5
        url_end = s[start + url_start + 1:end].find('"')
        url = s[start + url_start:start + url_start + 1 + url_end]
        s = s[:start] + f'\n<a href="{url}">Picture {i}</a>' + s[end:]
    return s


def getTaskId(date: datetime) -> int:
    return date.year * 10**4 + date.month * 10**2 + date.day


def fixTagsAndImages(task: BotLeetCodeTask) -> BotLeetCodeTask:
    task.Title = removeUnsupportedTags(task.Title)
    task.Content = replaceImgWithA(removeUnsupportedTags(task.Content))
    for i in range(len(task.Hints)):
        task.Hints[i] = replaceImgWithA(removeUnsupportedTags(task.Hints[i]))
    return task


def sendMessage(body: str) -> bool:
    @retry(times=3, exceptions=Exception)
    def sendMessageImpl():
        r = requests.post(SENDING_URL.format(token=os.getenv('SENDING_TOKEN')), headers = {'Content-Type': 'application/json'}, json=body)
        if r.status_code != 200 or r.json()['ok'] != True:
            raise SendMessageError(f'Status code: {r.status_code}, response: {r.json()}')

        return True
    try:
        sendMessageImpl()
    except:
        return False
    return True


def retry(times, exceptions):
    """
    Retry Decorator
    Retries the wrapped function/method `times` times if the exceptions listed
    in ``exceptions`` are thrown
    :param times: The number of times to repeat the wrapped function/method
    :type times: Int
    :param Exceptions: Lists of exceptions that trigger a retry attempt
    :type Exceptions: Tuple of Exceptions
    """
    def decorator(func):
        def newfn(*args, **kwargs):
            attempt = 0
            while attempt < times:
                try:
                    return func(*args, **kwargs)
                except exceptions:
                    print(
                        'Exception thrown when attempting to run %s, attempt '
                        '%d of %d' % (func, attempt, times)
                    )
                    attempt += 1
            return func(*args, **kwargs)
        return newfn
    return decorator
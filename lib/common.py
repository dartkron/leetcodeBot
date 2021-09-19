from typing import List
from datetime import datetime
import json

from lib.leetcode_api import LeetCodeTask



class BotLeetCodeTask(LeetCodeTask):
    def __init__(self, dateId: int = 0, questionId: int = 0, title: str = '', content: str = '', hints: List[str] = []) -> None:
        self.DateId = dateId
        LeetCodeTask.__init__(self, questionId, title, content, hints)

    def fromLeetCodeTask(self, task: LeetCodeTask):
        LeetCodeTask.__init__(self, task.QuestionId, task.Title, task.Content, task.Hints)

    def __repr__(self) -> str:
        return f'DateId: {self.DateId}\n' + LeetCodeTask.__repr__(self)

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

from datetime import datetime

from lib.leetcode_api import LeetCodeTask


class BotLeetCodeTask(LeetCodeTask):
    def __init__(self, dateId: int = 0, questionId: int = 0, title: str = '', content: str = '') -> None:
        self.DateId = dateId
        LeetCodeTask.__init__(self, questionId, title, content)

    def fromLeetCodeTask(self, task: LeetCodeTask):
        LeetCodeTask.__init__(self, task.QuestionId, task.Title, task.Content)

    def __repr__(self) -> str:
        return f'DateId: {self.DateId}\n' + LeetCodeTask.__repr__(self)


def removeUnsupportedTags(s: str) -> str:
    for tag in ('<p>', '</p>','<ul>', '</ul>', '</li>', '</sup>', '<em>', '</em>', '\n\n'):
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
    return task


def addTaskLinkToContent(task: BotLeetCodeTask) -> BotLeetCodeTask:
    task.Content += f'\n\n\n<strong>Link to task:</strong> https://leetcode.com/explore/item/{task.QuestionId}'
    return task

from datetime import datetime
from typing import Dict, Any, List

import requests


class LeetCodeTask: # pylint: disable=too-few-public-methods
    def __init__(self, questionId: int = 0, itemId: int = 0, title: str = '', content: str = '', hints: List[str] = []) -> None:
        self.QuestionId = questionId
        self.ItemId = itemId
        self.Title = title
        self.Content = content
        self.Hints = hints

    def __repr__(self) -> str:
        return f'QuesetionId: {self.QuestionId}\nItemId: {self.ItemId}\nTitle: {self.Title}\nContent: {self.Content}\nHints: {self.Hints}'


MONTH_NAMES = {
    1: 'january',
    2: 'february',
    3: 'march',
    4: 'april',
    5: 'may',
    6: 'june',
    7: 'july',
    8: 'august',
    9: 'september',
    10: 'october',
    11: 'november',
    12: 'december'
}


GRAPHQL_URL = 'https://leetcode.com/graphql'


GET_CHAPTERS: Dict[str, Any] = {
    "operationName": "GetChaptersWithItems",
    "variables": {
        "cardSlug": ""
    },
    "query": "query GetChaptersWithItems($cardSlug: String!) { chapters(cardSlug: $cardSlug) { items {id title type }}}",
}


GET_SLUG: Dict[str, Any] = {
    "operationName": "GetItem",
    "variables": {
        "itemId": ""
    },
    "query": "query GetItem($itemId: String!) {item(id: $itemId) { question { questionId title titleSlug }}}"
}


GET_QUESTION: Dict[str, Any] = {
    "operationName": "GetQuestion",
    "variables": {
        "titleSlug": ""
    },
    "query": "query GetQuestion($titleSlug: String!) {question(titleSlug: $titleSlug) { questionId questionTitle categoryTitle submitUrl content urlManager hints}}"
}


def requestGraphQl(json_param: Dict[Any, Any]):
    return requests.get(url=GRAPHQL_URL, json=json_param)


class Error(Exception):
    pass

class NonPredictableError(Error):
    pass


def getDailyTaskId(targetDate: datetime) -> str:
    GET_CHAPTERS['variables']['cardSlug'] = generateCardSlug(targetDate)
    r = requestGraphQl(GET_CHAPTERS)
    js = r.json()
    cnt = 0
    item = 0
    for chapter in range(len(js['data']['chapters'])):
        if cnt + len(js['data']['chapters'][chapter]['items']) - 1 >= targetDate.day: # pylint: disable=no-else-break
            item = targetDate.day - cnt
            break
        else:
            cnt += len(js['data']['chapters'][chapter]['items']) - 1
    if item == 0:
        raise NonPredictableError('Not enought days in leetcode Json ?!')

    return js['data']['chapters'][chapter]['items'][item]['id']


def getQestionSlug(itemId: str) -> str:
    GET_SLUG['variables']['itemId'] = itemId
    r = requestGraphQl(GET_SLUG)
    return r.json()['data']['item']['question']['titleSlug']


def getQuestionDetails(slugId: str):
    GET_QUESTION['variables']['titleSlug'] = slugId
    r = requestGraphQl(GET_QUESTION)
    return r.json()['data']['question']


def generateCardSlug(date: datetime) -> str:
    return f'{MONTH_NAMES[date.month]}-leetcoding-challenge-{date.year}'


def getTaskOfTheDay(targetDate: datetime) -> LeetCodeTask:
    itemId = getDailyTaskId(targetDate)
    s = getQestionSlug(itemId)
    question = getQuestionDetails(s)
    return LeetCodeTask(int(question['questionId']), int(itemId), question['questionTitle'], question['content'], question['hints'])

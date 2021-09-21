import os
from datetime import datetime
from typing import Tuple

import json

from pytz import timezone

from lib import bot_ydb,leetcode_api
from lib.common import BotLeetCodeTask, fixTagsAndImages, getTaskId

CACHE_FILE_NAME = '/tmp/task_{taskId}.cache' # nosec

def getFromCache(taskId: int) -> Tuple[bool, BotLeetCodeTask]:
    fileName = CACHE_FILE_NAME.format(taskId=taskId)
    if os.path.isfile(fileName):
        with open(fileName) as cacheFile:
            task = loadTaskFromJsonStr(cacheFile.read())
    else:
        task = BotLeetCodeTask(taskId)
    return (bool(task.QuestionId), task)


def saveToCache(task: BotLeetCodeTask) -> None:
    fileName = CACHE_FILE_NAME.format(taskId=task.DateId)
    if not os.path.exists(fileName):
        with open(fileName) as cacheFile:
            cacheFile.write(task.toJsonStr())
    return None


def loadTaskFromJsonStr(jsonStr: str) -> BotLeetCodeTask:
        jsonTask = json.loads(jsonStr)
        return BotLeetCodeTask(jsonTask['dateId'], jsonTask['questionId'], jsonTask['title'], jsonTask['content'], jsonTask['hints'])


def getFixedTaskForDate(targetDate: datetime) -> BotLeetCodeTask:
    taskId = getTaskId(targetDate)

    existsInCache, task = getFromCache(taskId)
    if existsInCache:
        return task

    task.fromLeetCodeTask(leetcode_api.getTaskOfTheDay(targetDate))
    task = fixTagsAndImages(task)
    saveToCache(task)
    return task


def getDailyTask() -> BotLeetCodeTask:
    targetDate = datetime.now(tz=timezone('US/Pacific'))
    storedTask = bot_ydb.getTaskOfTheDay(targetDate)
    if storedTask[0]:
        task = storedTask[1]
    else:
        task = getFixedTaskForDate(targetDate)
        bot_ydb.saveTaskOfTheDay(task)
    return task
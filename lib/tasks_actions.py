from datetime import datetime
from pytz import timezone

from lib import bot_ydb,leetcode_api
from lib.common import BotLeetCodeTask, fixTagsAndImages, getTaskId


def getFixedTaskForDate(targetDate: datetime) -> BotLeetCodeTask:
    task = BotLeetCodeTask(getTaskId(targetDate))
    task.fromLeetCodeTask(leetcode_api.getTaskOfTheDay(targetDate))
    task = fixTagsAndImages(task)
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
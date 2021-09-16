import json
import os

from typing import Tuple
from datetime import datetime

import ydb

from lib.common import BotLeetCodeTask, getTaskId


DATABASE_CONNECTED = True

if not os.getenv('YDB_ENDPOINT') or not os.getenv('YDB_DATABASE'):
    print('Environment variables "YDB_ENDPOINT" and/or "YDB_DATABASE" are not set.\nDatabase functions will do nothing!')
    DATABASE_CONNECTED = False
else:
    try:
        # create driver in global space.
        driver_config = ydb.DriverConfig(
            os.getenv('YDB_ENDPOINT'), os.getenv('YDB_DATABASE'),
            credentials=ydb.construct_credentials_from_environ(),
            root_certificates=ydb.load_ydb_root_certificate(),
        )
        driver = ydb.Driver(driver_config)
        # Wait for the driver to become active for requests.
        driver.wait(fail_fast=True, timeout=5)
        # Create the session pool instance to manage YDB sessions.
        pool = ydb.SessionPool(driver)
    except Exception as e: # pylint: disable=broad-except
        print(f'Error on database module inititalization:\n{e}\nAll database operations will be omitted')
        DATABASE_CONNECTED = False


def getTaskOfTheDay(targetDate: datetime) -> Tuple[bool, BotLeetCodeTask]:
    if not DATABASE_CONNECTED:
        return (False, BotLeetCodeTask())
    dateId = getTaskId(targetDate)
    def execute_select_query(session):
        # create the transaction and execute query.
        return session.transaction().execute(
            f'select title, text, questionId from dailyQuestion where id = {dateId};',
            commit_tx=True,
            settings=ydb.BaseRequestSettings().with_timeout(3).with_operation_timeout(2)
        )
    result = pool.retry_operation_sync(execute_select_query)[0].rows
    return (bool(len(result)), BotLeetCodeTask(dateId, result[0].questionId, result[0].title.decode('UTF-8'), result[0].text.decode('UTF-8')) if len(result)  else BotLeetCodeTask())


def saveTaskOfTheDay(task: BotLeetCodeTask) -> None:
    if not DATABASE_CONNECTED:
        return None
    t64 = json.dumps(task.Title)
    c64 = json.dumps(task.Content)

    def do_insert(session):
        return session.transaction().execute(
            f'replace into dailyQuestion (id, questionId, title, text) VALUES ({task.DateId}, {task.QuestionId},{t64}, {c64});',
            commit_tx=True,
            settings=ydb.BaseRequestSettings().with_timeout(3).with_operation_timeout(2)
        )
    return pool.retry_operation_sync(do_insert)

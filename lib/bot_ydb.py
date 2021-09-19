import os
import json
from time import sleep

from typing import Tuple
from datetime import datetime

import ydb

from lib.common import BotLeetCodeTask, getTaskId, User


DATABASE_CONNECTED = True

if not os.getenv('YDB_ENDPOINT') or not os.getenv('YDB_DATABASE'):
    print('Environment variables "YDB_ENDPOINT" and/or "YDB_DATABASE" are not set.\nDatabase functions will do nothing!')
    DATABASE_CONNECTED = False
else:
    for i in range(3):
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
            DATABASE_CONNECTED = True
        except Exception as e: # pylint: disable=broad-except
            print(f'Error on database module inititalization:\n{e}\nRetry in 100ms')
            DATABASE_CONNECTED = False
            sleep(0.1)
            continue
        break
    if not DATABASE_CONNECTED:
        print('Connection to database failed. All database operations will do nothing =(')


def getTaskOfTheDay(targetDate: datetime) -> Tuple[bool, BotLeetCodeTask]:
    if not DATABASE_CONNECTED:
        return (False, BotLeetCodeTask())
    return getTaskById(getTaskId(targetDate))


def getTaskById(dateId: int) -> Tuple[bool, BotLeetCodeTask]:
    def execute_select_query(session):
        # create the transaction and execute query.
        query = '''
        DECLARE $dateId AS Int64;

        SELECT title, text, questionId, hints
        FROM dailyQuestion
        WHERE id = $dateId;
        '''
        prepared_query = session.prepare(query)
        return session.transaction(ydb.SerializableReadWrite()).execute(
            prepared_query, {
                '$dateId': dateId,
            },
            commit_tx=True,
            settings=ydb.BaseRequestSettings().with_timeout(3).with_operation_timeout(2)
        )

    result = pool.retry_operation_sync(execute_select_query)[0].rows
    return (
        bool(len(result)),
        BotLeetCodeTask(
            dateId,
            result[0].questionId,
            result[0].title.decode('UTF-8'),
            result[0].text.decode('UTF-8'),
            json.loads(result[0].hints.decode('UTF-8')),
        ) if len(result) else BotLeetCodeTask(dateId)
    )


def saveTaskOfTheDay(task: BotLeetCodeTask) -> None:
    if not DATABASE_CONNECTED:
        return None

    def do_insert(session):
        query = '''
        DECLARE $dateId AS Int64;
        DECLARE $questionId AS Int64;
        DECLARE $title AS String;
        DECLARE $content AS String;
        DECLARE $hints AS String;

        REPLACE INTO dailyQuestion (id, questionId, title, text, hints)
        VALUES ($dateId, $questionId, $title, $content, $hints);
        '''
        prepared_query = session.prepare(query)
        return session.transaction(ydb.SerializableReadWrite()).execute(
            prepared_query, {
                '$dateId': task.DateId,
                '$questionId': task.QuestionId,
                '$title': task.Title.encode('UTF-8'),
                '$content': task.Content.encode('UTF-8'),
                '$hints': json.dumps(task.Hints).encode('UTF-8'),
            },
            commit_tx=True,
            settings=ydb.BaseRequestSettings().with_timeout(3).with_operation_timeout(2)
        )
    return pool.retry_operation_sync(do_insert)


def getUser(userId: int) -> Tuple[bool, User]:
    if not DATABASE_CONNECTED:
        return (False, User())

    def execute_select_query(session):
        # create the transaction and execute query.
        query = '''
        DECLARE $userId AS Uint64;

        SELECT chat_id, firstName, lastName, username, subscribed
        FROM users
        WHERE id = $userId;
        '''
        prepared_query = session.prepare(query)
        return session.transaction(ydb.SerializableReadWrite()).execute(
            prepared_query, {
                '$userId': userId,
            },
            commit_tx=True,
            settings=ydb.BaseRequestSettings().with_timeout(3).with_operation_timeout(2)
        )

    result = pool.retry_operation_sync(execute_select_query)[0].rows
    return (
        bool(len(result)),
        User(
            userId,
            result[0].chat_id,
            result[0].username.decode('UTF-8'),
            result[0].firstName.decode('UTF-8'),
            result[0].lastName.decode('UTF-8'),
            result[0].subscribed) if len(result) else User(userId)
    )


def saveUser(user: User) -> None:
    if not DATABASE_CONNECTED:
        return None

    def do_insert(session):
        # create the transaction and execute query.
        query = '''
        DECLARE $id AS Uint64;
        DECLARE $chat_id AS Uint64;
        DECLARE $firstname AS String;
        DECLARE $lastname AS String;
        DECLARE $username AS String;
        DECLARE $subscribed AS Bool;

        REPLACE INTO users (id, chat_id, firstName, lastName, username, subscribed)
        VALUES ($id, $chat_id, $firstname, $lastname, $username, $subscribed);
        '''
        prepared_query = session.prepare(query)
        return session.transaction(ydb.SerializableReadWrite()).execute(
            prepared_query, {
                '$id': user.UserId,
                '$chat_id': user.ChatId,
                '$username': user.Username.encode('UTF-8'),
                '$firstname': user.FirstName.encode('UTF-8'),
                '$lastname': user.LastName.encode('UTF-8'),
                '$subscribed': user.Subscribed,
            },
            commit_tx=True,
            settings=ydb.BaseRequestSettings().with_timeout(3).with_operation_timeout(2)
        )

    return pool.retry_operation_sync(do_insert)


def subscribeUser(user: User) -> None:
    if not DATABASE_CONNECTED:
        return None

    def do_update(session):
        # create the transaction and execute query.
        query = '''
        DECLARE $userId AS Uint64;

        UPDATE `users` set subscribed = true
        WHERE id=$userId;
        '''
        prepared_query = session.prepare(query)
        return session.transaction(ydb.SerializableReadWrite()).execute(
            prepared_query, {
                '$userId': user.UserId,
            },
            commit_tx=True,
            settings=ydb.BaseRequestSettings().with_timeout(3).with_operation_timeout(2)
        )

    return pool.retry_operation_sync(do_update)


def unsubscribeUser(user: User) -> None:
    if not DATABASE_CONNECTED:
        return None

    def do_update(session):
        # create the transaction and execute query.
        query = '''
        DECLARE $userId AS Uint64;

        UPDATE `users` set subscribed = false
        WHERE id=$userId;
        '''
        prepared_query = session.prepare(query)
        return session.transaction(ydb.SerializableReadWrite()).execute(
            prepared_query, {
                '$userId': user.UserId,
            },
            commit_tx=True,
            settings=ydb.BaseRequestSettings().with_timeout(3).with_operation_timeout(2)
        )

    return pool.retry_operation_sync(do_update)
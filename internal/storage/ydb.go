package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dartkron/leetcodeBot/v2/internal/common"
	"github.com/yandex-cloud/ydb-go-sdk/v2"
	"github.com/yandex-cloud/ydb-go-sdk/v2/connect"
	"github.com/yandex-cloud/ydb-go-sdk/v2/table"
)

const (
	getTaskQuery = `
	DECLARE $dateId AS Uint64;

	SELECT title, text, questionId, itemId, hints, difficulty
	FROM dailyQuestion
	WHERE id = $dateId;
	`
	replaceTaskQuery = `
	DECLARE $dateId AS Uint64;
	DECLARE $questionId AS Uint64;
	DECLARE $itemId AS Uint64;
	DECLARE $title AS String;
	DECLARE $content AS String;
	DECLARE $hints AS String;
	DECLARE $difficulty AS Uint8;

	REPLACE INTO dailyQuestion (id, questionId, itemId, title, text, hints, difficulty)
	VALUES ($dateId, $questionId, $itemId, $title, $content, $hints, $difficulty);
	`
	getUserQuery = `
	DECLARE $userId AS Uint64;

	SELECT chat_id, firstName, lastName, username, subscribed
	FROM users
	WHERE id = $userId;
	`
	getSubscribedUsersQuery = `
	SELECT id, chat_id, firstName, lastName, username
	FROM users
	WHERE subscribed = true;
	`
	saveUserQuery = `
	DECLARE $id AS Uint64;
	DECLARE $chat_id AS Uint64;
	DECLARE $firstname AS String;
	DECLARE $lastname AS String;
	DECLARE $username AS String;
	DECLARE $subscribed AS Bool;

	REPLACE INTO users (id, chat_id, firstName, lastName, username, subscribed)
	VALUES ($id, $chat_id, $firstname, $lastname, $username, $subscribed);
	`
	subscribeUserQuery = `
	DECLARE $userId AS Uint64;

    UPDATE users set subscribed = true
    WHERE id=$userId;
	`
	unsubscribeUserQuery = `
	DECLARE $userId AS Uint64;

    UPDATE users set subscribed = false
    WHERE id=$userId;
	`
)

type ydbStorage struct {
	txc *table.TransactionControl
}

func newYdbStorage() *ydbStorage {
	return &ydbStorage{
		txc: table.TxControl(
			table.BeginTx(table.WithSerializableReadWrite()),
			table.CommitTx(),
		),
	}
}

func (y *ydbStorage) getTask(dateID uint64) (common.BotLeetCodeTask, error) {
	ctx := context.Background()
	connectCtx, cancelFunc := context.WithTimeout(ctx, time.Second)
	defer cancelFunc()
	connection, err := getConnection(connectCtx)
	if err != nil {
		return common.BotLeetCodeTask{}, err
	}
	defer connection.Close()

	var res *table.Result
	err = table.Retry(
		ctx,
		connection.Table().Pool(),
		table.OperationFunc(func(ctx context.Context, s *table.Session) (err error) {
			_, res, err = s.Execute(ctx, y.txc,
				getTaskQuery,
				table.NewQueryParameters(
					table.ValueParam("$dateId", ydb.Uint64Value(dateID)),
				),
			)
			return err
		}),
	)
	if err != nil {
		return common.BotLeetCodeTask{}, err
	}

	if res.RowCount() == 0 {
		return common.BotLeetCodeTask{}, ErrNoSuchTask
	}

	var (
		title      *string
		text       *string
		questionID *uint64
		itemID     *uint64
		hints      *string
		difficulty *uint8
	)

	returnValue := common.BotLeetCodeTask{DateID: dateID}

	for res.NextResultSet(ctx, "title", "text", "questionId", "itemId", "hints", "difficulty") {
		for res.NextRow() {
			err := res.Scan(
				&title,
				&text,
				&questionID,
				&itemID,
				&hints,
				&difficulty,
			)
			if err != nil {
				return common.BotLeetCodeTask{}, err
			}
			returnValue.Title = *title
			returnValue.Content = *text
			returnValue.QuestionID = *questionID
			returnValue.ItemID = *itemID
			returnValue.SetDifficultyFromNum(*difficulty)
			err = json.Unmarshal([]byte(*hints), &returnValue.Hints)
			if err != nil {
				return returnValue, err
			}
		}
	}
	if res.Err() != nil {
		return common.BotLeetCodeTask{}, res.Err()
	}
	return returnValue, nil
}

func (y *ydbStorage) saveTask(task common.BotLeetCodeTask) error {
	ctx := context.Background()
	connectCtx, cancelFunc := context.WithTimeout(ctx, time.Second)
	defer cancelFunc()
	connection, err := getConnection(connectCtx)
	if err != nil {
		return err
	}
	defer connection.Close()

	marshalledHints, err := json.Marshal(task.Hints)
	if err != nil {
		return err
	}

	err = table.Retry(
		ctx,
		connection.Table().Pool(),
		table.OperationFunc(func(ctx context.Context, s *table.Session) (err error) {
			_, _, err = s.Execute(ctx, y.txc,
				replaceTaskQuery,
				table.NewQueryParameters(
					table.ValueParam("$dateId", ydb.Uint64Value(task.DateID)),
					table.ValueParam("$questionId", ydb.Uint64Value(task.QuestionID)),
					table.ValueParam("$itemId", ydb.Uint64Value(task.ItemID)),
					table.ValueParam("$title", ydb.StringValue([]byte(task.Title))),
					table.ValueParam("$content", ydb.StringValue([]byte(task.Content))),
					table.ValueParam("$hints", ydb.StringValue(marshalledHints)),
					table.ValueParam("$difficulty", ydb.Uint8Value(task.GetDifficultyNum())),
				),
			)
			return err
		}),
	)
	return err
}

func (y *ydbStorage) getUser(userID uint64) (common.User, error) {
	ctx := context.Background()
	connectCtx, cancelFunc := context.WithTimeout(ctx, time.Second)
	defer cancelFunc()
	connection, err := getConnection(connectCtx)
	if err != nil {
		return common.User{}, err
	}
	defer connection.Close()

	var res *table.Result
	err = table.Retry(
		ctx,
		connection.Table().Pool(),
		table.OperationFunc(func(ctx context.Context, s *table.Session) (err error) {
			_, res, err = s.Execute(ctx, y.txc,
				getUserQuery,
				table.NewQueryParameters(
					table.ValueParam("$userId", ydb.Uint64Value(userID)),
				),
			)
			return err
		}),
	)
	if err != nil {
		return common.User{}, err
	}

	if res.RowCount() == 0 {
		return common.User{}, ErrNoSuchUser
	}

	var (
		chatID     *uint64
		username   *string
		firstName  *string
		lastName   *string
		subscribed *bool
	)

	returnValue := common.User{ID: userID}

	for res.NextResultSet(ctx, "chat_id", "firstName", "lastName", "username", "subscribed") {
		for res.NextRow() {
			err := res.Scan(
				&chatID,
				&username,
				&firstName,
				&lastName,
				&subscribed,
			)
			if err != nil {
				return common.User{}, err
			}
			returnValue.ChatID = *chatID
			returnValue.Username = *username
			returnValue.FirstName = *firstName
			returnValue.LastName = *lastName
			returnValue.Subscribed = *subscribed
		}
	}
	if res.Err() != nil {
		return common.User{}, res.Err()
	}
	return returnValue, nil
}

func (y *ydbStorage) getSubscribedUsers() ([]common.User, error) {
	ctx := context.Background()
	connectCtx, cancelFunc := context.WithTimeout(ctx, time.Second)
	defer cancelFunc()
	connection, err := getConnection(connectCtx)
	if err != nil {
		return []common.User{}, err
	}
	defer connection.Close()

	var res *table.Result
	err = table.Retry(
		ctx,
		connection.Table().Pool(),
		table.OperationFunc(func(ctx context.Context, s *table.Session) (err error) {
			_, res, err = s.Execute(ctx, y.txc,
				getSubscribedUsersQuery,
				table.NewQueryParameters(),
			)
			return err
		}),
	)
	if err != nil {
		return []common.User{}, err
	}

	var (
		userID    *uint64
		chatID    *uint64
		username  *string
		firstName *string
		lastName  *string
	)

	returnValue := []common.User{}

	for res.NextResultSet(ctx, "id", "chat_id", "firstName", "lastName", "username") {
		for res.NextRow() {
			err := res.Scan(
				&userID,
				&chatID,
				&username,
				&firstName,
				&lastName,
			)
			if err != nil {
				return []common.User{}, err
			}
			returnValue = append(returnValue, common.User{
				ID:         *userID,
				ChatID:     *chatID,
				Username:   *username,
				FirstName:  *firstName,
				LastName:   *lastName,
				Subscribed: true,
			})

		}
	}
	if res.Err() != nil {
		return []common.User{}, res.Err()
	}
	return returnValue, nil
}

func (y *ydbStorage) saveUser(user common.User) error {
	ctx := context.Background()
	connectCtx, cancelFunc := context.WithTimeout(ctx, time.Second)
	defer cancelFunc()
	connection, err := getConnection(connectCtx)
	if err != nil {
		return err
	}
	defer connection.Close()

	err = table.Retry(
		ctx,
		connection.Table().Pool(),
		table.OperationFunc(func(ctx context.Context, s *table.Session) (err error) {
			_, _, err = s.Execute(ctx, y.txc,
				saveUserQuery,
				table.NewQueryParameters(
					table.ValueParam("$id", ydb.Uint64Value(user.ID)),
					table.ValueParam("$chat_id", ydb.Uint64Value(user.ChatID)),
					table.ValueParam("$firstname", ydb.StringValue([]byte(user.FirstName))),
					table.ValueParam("$lastname", ydb.StringValue([]byte(user.LastName))),
					table.ValueParam("$username", ydb.StringValue([]byte(user.Username))),
					table.ValueParam("$subscribed", ydb.BoolValue(user.Subscribed)),
				),
			)
			return err
		}),
	)
	return err
}

func (y *ydbStorage) subscribeUser(userID uint64) error {
	ctx := context.Background()
	connectCtx, cancelFunc := context.WithTimeout(ctx, time.Second)
	defer cancelFunc()
	connection, err := getConnection(connectCtx)
	if err != nil {
		return err
	}
	defer connection.Close()

	err = table.Retry(
		ctx,
		connection.Table().Pool(),
		table.OperationFunc(func(ctx context.Context, s *table.Session) (err error) {
			_, _, err = s.Execute(ctx, y.txc,
				subscribeUserQuery,
				table.NewQueryParameters(
					table.ValueParam("$userId", ydb.Uint64Value(userID)),
				),
			)
			return err
		}),
	)
	return err
}

func (y *ydbStorage) unsubscribeUser(userID uint64) error {
	ctx := context.Background()
	connectCtx, cancelFunc := context.WithTimeout(ctx, time.Second)
	defer cancelFunc()
	connection, err := getConnection(connectCtx)
	if err != nil {
		return err
	}
	defer connection.Close()

	err = table.Retry(
		ctx,
		connection.Table().Pool(),
		table.OperationFunc(func(ctx context.Context, s *table.Session) (err error) {
			_, _, err = s.Execute(ctx, y.txc,
				unsubscribeUserQuery,
				table.NewQueryParameters(
					table.ValueParam("$userId", ydb.Uint64Value(userID)),
				),
			)
			return err
		}),
	)
	return err
}

func getConnection(connectCtx context.Context) (*connect.Connection, error) {
	return connect.New(
		connectCtx,
		connect.MustConnectionString(
			fmt.Sprintf("%s/?database=%s", os.Getenv("YDB_ENDPOINT"), os.Getenv("YDB_DATABASE")),
		),
	)
}

package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/dartkron/leetcodeBot/v2/internal/common"
	"github.com/yandex-cloud/ydb-go-sdk/v2"
	"github.com/yandex-cloud/ydb-go-sdk/v2/connect"
	"github.com/yandex-cloud/ydb-go-sdk/v2/table"
)

const (
	getTaskQuery = `
	DECLARE $dateId AS Uint64;

	SELECT title, content, questionId, titleSlug, hints, difficulty
	FROM dailyQuestion
	WHERE id = $dateId;
	`
	replaceTaskQuery = `
	DECLARE $dateId AS Uint64;
	DECLARE $questionId AS Uint64;
	DECLARE $titleSlug AS String;
	DECLARE $title AS String;
	DECLARE $content AS String;
	DECLARE $hints AS String;
	DECLARE $difficulty AS Uint8;

	REPLACE INTO dailyQuestion (id, questionId, titleSlug, title, content, hints, difficulty)
	VALUES ($dateId, $questionId, $titleSlug, $title, $content, $hints, $difficulty);
	`
	getUserQuery = `
	DECLARE $id AS Uint64;

	SELECT chat_id, firstName, lastName, username, subscribed, sendingHour
	FROM users
	WHERE id = $id;
	`
	getSubscribedUsersQuery = `
	DECLARE $sendingHour AS Uint8;
	SELECT id, chat_id, firstName, lastName, username
	FROM users
	WHERE subscribed = true and sendingHour = $sendingHour;
	`
	saveUserQuery = `
	DECLARE $id AS Uint64;
	DECLARE $chat_id AS Uint64;
	DECLARE $firstname AS String;
	DECLARE $lastname AS String;
	DECLARE $username AS String;
	DECLARE $subscribed AS Bool;
	DECLARE $sendingHour AS Uint8;

	REPLACE INTO users (id, chat_id, firstName, lastName, username, subscribed, sendingHour)
	VALUES ($id, $chat_id, $firstname, $lastname, $username, $subscribed, $sendingHour);
	`
	subscribeUserQuery = `
	DECLARE $id AS Uint64;
	DECLARE $sendingHour AS Uint8;

    UPDATE users set subscribed = true, sendingHour = $sendingHour
    WHERE id=$id;
	`
	unsubscribeUserQuery = `
	DECLARE $id AS Uint64;

    UPDATE users set subscribed = false
    WHERE id=$id;
	`
)

// YDBResult IMO is what supposed to be a part of ydb package. Interface to allow YDB response mocks
type YDBResult interface {
	RowCount() int
	NextResultSet(context.Context, ...string) bool
	NextRow() bool
	Err() error
	Scan(values ...interface{}) error
}

type queryExecuter interface {
	ProcessQuery(context.Context, string, *table.QueryParameters) (YDBResult, error)
}

type ydbQueryExecuter struct {
	ExecQueryFunc      func(context.Context, table.SessionProvider, table.Operation) error
	GetConnectionFunc  func(context.Context, connect.ConnectParams, ...connect.ConnectOption) (*connect.Connection, error)
	txc                *table.TransactionControl
	connection         *connect.Connection
	connectionInitOnce sync.Once
}

type ydbStorage struct {
	ydbExecuter queryExecuter
}

var initializedExecuter *ydbQueryExecuter
var executerInitOnce sync.Once

func newYDBQueryExecuter() *ydbQueryExecuter {
	executerInitOnce.Do(func() {
		initializedExecuter = &ydbQueryExecuter{
			txc: table.TxControl(
				table.BeginTx(table.WithSerializableReadWrite()),
				table.CommitTx(),
			),
			ExecQueryFunc:     table.Retry,
			GetConnectionFunc: connect.New,
		}
	})
	return initializedExecuter
}

func newYdbStorage() *ydbStorage {
	ydbExecuter := newYDBQueryExecuter()
	return &ydbStorage{
		ydbExecuter: ydbExecuter,
	}
}

func (y *ydbQueryExecuter) getConnection() (*connect.Connection, error) {
	var err error
	y.connectionInitOnce.Do(func() {
		y.connection, err = y.GetConnectionFunc(
			context.TODO(),
			connect.MustConnectionString(
				fmt.Sprintf("%s/?database=%s", os.Getenv("YDB_ENDPOINT"), os.Getenv("YDB_DATABASE")),
			),
		)
	})
	if err != nil {
		return nil, err
	}
	if y.connection == nil {
		return nil, ErrNoActiveTasksStorage
	}

	return y.connection, nil
}

func (y *ydbQueryExecuter) ProcessQuery(ctx context.Context, query string, queryParams *table.QueryParameters) (YDBResult, error) {
	finishChan := make(chan error)
	var connection *connect.Connection
	var err error

	go func() {
		connection, err = y.getConnection()
		finishChan <- err
	}()

	select {
	case <-ctx.Done():
		err = common.ErrClosedContext
	case err = <-finishChan:
	}

	if err != nil {
		return nil, err
	}
	var res *table.Result
	err = y.ExecQueryFunc(
		ctx,
		connection.Table().Pool(),
		table.OperationFunc(func(ctx context.Context, s *table.Session) (err error) {
			_, res, err = s.Execute(ctx, y.txc, query, queryParams)
			return err
		}),
	)
	return res, err
}

func (y *ydbStorage) getTask(ctx context.Context, dateID uint64) (common.BotLeetCodeTask, error) {
	res, err := y.ydbExecuter.ProcessQuery(ctx, getTaskQuery, table.NewQueryParameters(
		table.ValueParam("$dateId", ydb.Uint64Value(dateID)),
	))
	if err != nil {
		return common.BotLeetCodeTask{}, err
	}

	if res.RowCount() == 0 {
		return common.BotLeetCodeTask{}, ErrNoSuchTask
	}

	var (
		title      *string
		content    *string
		questionID *uint64
		titleSlug  *string
		hints      *string
		difficulty *uint8
	)

	returnValue := common.BotLeetCodeTask{DateID: dateID}

	for res.NextResultSet(ctx, "title", "content", "questionId", "titleSlug", "hints", "difficulty") {
		for res.NextRow() {
			err = res.Scan(
				&title,
				&content,
				&questionID,
				&titleSlug,
				&hints,
				&difficulty,
			)
			if err != nil {
				break
			}
			fmt.Println("Before asignment value is", title)
			returnValue.Title = *title
			returnValue.Content = *content
			returnValue.QuestionID = *questionID
			returnValue.TitleSlug = *titleSlug
			returnValue.SetDifficultyFromNum(*difficulty)
			err = json.Unmarshal([]byte(*hints), &returnValue.Hints)
			if err != nil {
				break
			}
		}
		if err != nil {
			return common.BotLeetCodeTask{}, err
		}
	}
	return returnValue, res.Err()
}

func (y *ydbStorage) saveTask(ctx context.Context, task common.BotLeetCodeTask) error {
	marshalledHints, err := json.Marshal(task.Hints)
	if err != nil {
		return err
	}
	_, err = y.ydbExecuter.ProcessQuery(ctx, replaceTaskQuery, table.NewQueryParameters(
		table.ValueParam("$dateId", ydb.Uint64Value(task.DateID)),
		table.ValueParam("$questionId", ydb.Uint64Value(task.QuestionID)),
		table.ValueParam("$titleSlug", ydb.StringValue([]byte(task.TitleSlug))),
		table.ValueParam("$title", ydb.StringValue([]byte(task.Title))),
		table.ValueParam("$content", ydb.StringValue([]byte(task.Content))),
		table.ValueParam("$hints", ydb.StringValue(marshalledHints)),
		table.ValueParam("$difficulty", ydb.Uint8Value(task.GetDifficultyNum())),
	),
	)
	return err
}

func (y *ydbStorage) getUser(ctx context.Context, userID uint64) (common.User, error) {
	res, err := y.ydbExecuter.ProcessQuery(ctx, getUserQuery,
		table.NewQueryParameters(
			table.ValueParam("$id", ydb.Uint64Value(userID)),
		),
	)
	if err != nil {
		return common.User{}, err
	}

	if res.RowCount() == 0 {
		return common.User{}, ErrNoSuchUser
	}

	var (
		chatID      *uint64
		username    *string
		firstName   *string
		lastName    *string
		subscribed  *bool
		sendingHour *uint8
	)

	returnValue := common.User{ID: userID}

	for res.NextResultSet(ctx, "chat_id", "firstName", "lastName", "username", "subscribed", "sendingHour") {
		for res.NextRow() {
			err := res.Scan(
				&chatID,
				&firstName,
				&lastName,
				&username,
				&subscribed,
				&sendingHour,
			)
			if err != nil {
				return common.User{}, err
			}
			returnValue.ChatID = *chatID
			returnValue.Username = *username
			returnValue.FirstName = *firstName
			returnValue.LastName = *lastName
			returnValue.Subscribed = *subscribed
			returnValue.SendingHour = *sendingHour
		}
	}
	return returnValue, res.Err()
}

func (y *ydbStorage) getSubscribedUsers(ctx context.Context, sendingHour uint8) ([]common.User, error) {
	res, err := y.ydbExecuter.ProcessQuery(ctx, getSubscribedUsersQuery, table.NewQueryParameters(table.ValueParam("$sendingHour", ydb.Uint8Value(sendingHour))))
	if err != nil {
		return []common.User{}, err
	}

	var (
		id        *uint64
		chatID    *uint64
		username  *string
		firstName *string
		lastName  *string
	)
	returnValue := []common.User{}

	for res.NextResultSet(ctx, "id", "chat_id", "firstName", "lastName", "username") {
		for res.NextRow() {
			err := res.Scan(
				&id,
				&chatID,
				&firstName,
				&lastName,
				&username,
			)
			if err != nil {
				return []common.User{}, err
			}
			returnValue = append(returnValue, common.User{
				ID:          *id,
				ChatID:      *chatID,
				Username:    *username,
				FirstName:   *firstName,
				LastName:    *lastName,
				Subscribed:  true,
				SendingHour: sendingHour,
			})

		}
	}
	return returnValue, res.Err()
}

func (y *ydbStorage) saveUser(ctx context.Context, user common.User) error {
	_, err := y.ydbExecuter.ProcessQuery(ctx, saveUserQuery, table.NewQueryParameters(
		table.ValueParam("$id", ydb.Uint64Value(user.ID)),
		table.ValueParam("$chat_id", ydb.Uint64Value(user.ChatID)),
		table.ValueParam("$firstname", ydb.StringValue([]byte(user.FirstName))),
		table.ValueParam("$lastname", ydb.StringValue([]byte(user.LastName))),
		table.ValueParam("$username", ydb.StringValue([]byte(user.Username))),
		table.ValueParam("$subscribed", ydb.BoolValue(user.Subscribed)),
		table.ValueParam("$sendingHour", ydb.Uint8Value(user.SendingHour)),
	),
	)
	return err
}

func (y *ydbStorage) subscribeUser(ctx context.Context, userID uint64, sendingHour uint8) error {
	_, err := y.ydbExecuter.ProcessQuery(ctx, subscribeUserQuery, table.NewQueryParameters(
		table.ValueParam("$id", ydb.Uint64Value(userID)),
		table.ValueParam("$sendingHour", ydb.Uint8Value(sendingHour)),
	),
	)
	return err
}

func (y *ydbStorage) unsubscribeUser(ctx context.Context, userID uint64) error {
	_, err := y.ydbExecuter.ProcessQuery(ctx, unsubscribeUserQuery, table.NewQueryParameters(
		table.ValueParam("$id", ydb.Uint64Value(userID)),
	),
	)
	return err
}

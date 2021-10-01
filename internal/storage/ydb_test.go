package storage

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/dartkron/leetcodeBot/v2/internal/common"
	"github.com/dartkron/leetcodeBot/v2/pkg/leetcodeclient"
	"github.com/dartkron/leetcodeBot/v2/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yandex-cloud/ydb-go-sdk/v2"
	"github.com/yandex-cloud/ydb-go-sdk/v2/table"
)

type MockQueryExecuter struct {
	mock.Mock
	t *testing.T
}

func (m *MockQueryExecuter) ProcessQuery(ctx context.Context, query string, queryParams *table.QueryParameters) (YDBResult, error) {
	query = trimmQuery(query)
	queryTokens := query
	for _, sign := range []string{";", ",", "(", ")", "="} {
		queryTokens = strings.ReplaceAll(queryTokens, sign, " ")
	}
	queryTokens = trimmQuery(queryTokens)
	tokens := strings.Split(queryTokens, " ")
	declaredVars := map[string]int{}
	for i, part := range tokens {
		token := part

		if _, ok := declaredVars[token]; ok {
			declaredVars[token]++
		}
		if part == "DECLARE" {
			_, ok := declaredVars[tokens[i+1]]
			assert.Falsef(m.t, ok, "variable %s DECLAREd twice in query ", tokens[i+1])
			declaredVars[tokens[i+1]] = 0
		}
	}
	for variable, amount := range declaredVars {
		assert.Equalf(m.t, amount, 2, "variable %s appeared only once in query. Seems it's only declared and not used", variable)
	}

	queryParams.Each(func(name string, _ ydb.Value) {
		_, ok := declaredVars[name]
		assert.Truef(m.t, ok, "Variable %s set in query params, but not exists in the query", name)
		delete(declaredVars, name)
	})
	assert.Empty(m.t, declaredVars, "There is variables DECLAREd in query, but omit in QueryParams")

	args := m.Called(query, queryParams.String())
	return args.Get(0).(YDBResult), args.Error(1)
}

type YDBResultMock struct {
	rows       []interface{}
	currentRow int
	err        error
	respFields []string
	t          *testing.T
	scanError  error
}

func (m *YDBResultMock) RowCount() int {
	return len(m.rows)
}

func (m *YDBResultMock) NextResultSet(ctx context.Context, fields ...string) bool {
	if m.currentRow > len(m.rows)-1 {
		return false
	}
	rowVal := reflect.ValueOf(m.rows[m.currentRow])
	rightFiledsNames := []string{}
	for _, field := range fields {
		fieldLower := strings.ReplaceAll(strings.ToLower(field), "_", "")
		field, ok := rowVal.Type().FieldByNameFunc(func(name string) bool {
			return strings.ToLower(name) == fieldLower
		})
		assert.Truef(m.t, ok, "Field %s not found in struct %s", fieldLower, rowVal.Type().Name())
		rightFiledsNames = append(rightFiledsNames, field.Name)
	}
	m.respFields = rightFiledsNames
	return true
}

func (m *YDBResultMock) NextRow() bool {
	m.currentRow++
	return m.currentRow <= len(m.rows)
}
func (m *YDBResultMock) Err() error {
	return m.err
}
func (m *YDBResultMock) Scan(values ...interface{}) error {
	if m.scanError != nil {
		return m.scanError
	}
	rowVal := reflect.ValueOf(m.rows[m.currentRow-1])
	for i, fieldName := range m.respFields {
		_, ok := rowVal.Type().FieldByNameFunc(func(name string) bool {
			return name == fieldName
		})
		if !assert.Truef(m.t, ok, "Field %s not found in struct %s", fieldName, rowVal.Type().Name()) {
			return errors.New("No such field in struct")
		}
		p := reflect.New(rowVal.FieldByName(fieldName).Type())
		p.Elem().Set(rowVal.FieldByName(fieldName))
		reflect.ValueOf(values[i]).Elem().Set(p)
	}
	return nil
}

func trimmQuery(query string) string {
	query = strings.ReplaceAll(query, "\n", " ")
	query = strings.ReplaceAll(query, "\t", " ")
	space := regexp.MustCompile(`\s+`)
	return space.ReplaceAllString(query, " ")
}

func TestNewYdbStorage(t *testing.T) {
	ydbStore := newYdbStorage()
	assert.NotNil(t, ydbStore.ydbExecuter, "ydbExecuter must be set in constructor")
}

func TestNewYDBExecuter(t *testing.T) {
	e := newYDBQueryExecuter()
	assert.NotNil(t, e.txc, "txc must be set in constructor")
	assert.NotNil(t, e.ExecQueryFunc, "ExecQueryFunc must be set in constructor")
	assert.NotNil(t, e.GetConnectionFunc, "GetConnectionFunc must be set in constructor")
}

func TestGetTaskYDBError(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	dateID := uint64(55674)
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(getTaskQuery),
		table.NewQueryParameters(
			table.ValueParam("$dateId", ydb.Uint64Value(dateID)),
		).String(),
	).Return(
		&table.Result{},
		tests.ErrBypassTest,
	)
	storage.ydbExecuter = mockExecuter
	resp, err := storage.getTask(context.Background(), dateID)
	assert.Equal(t, tests.ErrBypassTest, err, "Unexpected error")
	assert.Equal(t, common.BotLeetCodeTask{}, resp, "Unexpected task returned")
}

// It's necessary while in database []string storing as string
// and Difficutly is storing as uint8
type databaseBotLeetcodeTask struct {
	DateID     uint64
	QuestionID uint64
	TitleSlug  string
	Title      string
	Content    string
	Hints      string
	Difficulty uint8
}

func (dbTask *databaseBotLeetcodeTask) fillFromBotLeetcode(task common.BotLeetCodeTask) error {
	dbTask.DateID = task.DateID
	dbTask.QuestionID = task.QuestionID
	dbTask.TitleSlug = task.TitleSlug
	dbTask.Content = task.Content
	dbTask.Title = task.Title
	marshalledHints, err := json.Marshal(task.Hints)
	if err != nil {
		return err
	}
	dbTask.Hints = string(marshalledHints)
	dbTask.Difficulty = task.GetDifficultyNum()
	return nil
}

func TestGetTaskYDB(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	dateID := uint64(55674)
	taskToLoad := common.BotLeetCodeTask{
		DateID: dateID,
		LeetCodeTask: leetcodeclient.LeetCodeTask{
			QuestionID: 1235,
			TitleSlug:  "very-very-test-slug",
			Title:      "Very very test task",
			Content:    "My test context",
			Hints:      []string{"First hint", "Second hint", "Third hint"},
			Difficulty: "Easy",
		},
	}
	dbTask := databaseBotLeetcodeTask{}
	dbTask.fillFromBotLeetcode(taskToLoad)
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(getTaskQuery),
		table.NewQueryParameters(
			table.ValueParam("$dateId", ydb.Uint64Value(dateID)),
		).String(),
	).Return(
		&YDBResultMock{
			rows: []interface{}{dbTask},
			t:    t,
		},
		nil,
	)
	storage.ydbExecuter = mockExecuter
	resp, err := storage.getTask(context.Background(), dateID)
	assert.Nil(t, err, "Unexpected error")
	assert.Equal(t, taskToLoad, resp, "Unexpected task returned")
}

func TestGetTaskYDBNoRows(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	dateID := uint64(55674)
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(getTaskQuery),
		table.NewQueryParameters(
			table.ValueParam("$dateId", ydb.Uint64Value(dateID)),
		).String(),
	).Return(
		&YDBResultMock{
			rows: []interface{}{},
			t:    t,
		},
		nil,
	)
	storage.ydbExecuter = mockExecuter
	resp, err := storage.getTask(context.Background(), dateID)
	assert.Equal(t, ErrNoSuchTask, err, "Unexpected error")
	assert.Equal(t, common.BotLeetCodeTask{}, resp, "Unexpected task returned")
}

func TestGetTaskYDBBrokenJSON(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	dateID := uint64(55674)
	taskToLoad := common.BotLeetCodeTask{
		DateID: dateID,
		LeetCodeTask: leetcodeclient.LeetCodeTask{
			QuestionID: 1235,
			TitleSlug:  "very-very-test-slug",
			Title:      "Very very test task",
			Content:    "My test context",
			Hints:      []string{"First hint", "Second hint", "Third hint"},
			Difficulty: "Easy",
		},
	}
	dbTask := databaseBotLeetcodeTask{}
	dbTask.fillFromBotLeetcode(taskToLoad)
	dbTask.Hints = "{\"}"
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(getTaskQuery),
		table.NewQueryParameters(
			table.ValueParam("$dateId", ydb.Uint64Value(dateID)),
		).String(),
	).Return(
		&YDBResultMock{
			rows: []interface{}{dbTask},
			t:    t,
		},
		nil,
	)
	storage.ydbExecuter = mockExecuter
	resp, err := storage.getTask(context.Background(), dateID)
	if assert.NotNil(t, err, "Expect error on broken JSON") {
		assert.Equal(t, tests.ErrWrongJSON.Error(), err.Error(), "Unexpected error")
	}
	assert.Equal(t, common.BotLeetCodeTask{}, resp, "Unexpected task returned")
}

func TestGetTaskYDBScanError(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	dateID := uint64(55674)
	taskToLoad := common.BotLeetCodeTask{
		DateID: dateID,
		LeetCodeTask: leetcodeclient.LeetCodeTask{
			QuestionID: 1235,
			TitleSlug:  "very-very-test-slug",
			Title:      "Very very test task",
			Content:    "My test context",
			Hints:      []string{"First hint", "Second hint", "Third hint"},
			Difficulty: "Easy",
		},
	}
	dbTask := databaseBotLeetcodeTask{}
	dbTask.fillFromBotLeetcode(taskToLoad)
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(getTaskQuery),
		table.NewQueryParameters(
			table.ValueParam("$dateId", ydb.Uint64Value(dateID)),
		).String(),
	).Return(
		&YDBResultMock{
			rows:      []interface{}{dbTask},
			t:         t,
			scanError: tests.ErrBypassTest,
		},
		nil,
	)
	storage.ydbExecuter = mockExecuter
	resp, err := storage.getTask(context.Background(), dateID)
	assert.Equal(t, tests.ErrBypassTest, err, "Unexpected error")
	assert.Equal(t, common.BotLeetCodeTask{}, resp, "Unexpected task returned")
}

func TestSaveTask(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	storage.ydbExecuter = mockExecuter
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(replaceTaskQuery),
		mock.Anything,
	).Return(
		&YDBResultMock{
			rows:      []interface{}{},
			t:         t,
			scanError: tests.ErrBypassTest,
		},
		nil,
	)
	err := storage.saveTask(context.Background(), common.BotLeetCodeTask{})
	assert.Nil(t, err, "Unexpected error")
}

func TestSaveTaskErr(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	storage.ydbExecuter = mockExecuter
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(replaceTaskQuery),
		mock.Anything,
	).Return(
		&YDBResultMock{
			rows:      []interface{}{},
			t:         t,
			scanError: tests.ErrBypassTest,
		},
		tests.ErrBypassTest,
	)
	err := storage.saveTask(context.Background(), common.BotLeetCodeTask{})
	assert.Equal(t, tests.ErrBypassTest, err, "Unexpected error")
}

func TestSaveUser(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	storage.ydbExecuter = mockExecuter
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(saveUserQuery),
		mock.Anything,
	).Return(
		&YDBResultMock{
			rows:      []interface{}{},
			t:         t,
			scanError: tests.ErrBypassTest,
		},
		nil,
	)
	err := storage.saveUser(context.Background(), common.User{})
	assert.Nil(t, err, "Unexpected error")
}

func TestSaveUserErr(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	storage.ydbExecuter = mockExecuter
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(saveUserQuery),
		mock.Anything,
	).Return(
		&YDBResultMock{
			rows:      []interface{}{},
			t:         t,
			scanError: tests.ErrBypassTest,
		},
		tests.ErrBypassTest,
	)
	err := storage.saveUser(context.Background(), common.User{})
	assert.Equal(t, tests.ErrBypassTest, err, "Unexpected error")
}

func TestSubscribeUser(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	storage.ydbExecuter = mockExecuter
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(subscribeUserQuery),
		mock.Anything,
	).Return(
		&YDBResultMock{
			rows:      []interface{}{},
			t:         t,
			scanError: tests.ErrBypassTest,
		},
		nil,
	)
	err := storage.subscribeUser(context.Background(), 123)
	assert.Nil(t, err, "Unexpected error")
}

func TestSubscribeUserError(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	storage.ydbExecuter = mockExecuter
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(subscribeUserQuery),
		mock.Anything,
	).Return(
		&YDBResultMock{
			rows:      []interface{}{},
			t:         t,
			scanError: tests.ErrBypassTest,
		},
		tests.ErrBypassTest,
	)
	err := storage.subscribeUser(context.Background(), 123)
	assert.Equal(t, tests.ErrBypassTest, err, "Unexpected error")
}

func TestUnsubscribeUser(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	storage.ydbExecuter = mockExecuter
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(unsubscribeUserQuery),
		mock.Anything,
	).Return(
		&YDBResultMock{
			rows:      []interface{}{},
			t:         t,
			scanError: tests.ErrBypassTest,
		},
		nil,
	)
	err := storage.unsubscribeUser(context.Background(), 123)
	assert.Nil(t, err, "Unexpected error")
}

func TestUnsubscribeUserError(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	storage.ydbExecuter = mockExecuter
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(unsubscribeUserQuery),
		mock.Anything,
	).Return(
		&YDBResultMock{
			rows:      []interface{}{},
			t:         t,
			scanError: tests.ErrBypassTest,
		},
		tests.ErrBypassTest,
	)
	err := storage.unsubscribeUser(context.Background(), 123)
	assert.Equal(t, tests.ErrBypassTest, err, "Unexpected error")
}

func TestGetSubscribedUsersDB(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	storage.ydbExecuter = mockExecuter
	usersToCheck := []common.User{
		{
			ID:         123,
			ChatID:     123,
			Username:   "test1",
			FirstName:  "ftest1",
			LastName:   "ltest1",
			Subscribed: true,
		},
		{
			ID:         122,
			ChatID:     122,
			Username:   "test2",
			FirstName:  "ftest2",
			LastName:   "ltest2",
			Subscribed: true,
		},
		{
			ID:         124,
			ChatID:     124,
			Username:   "test3",
			FirstName:  "ftest3",
			LastName:   "ltest3",
			Subscribed: true,
		},
		{
			ID:         125,
			ChatID:     125,
			Username:   "test4",
			FirstName:  "ftest4",
			LastName:   "ltest4",
			Subscribed: true,
		},
	}
	rows := []interface{}{}
	for _, user := range usersToCheck {
		rows = append(rows, interface{}(user))
	}
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(getSubscribedUsersQuery),
		mock.Anything,
	).Return(
		&YDBResultMock{
			rows: rows,
			t:    t,
		},
		nil,
	)

	users, err := storage.getSubscribedUsers(context.Background())
	assert.Nil(t, err, "Unexpected error")
	assert.Equal(t, usersToCheck, users, "Unexpected users returned")
}

func TestGetSubscribedUsersDBError(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	storage.ydbExecuter = mockExecuter
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(getSubscribedUsersQuery),
		mock.Anything,
	).Return(
		&YDBResultMock{
			rows: []interface{}{},
			t:    t,
		},
		tests.ErrBypassTest,
	)

	users, err := storage.getSubscribedUsers(context.Background())
	assert.Equal(t, tests.ErrBypassTest, err, "Unexpected error")
	assert.Equal(t, []common.User{}, users, "Unexpected users returned")
}

func TestGetSubscribedUsersScanError(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	storage.ydbExecuter = mockExecuter
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(getSubscribedUsersQuery),
		mock.Anything,
	).Return(
		&YDBResultMock{
			rows:      []interface{}{common.User{}},
			t:         t,
			scanError: tests.ErrBypassTest,
		},
		nil,
	)

	users, err := storage.getSubscribedUsers(context.Background())
	assert.Equal(t, tests.ErrBypassTest, err, "Unexpected error")
	assert.Equal(t, []common.User{}, users, "Unexpected users returned")
}

func TestGetUser(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	storage.ydbExecuter = mockExecuter
	userToCheck := common.User{
		ID:         123,
		ChatID:     123,
		Username:   "test1",
		FirstName:  "ftest1",
		LastName:   "ltest1",
		Subscribed: true,
	}
	rows := []interface{}{interface{}(userToCheck)}
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(getUserQuery),
		mock.Anything,
	).Return(
		&YDBResultMock{
			rows: rows,
			t:    t,
		},
		nil,
	)

	user, err := storage.getUser(context.Background(), 123)
	assert.Nil(t, err, "Unexpected error")
	assert.Equal(t, userToCheck, user, "Unexpected user returned")
}

func TestGetUserNoRows(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	storage.ydbExecuter = mockExecuter
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(getUserQuery),
		mock.Anything,
	).Return(
		&YDBResultMock{
			rows: []interface{}{},
			t:    t,
		},
		nil,
	)

	user, err := storage.getUser(context.Background(), 123)
	assert.Equal(t, ErrNoSuchUser, err, "Unexpected error")
	assert.Equal(t, common.User{}, user, "Unexpected user returned")
}

func TestGetUserErrors(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	storage.ydbExecuter = mockExecuter
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(getUserQuery),
		mock.Anything,
	).Return(
		&YDBResultMock{
			rows: []interface{}{},
			t:    t,
		},
		tests.ErrBypassTest,
	)

	user, err := storage.getUser(context.Background(), 123)
	assert.Equal(t, tests.ErrBypassTest, err, "Unexpected error")
	assert.Equal(t, common.User{}, user, "Unexpected user returned")
}

func TestGetUserScanError(t *testing.T) {
	storage := newYdbStorage()
	mockExecuter := new(MockQueryExecuter)
	mockExecuter.t = t
	storage.ydbExecuter = mockExecuter
	mockExecuter.On(
		"ProcessQuery",
		trimmQuery(getUserQuery),
		mock.Anything,
	).Return(
		&YDBResultMock{
			rows:      []interface{}{common.User{}},
			t:         t,
			scanError: tests.ErrBypassTest,
		},
		nil,
	)

	user, err := storage.getUser(context.Background(), 123)
	assert.Equal(t, tests.ErrBypassTest, err, "Unexpected error")
	assert.Equal(t, common.User{}, user, "Unexpected user returned")
}

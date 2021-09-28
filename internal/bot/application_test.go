package bot

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/dartkron/leetcodeBot/v2/internal/common"
	"github.com/dartkron/leetcodeBot/v2/internal/storage"
	"github.com/dartkron/leetcodeBot/v2/pkg/leetcodeclient"
	lcclientmocks "github.com/dartkron/leetcodeBot/v2/pkg/leetcodeclient/mocks"
	"github.com/dartkron/leetcodeBot/v2/tests"
	"github.com/dartkron/leetcodeBot/v2/tests/mocks"
	"github.com/stretchr/testify/assert"
)

type MockStorageController struct {
	tasks                      map[uint64]*common.BotLeetCodeTask
	users                      map[uint64]*common.User
	callsJournal               []string
	getSubscribedUsersMustFail bool
	failedUserID               uint64
	failedTaskID               uint64
}

func (controller *MockStorageController) GetTask(dateID uint64) (common.BotLeetCodeTask, error) {
	controller.callsJournal = append(controller.callsJournal, fmt.Sprintf("GetTask %d", dateID))
	if dateID == controller.failedTaskID {
		return common.BotLeetCodeTask{}, tests.ErrBypassTest
	}
	if task, ok := controller.tasks[dateID]; ok {
		return *task, nil
	}
	return common.BotLeetCodeTask{}, storage.ErrNoSuchTask
}

func (controller *MockStorageController) SaveTask(task common.BotLeetCodeTask) error {
	controller.callsJournal = append(controller.callsJournal, fmt.Sprintf("SaveTask %d", task.DateID))
	if task.DateID == controller.failedTaskID {
		return tests.ErrBypassTest
	}
	controller.tasks[task.DateID] = &task
	return nil
}

func (controller *MockStorageController) SubscribeUser(user common.User) error {
	controller.callsJournal = append(controller.callsJournal, fmt.Sprintf("SubscribeUser %d", user.ID))
	if user.ID == controller.failedUserID {
		return tests.ErrBypassTest
	}
	if user, ok := controller.users[user.ID]; ok {
		if user.Subscribed {
			return storage.ErrUserAlreadySubscribed
		}
	} else {
		user.Subscribed = true
		controller.users[user.ID] = user
	}
	return nil
}

func (controller *MockStorageController) UnsubscribeUser(userID uint64) error {
	controller.callsJournal = append(controller.callsJournal, fmt.Sprintf("UnsubscribeUser %d", userID))
	if userID == controller.failedUserID {
		return tests.ErrBypassTest
	}
	if user, ok := controller.users[userID]; ok {
		if !user.Subscribed {
			return storage.ErrUserAlreadyUnsubscribed
		}
		user.Subscribed = false
	} else {
		return storage.ErrUserAlreadyUnsubscribed
	}
	return nil
}

func (controller *MockStorageController) GetSubscribedUsers() ([]common.User, error) {
	controller.callsJournal = append(controller.callsJournal, "GetSubscribedUsers")
	if controller.getSubscribedUsersMustFail {
		return []common.User{}, tests.ErrBypassTest
	}
	resp := []common.User{}
	for _, user := range controller.users {
		if user.Subscribed {
			resp = append(resp, *user)
		}
	}
	return resp, nil
}

func TestGetMainKeyboard(t *testing.T) {
	waitKeyboard := "{\"keyboard\":[[{\"text\":\"Get actual daily task\"}],[{\"text\":\"Subscribe\"},{\"text\":\"Unsubscribe\"}]],\"input_field_placeholder\":\"Please, use buttons below:\",\"resize_keyboard\":true}"
	keyboard, err := GetMainKeyboard()
	assert.Equal(t, err, nil, "Got uexpected error from GetMainKeyboard")
	assert.Equal(t, waitKeyboard, keyboard, "Returned keyboard not equal with expected")
}

func getTodaySendMessageString(chatID uint64) string {
	template := "{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":%d,\"text\":\"\\u003cstrong\\u003eTest title\\u003c/strong\\u003e\\n\\nTest content\",\"reply_markup\":\"" +
		"{\\\"inline_keyboard\\\":[[{\\\"text\\\":\\\"See task on LeetCode website\\\",\\\"url\\\":\\\"https://leetcode.com/explore/item/6534\\\"}]," +
		"[{\\\"text\\\":\\\"Hint 1\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"DATEID_PLACE\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":0}\\\"}," +
		"{\\\"text\\\":\\\"Hint 2\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"DATEID_PLACE\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":1}\\\"}]," +
		"[{\\\"text\\\":\\\"Hint: Get the difficulty of the task\\\",\\\"callback_data\\\":" +
		"\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"DATEID_PLACE\\\\\\\",\\\\\\\"callback_type\\\\\\\":1,\\\\\\\"hint\\\\\\\":0}\\\"}]]}\"}"
	dateID := common.GetDateIDForNow()
	template = strings.ReplaceAll(template, "DATEID_PLACE", fmt.Sprintf("%d", dateID))
	return fmt.Sprintf(template, chatID)
}

func getTestApp() (*mocks.MockHTTPPost, *MockStorageController, *lcclientmocks.MockLeetcodeClient, *Application) {
	httpMock := &mocks.MockHTTPPost{}
	leetcodeClient := &lcclientmocks.MockLeetcodeClient{}
	storageController := &MockStorageController{
		tasks: map[uint64]*common.BotLeetCodeTask{},
		users: map[uint64]*common.User{
			1124: {
				ID:         1124,
				ChatID:     1124,
				Username:   "testuser1124",
				FirstName:  "1124firstname",
				LastName:   "1124lastname",
				Subscribed: false,
			},
			1126: {
				ID:         1126,
				ChatID:     1126,
				Username:   "testuser1126",
				FirstName:  "1126firstname",
				LastName:   "1126lastname",
				Subscribed: true,
			},
			1128: {
				ID:         1128,
				ChatID:     1128,
				Username:   "testuser1128",
				FirstName:  "1128firstname",
				LastName:   "1128lastname",
				Subscribed: false,
			},
			1120: {
				ID:         1120,
				ChatID:     1120,
				Username:   "testuser1120",
				FirstName:  "1120firstname",
				LastName:   "1120lastname",
				Subscribed: true,
			},
		},
	}
	app := &Application{
		leetcodeAPIClient: leetcodeClient,
		PostFunc:          httpMock.Func,
		storageController: storageController,
	}
	return httpMock, storageController, leetcodeClient, app
}

func TestSendDailyTaskToSubscribedUsers(t *testing.T) {
	httpMock, storageController, _, app := getTestApp()
	httpMock.On(
		"Func",
		"https://api.telegram.org/bot/sendMessage",
		"application/json",
		getTodaySendMessageString(1120),
	).Return(
		&http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("1120"))},
		nil,
	).Times(1)
	httpMock.On(
		"Func",
		"https://api.telegram.org/bot/sendMessage",
		"application/json",
		getTodaySendMessageString(1126),
	).Return(
		&http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("1126"))},
		nil,
	).Times(1)
	taskDateID := common.GetDateIDForNow()
	storageController.tasks[taskDateID] = &common.BotLeetCodeTask{
		DateID: taskDateID,
		LeetCodeTask: leetcodeclient.LeetCodeTask{
			QuestionID: 1445,
			ItemID:     6534,
			Title:      "Test title",
			Content:    "Test content",
			Hints:      []string{"first hint", "Second Hint"},
			Difficulty: "Easy",
		},
	}

	err := app.SendDailyTaskToSubscribedUsers()
	assert.Equal(t, err, nil, "Got unexprected error from SendDailyTaskToSubscribedUsers")
	httpMock.AssertNumberOfCalls(t, "Func", 2)
}

func TestSendDailyTaskToSubscribedUsersError(t *testing.T) {
	httpMock, storageController, _, app := getTestApp()
	httpMock.On(
		"Func",
		"https://api.telegram.org/bot/sendMessage",
		"application/json",
		getTodaySendMessageString(1127),
	).Return(
		&http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("1127"))},
		nil,
	).Times(1)
	httpMock.On(
		"Func",
		"https://api.telegram.org/bot/sendMessage",
		"application/json",
		getTodaySendMessageString(1126),
	).Return(
		&http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("1126"))},
		nil,
	).Times(1)
	httpMock.On(
		"Func",
		"https://api.telegram.org/bot/sendMessage",
		"application/json",
		getTodaySendMessageString(1121),
	).Return(
		&http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("1121"))},
		tests.ErrBypassTest,
	).Times(3)
	httpMock.On(
		"Func",
		"https://api.telegram.org/bot/sendMessage",
		"application/json",
		getTodaySendMessageString(1120),
	).Return(
		&http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("1120"))},
		nil,
	).Times(3)
	taskDateID := common.GetDateIDForNow()
	storageController.tasks[taskDateID] = &common.BotLeetCodeTask{
		DateID: taskDateID,
		LeetCodeTask: leetcodeclient.LeetCodeTask{
			QuestionID: 1445,
			ItemID:     6534,
			Title:      "Test title",
			Content:    "Test content",
			Hints:      []string{"first hint", "Second Hint"},
			Difficulty: "Easy",
		},
	}

	storageController.users[1121] = &common.User{
		ID:         1121,
		ChatID:     1121,
		Username:   "testuser1121",
		FirstName:  "1121firstname",
		LastName:   "1121lastname",
		Subscribed: true,
	}

	storageController.users[1127] = &common.User{
		ID:         1127,
		ChatID:     1127,
		Username:   "testuser1127",
		FirstName:  "1127firstname",
		LastName:   "1127lastname",
		Subscribed: true,
	}

	err := app.SendDailyTaskToSubscribedUsers()
	assert.Equal(t, err, nil, "Unexprected error from SendDailyTaskToSubscribedUsers")
	httpMock.AssertExpectations(t)
}

func TestSendDailyTaskToSubscribedUsersWithoutTasks(t *testing.T) {
	httpMock, storageController, leetcodeClient, app := getTestApp()
	taskDateID := common.GetDateIDForNow()
	storageController.failedTaskID = taskDateID
	leetcodeClient.On(
		"GetDailyTask",
		taskDateID,
	).Return(
		leetcodeclient.LeetCodeTask{},
		tests.ErrBypassTest,
	).Times(1)

	err := app.SendDailyTaskToSubscribedUsers()
	assert.Equal(t, err, tests.ErrBypassTest, "Unexprected error from SendDailyTaskToSubscribedUsers")
	leetcodeClient.AssertExpectations(t)
	httpMock.AssertExpectations(t)
}

func TestSendDailyTaskToSubscribedUsersWithErrorOnUsersList(t *testing.T) {
	httpMock, storageController, leetcodeClient, app := getTestApp()
	taskDateID := common.GetDateIDForNow()
	storageController.failedTaskID = taskDateID
	storageController.getSubscribedUsersMustFail = true

	err := app.SendDailyTaskToSubscribedUsers()
	assert.Equal(t, err, tests.ErrBypassTest, "Unexprected error from SendDailyTaskToSubscribedUsers")
	leetcodeClient.AssertExpectations(t)
	httpMock.AssertExpectations(t)
}

func TestGetTodayTaskFromAllPossibleSourcesFromClient(t *testing.T) {
	_, storageController, leetcodeClient, app := getTestApp()
	todayDateID := common.GetDateIDForNow()
	testTask := common.BotLeetCodeTask{
		DateID: todayDateID,
		LeetCodeTask: leetcodeclient.LeetCodeTask{
			QuestionID: 202,
			ItemID:     667,
			Title:      "Test task from client",
			Content:    "Test content from client",
			Hints:      []string{"one hint", "second hint"},
			Difficulty: "Easy",
		},
	}
	leetcodeClient.On(
		"GetDailyTask",
		todayDateID,
	).Return(
		testTask.LeetCodeTask,
		nil,
	).Times(1)
	task, err := app.GetTodayTaskFromAllPossibleSources()
	assert.Equal(t, err, nil, "Unexprected error from GetTodayTaskFromAllPossibleSources")
	assert.Equal(t, task, testTask, "Returned task not equal with expected")

	awaitedCalls := []string{fmt.Sprintf("GetTask %d", todayDateID), fmt.Sprintf("SaveTask %d", todayDateID)}
	if !reflect.DeepEqual(storageController.callsJournal, awaitedCalls) {
		t.Errorf("Storage controller calls:\n%q\nisn't equal to expected:\n%q\n", storageController.callsJournal, awaitedCalls)
	}

	awaitedCalls = []string{fmt.Sprintf("GetDailyTask %d", todayDateID)}
	leetcodeClient.AssertExpectations(t)

}

func TestGetTodayTaskFromAllPossibleSourcesFromClientWithErrorFromStorage(t *testing.T) {
	_, storageController, leetcodeClient, app := getTestApp()
	todayDateID := common.GetDateIDForNow()
	testTask := common.BotLeetCodeTask{
		DateID: todayDateID,
		LeetCodeTask: leetcodeclient.LeetCodeTask{
			QuestionID: 202,
			ItemID:     667,
			Title:      "Test task from client",
			Content:    "Test content from client",
			Hints:      []string{"one hint", "second hint"},
			Difficulty: "Easy",
		},
	}
	leetcodeClient.On(
		"GetDailyTask",
		todayDateID,
	).Return(
		testTask.LeetCodeTask,
		nil,
	).Times(1)
	storageController.failedTaskID = todayDateID
	task, err := app.GetTodayTaskFromAllPossibleSources()
	assert.Equal(t, err, tests.ErrBypassTest, "Unexprected error from GetTodayTaskFromAllPossibleSources")
	assert.Equal(t, task, testTask, "Returned task not equal with expected")

	awaitedCalls := []string{fmt.Sprintf("GetTask %d", todayDateID), fmt.Sprintf("SaveTask %d", todayDateID)}
	if !reflect.DeepEqual(storageController.callsJournal, awaitedCalls) {
		t.Errorf("Storage controller calls:\n%q\nisn't equal to expected:\n%q\n", storageController.callsJournal, awaitedCalls)
	}
	leetcodeClient.AssertExpectations(t)
}

func TestGetTodayTaskFromAllPossibleSourcesFromStorage(t *testing.T) {
	_, storageController, leetcodeClient, app := getTestApp()
	todayDateID := common.GetDateIDForNow()
	testTask := common.BotLeetCodeTask{
		DateID: todayDateID,
		LeetCodeTask: leetcodeclient.LeetCodeTask{
			QuestionID: 202,
			ItemID:     667,
			Title:      "Test task from client",
			Content:    "Test content from client",
			Hints:      []string{"one hint", "second hint"},
			Difficulty: "Easy",
		},
	}
	storageController.tasks[todayDateID] = &testTask
	task, err := app.GetTodayTaskFromAllPossibleSources()
	assert.Equal(t, err, nil, "Unexprected error from GetTodayTaskFromAllPossibleSources")
	assert.Equal(t, task, testTask, "Returned task not equal with expected")
	assert.Equal(t, storageController.callsJournal, []string{fmt.Sprintf("GetTask %d", todayDateID)}, "Unexpected storageController calls journal")
	leetcodeClient.AssertExpectations(t)
}

func TestNewApplication(t *testing.T) {
	app := NewApplication()
	assert.NotNil(t, app.leetcodeAPIClient, "leetcodeAPIClient must be set in constructor")
	assert.NotNil(t, app.storageController, "storageController must be set in constructor")
	assert.NotNil(t, app.PostFunc, "PostFunc must be set in constructor")
}

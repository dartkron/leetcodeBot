package bot

import (
	"encoding/json"
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

func TestSubscribeAction(t *testing.T) {
	_, storageController, _, app := getTestApp()
	userBeforeRequest := storageController.users[1124]
	assert.False(t, userBeforeRequest.Subscribed, "Before request user shouldn't be scubscribed")
	request := TelegramRequest{}
	request.Message.From.ID = 1124
	request.Message.Chat.ID = 1124
	request.Message.From.Username = "testuser1125"
	request.Message.From.FirstName = "1125firstname"
	request.Message.From.LastName = "1125lastname"
	response := TelegramResponse{}
	err := app.subscribeAction(&request, &response)
	assert.Nil(t, err, "Unexpected subscribeAction error")
	userBeforeRequest.Subscribed = true
	assert.Equal(t, userBeforeRequest, storageController.users[1124], "Unexpected changes in stored user after subscribe action")
	assert.Equal(t, fmt.Sprintf(subcribedMessage, request.Message.From.FirstName), response.Text, "Unexpected response text")
}

func TestSubscribeActionAlreadySubscribed(t *testing.T) {
	_, storageController, _, app := getTestApp()
	userBeforeRequest := storageController.users[1126]
	assert.True(t, userBeforeRequest.Subscribed, "Before request user should be scubscribed")
	request := TelegramRequest{}
	request.Message.From.ID = 1126
	request.Message.Chat.ID = 1126
	request.Message.From.Username = "testuser1125"
	request.Message.From.FirstName = "1125firstname"
	request.Message.From.LastName = "1125lastname"
	response := TelegramResponse{}
	err := app.subscribeAction(&request, &response)
	assert.Nil(t, err, "Unexpected subscribeAction error")
	assert.Equal(t, userBeforeRequest, storageController.users[1126], "Unexpected changes in stored user after subscribe action")
	assert.Equal(t, fmt.Sprintf(alreadySubscribedMessage, request.Message.From.FirstName), response.Text, "Unexpected response text")
}

func TestSubscribeActionWithError(t *testing.T) {
	_, storageController, _, app := getTestApp()
	storageController.failedUserID = 1126
	request := TelegramRequest{}
	request.Message.From.ID = 1126
	request.Message.Chat.ID = 1126
	request.Message.From.Username = "testuser1125"
	request.Message.From.FirstName = "1125firstname"
	request.Message.From.LastName = "1125lastname"
	response := TelegramResponse{}
	err := app.subscribeAction(&request, &response)
	assert.Equal(t, err, tests.ErrBypassTest, "Unexpected subscribeAction error")
	assert.Empty(t, response.Text, "Response text should be empty on error")
}

func TestUnsubscribeAction(t *testing.T) {
	_, storageController, _, app := getTestApp()
	userBeforeRequest := storageController.users[1126]
	assert.True(t, userBeforeRequest.Subscribed, "Before request user should be subscribed")
	request := TelegramRequest{}
	request.Message.From.ID = 1126
	request.Message.Chat.ID = 1126
	request.Message.From.Username = "testuser1125"
	request.Message.From.FirstName = "1125firstname"
	request.Message.From.LastName = "1125lastname"
	response := TelegramResponse{}
	err := app.unsubscribeAction(&request, &response)
	assert.Nil(t, err, "Unexpected unsubscribeAction error")
	userBeforeRequest.Subscribed = false
	assert.Equal(t, userBeforeRequest, storageController.users[1126], "Unexpected changes in stored user after subscribe action")
	assert.Equal(t, fmt.Sprintf(unsubscribedMessage, request.Message.From.FirstName), response.Text, "Unexpected response text")
}

func TestUnsubscribeActionAlreadyUnsubscribed(t *testing.T) {
	_, storageController, _, app := getTestApp()
	userBeforeRequest := storageController.users[1124]
	assert.False(t, userBeforeRequest.Subscribed, "Before request user shouldn't be subscribed")
	request := TelegramRequest{}
	request.Message.From.ID = 1124
	request.Message.Chat.ID = 1124
	request.Message.From.Username = "testuser1125"
	request.Message.From.FirstName = "1125firstname"
	request.Message.From.LastName = "1125lastname"
	response := TelegramResponse{}
	err := app.unsubscribeAction(&request, &response)
	assert.Nil(t, err, "Unexpected unsubscribeAction error")
	userBeforeRequest.Subscribed = false
	assert.Equal(t, userBeforeRequest, storageController.users[1124], "Unexpected changes in stored user after subscribe action")
	assert.Equal(t, fmt.Sprintf(alreadyUnsubscribedMessage, request.Message.From.FirstName), response.Text, "Unexpected response text")
}

func TestUnsubscribeActionWithError(t *testing.T) {
	_, storageController, _, app := getTestApp()
	storageController.failedUserID = 1126
	request := TelegramRequest{}
	request.Message.From.ID = 1126
	request.Message.Chat.ID = 1126
	request.Message.From.Username = "testuser1125"
	request.Message.From.FirstName = "1125firstname"
	request.Message.From.LastName = "1125lastname"
	response := TelegramResponse{}
	err := app.unsubscribeAction(&request, &response)
	assert.Equal(t, err, tests.ErrBypassTest, "Unexpected subscribeAction error")
	assert.Empty(t, response.Text, "Response text should be empty on error")
}

func TestGetTaskAction(t *testing.T) {
	_, storageController, _, app := getTestApp()
	todayTaskID := common.GetDateIDForNow()
	storageController.tasks[todayTaskID] = &common.BotLeetCodeTask{
		DateID: todayTaskID,
		LeetCodeTask: leetcodeclient.LeetCodeTask{
			QuestionID: 1445,
			ItemID:     6534,
			Title:      "Test title",
			Content:    "Test content",
			Hints:      []string{"first hint", "Second Hint"},
			Difficulty: "Easy",
		},
	}
	response := TelegramResponse{}
	err := app.getTaskAction(&response)
	assert.Nil(t, err, "Unexpected getTaskAction error")
	assert.Equal(t, response.Text, storageController.tasks[todayTaskID].GetTaskText(), "Unexpected response text")
	assert.Equal(t, response.ReplyMarkup, storageController.tasks[todayTaskID].GetInlineKeyboard(), "Unexpected reply markup text")
}

func TestGetTaskActionError(t *testing.T) {
	_, storageController, lcClient, app := getTestApp()
	todayTaskID := common.GetDateIDForNow()
	lcClient.On(
		"GetDailyTask",
		todayTaskID,
	).Return(
		leetcodeclient.LeetCodeTask{},
		tests.ErrBypassTest,
	)
	delete(storageController.tasks, todayTaskID)
	response := TelegramResponse{}
	err := app.getTaskAction(&response)
	assert.Equal(t, err, tests.ErrBypassTest, "Unexpected getTaskAction error")
	assert.Empty(t, response.Text, "Unexpected response text")
	assert.Empty(t, response.ReplyMarkup, "Unexpected reply markup text")
}

func TestNewTelegramResponse(t *testing.T) {
	response := NewTelegramResponse()
	assert.NotEmpty(t, response.Method, "Constructor should fill Method")
	assert.NotEmpty(t, response.ParseMode, "Constructor should fill ParseMode")
}

func TestProcessRequestBodyHelp(t *testing.T) {
	_, _, _, app := getTestApp()
	request := TelegramRequest{}
	request.Message.From.ID = 6667
	request.Message.Text = "My test request!"
	requestbytes, err := json.Marshal(request)
	assert.Nil(t, err, "Unexpected json.Marshal error")
	responseBytes, err := app.ProcessRequestBody(requestbytes)
	assert.Nil(t, err, "Unexpected ProcessRequestBody error")
	expectedResponse := "{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":0,\"text\":\"You command \\\"My test request!\\\" isn't recognized =(\\nList of available commands:\\n/getDailyTask — get actual dailyTask\\n/Subscribe — start automatically sending of daily tasks\\n/Unsubscribe — stop automatically sending of daily tasks\",\"reply_markup\":\"{\\\"keyboard\\\":[[{\\\"text\\\":\\\"Get actual daily task\\\"}],[{\\\"text\\\":\\\"Subscribe\\\"},{\\\"text\\\":\\\"Unsubscribe\\\"}]],\\\"input_field_placeholder\\\":\\\"Please, use buttons below:\\\",\\\"resize_keyboard\\\":true}\"}"
	assert.Equal(t, responseBytes, []byte(expectedResponse), "Unexprected response bytes")
}

func TestProcessRequestSubscribe(t *testing.T) {
	_, _, _, app := getTestApp()
	request := TelegramRequest{}
	request.Message.From.ID = 1126
	request.Message.Chat.ID = 1126
	request.Message.From.FirstName = "TestUser"
	request.Message.Text = subscribeCommand
	requestbytes, err := json.Marshal(request)
	assert.Nil(t, err, "Unexpected json.Marshal error")
	responseBytes, err := app.ProcessRequestBody(requestbytes)
	assert.Nil(t, err, "Unexpected ProcessRequestBody error")
	expectedResponse := "{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":1126,\"text\":\"TestUser, you have \\u003cstrong\\u003ealready subscribed\\u003c/strong\\u003e for daily updates, nothing to do.\",\"reply_markup\":\"{\\\"keyboard\\\":[[{\\\"text\\\":\\\"Get actual daily task\\\"}],[{\\\"text\\\":\\\"Subscribe\\\"},{\\\"text\\\":\\\"Unsubscribe\\\"}]],\\\"input_field_placeholder\\\":\\\"Please, use buttons below:\\\",\\\"resize_keyboard\\\":true}\"}"
	assert.Equal(t, responseBytes, []byte(expectedResponse), "Unexprected response bytes")
}

func TestProcessRequestSubscribeError(t *testing.T) {
	_, storageController, _, app := getTestApp()
	request := TelegramRequest{}
	request.Message.From.ID = 1126
	request.Message.Chat.ID = 1126
	request.Message.From.FirstName = "TestUser"
	request.Message.Text = subscribeCommand
	storageController.failedUserID = 1126
	requestbytes, err := json.Marshal(request)
	assert.Nil(t, err, "Unexpected json.Marshal error")
	responseBytes, err := app.ProcessRequestBody(requestbytes)
	assert.Equal(t, err, tests.ErrBypassTest, "Unexpected ProcessRequestBody text")
	assert.Empty(t, responseBytes, "Unexprected response bytes")
}

func TestProcessRequestUnsubscribe(t *testing.T) {
	_, _, _, app := getTestApp()
	request := TelegramRequest{}
	request.Message.From.ID = 1126
	request.Message.Chat.ID = 1126
	request.Message.From.FirstName = "TestUser"
	request.Message.Text = unsubscribeCommand
	requestbytes, err := json.Marshal(request)
	assert.Nil(t, err, "Unexpected json.Marshal error")
	responseBytes, err := app.ProcessRequestBody(requestbytes)
	assert.Nil(t, err, "Unexpected ProcessRequestBody error")
	expectedResponse := "{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":1126,\"text\":\"TestUser, you have \\u003cstrong\\u003esucessfully unsubscribed\\u003c/strong\\u003e. You'll not automatically recieve daily tasks.\\nIf you've found this bot useless and have ideas of possible improvements, please, add them to https://github.com/dartkron/leetcodeBot/issues\",\"reply_markup\":\"{\\\"keyboard\\\":[[{\\\"text\\\":\\\"Get actual daily task\\\"}],[{\\\"text\\\":\\\"Subscribe\\\"},{\\\"text\\\":\\\"Unsubscribe\\\"}]],\\\"input_field_placeholder\\\":\\\"Please, use buttons below:\\\",\\\"resize_keyboard\\\":true}\"}"
	assert.Equal(t, responseBytes, []byte(expectedResponse), "Unexprected response bytes")
}

func TestProcessRequestUnsubscribeError(t *testing.T) {
	_, storageController, _, app := getTestApp()
	request := TelegramRequest{}
	request.Message.From.ID = 1126
	request.Message.Chat.ID = 1126
	request.Message.From.FirstName = "TestUser"
	request.Message.Text = unsubscribeCommand
	storageController.failedUserID = 1126
	requestbytes, err := json.Marshal(request)
	assert.Nil(t, err, "Unexpected json.Marshal error")
	responseBytes, err := app.ProcessRequestBody(requestbytes)
	assert.Equal(t, err, tests.ErrBypassTest, "Unexpected ProcessRequestBody error")
	assert.Empty(t, responseBytes, "Unexprected response bytes")
}

func TestProcessRequestTaskMessage(t *testing.T) {
	_, storageController, _, app := getTestApp()
	request := TelegramRequest{}
	request.Message.From.ID = 1126
	request.Message.Chat.ID = 1126
	request.Message.From.FirstName = "TestUser"
	request.Message.Text = getActualDailyTaskCommand
	todayTaskID := common.GetDateIDForNow()
	storageController.tasks[todayTaskID] = &common.BotLeetCodeTask{
		DateID: todayTaskID,
		LeetCodeTask: leetcodeclient.LeetCodeTask{
			QuestionID: 1445,
			ItemID:     6534,
			Title:      "Test title",
			Content:    "Test content",
			Hints:      []string{"first hint", "Second Hint"},
			Difficulty: "Easy",
		},
	}
	requestbytes, err := json.Marshal(request)
	assert.Nil(t, err, "Unexpected json.Marshal error")
	responseBytes, err := app.ProcessRequestBody(requestbytes)
	assert.Nil(t, err, "Unexpected ProcessRequestBody error")
	expectedResponse := fmt.Sprintf("{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":1126,\"text\":\"\\u003cstrong\\u003eTest title\\u003c/strong\\u003e\\n\\nTest content\",\"reply_markup\":\"{\\\"inline_keyboard\\\":[[{\\\"text\\\":\\\"See task on LeetCode website\\\",\\\"url\\\":\\\"https://leetcode.com/explore/item/6534\\\"}],[{\\\"text\\\":\\\"Hint 1\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"%d\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":0}\\\"},{\\\"text\\\":\\\"Hint 2\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"%d\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":1}\\\"}],[{\\\"text\\\":\\\"Hint: Get the difficulty of the task\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"%d\\\\\\\",\\\\\\\"callback_type\\\\\\\":1,\\\\\\\"hint\\\\\\\":0}\\\"}]]}\"}", todayTaskID, todayTaskID, todayTaskID)
	assert.Equal(t, responseBytes, []byte(expectedResponse), "Unexprected response bytes")
}

func TestProcessRequestTaskMessageError(t *testing.T) {
	_, storageController, lcClient, app := getTestApp()
	request := TelegramRequest{}
	request.Message.From.ID = 1126
	request.Message.Chat.ID = 1126
	request.Message.From.FirstName = "TestUser"
	request.Message.Text = getActualDailyTaskCommand
	todayTaskID := common.GetDateIDForNow()
	delete(storageController.tasks, todayTaskID)
	lcClient.On(
		"GetDailyTask",
		todayTaskID,
	).Return(
		leetcodeclient.LeetCodeTask{},
		tests.ErrBypassTest,
	)
	requestbytes, err := json.Marshal(request)
	assert.Nil(t, err, "Unexpected json.Marshal error")
	responseBytes, err := app.ProcessRequestBody(requestbytes)
	assert.Equal(t, err, tests.ErrBypassTest, "Unexpected ProcessRequestBody error")
	assert.Empty(t, responseBytes, "Unexprected response bytes")
}

func TestProcessRequestTaskHint(t *testing.T) {
	_, storageController, _, app := getTestApp()
	todayTaskID := common.GetDateIDForNow()
	storageController.tasks[todayTaskID] = &common.BotLeetCodeTask{
		DateID: todayTaskID,
		LeetCodeTask: leetcodeclient.LeetCodeTask{
			QuestionID: 1445,
			ItemID:     6534,
			Title:      "Test title",
			Content:    "Test content",
			Hints:      []string{"first hint", "Second Hint"},
			Difficulty: "Easy",
		},
	}
	request := TelegramRequest{}
	request.CallbackQuery.From.ID = 1126
	data, err := common.GetMarshalledCallbackData(todayTaskID, 1, common.HintReuqest)
	assert.Nil(t, err, "Unexpected GetMarshalledCallbackData error")
	request.CallbackQuery.Data = data
	requestbytes, err := json.Marshal(request)
	assert.Nil(t, err, "Unexpected json.Marshal error")
	responseBytes, err := app.ProcessRequestBody(requestbytes)
	assert.Nil(t, err, "Unexpected ProcessRequestBody error")
	expectedResponse := "{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":1126,\"text\":\"Hint #2: Second Hint\",\"reply_markup\":\"\"}"
	assert.Equal(t, responseBytes, []byte(expectedResponse), "Unexprected response bytes")
}

func TestProcessRequestTaskHintMoreThenExists(t *testing.T) {
	_, storageController, _, app := getTestApp()
	taskID := uint64(20210929)
	storageController.tasks[taskID] = &common.BotLeetCodeTask{
		DateID: taskID,
		LeetCodeTask: leetcodeclient.LeetCodeTask{
			QuestionID: 1445,
			ItemID:     6534,
			Title:      "Test title",
			Content:    "Test content",
			Hints:      []string{"first hint", "Second Hint"},
			Difficulty: "Easy",
		},
	}
	request := TelegramRequest{}
	request.CallbackQuery.From.ID = 1126
	data, err := common.GetMarshalledCallbackData(taskID, 2, common.HintReuqest)
	assert.Nil(t, err, "Unexpected GetMarshalledCallbackData error")
	request.CallbackQuery.Data = data
	requestbytes, err := json.Marshal(request)
	assert.Nil(t, err, "Unexpected json.Marshal error")
	responseBytes, err := app.ProcessRequestBody(requestbytes)
	assert.Nil(t, err, "Unexpected ProcessRequestBody error")
	expectedResponse := "{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":1126,\"text\":\"There is no such hint for task 20210929\",\"reply_markup\":\"\"}"
	assert.Equal(t, responseBytes, []byte(expectedResponse), "Unexprected response bytes")
}

func TestProcessRequestTaskHintError(t *testing.T) {
	_, storageController, _, app := getTestApp()
	todayTaskID := common.GetDateIDForNow()
	storageController.failedTaskID = todayTaskID
	request := TelegramRequest{}
	request.CallbackQuery.From.ID = 1126
	data, err := common.GetMarshalledCallbackData(todayTaskID, 2, common.HintReuqest)
	assert.Nil(t, err, "Unexpected GetMarshalledCallbackData error")
	request.CallbackQuery.Data = data
	requestbytes, err := json.Marshal(request)
	assert.Nil(t, err, "Unexpected json.Marshal error")
	responseBytes, err := app.ProcessRequestBody(requestbytes)
	assert.Nil(t, err, "Unexpected ProcessRequestBody error")
	expectedResponse := "{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":1126,\"text\":\"Something went completely wrong\",\"reply_markup\":\"\"}"
	assert.Equal(t, responseBytes, []byte(expectedResponse), "Unexprected response bytes")
}

func TestProcessRequestTaskHintNoTask(t *testing.T) {
	_, storageController, _, app := getTestApp()
	todayTaskID := common.GetDateIDForNow()
	delete(storageController.tasks, todayTaskID)
	request := TelegramRequest{}
	request.CallbackQuery.From.ID = 1126
	data, err := common.GetMarshalledCallbackData(todayTaskID, 2, common.HintReuqest)
	assert.Nil(t, err, "Unexpected GetMarshalledCallbackData error")
	request.CallbackQuery.Data = data
	requestbytes, err := json.Marshal(request)
	assert.Nil(t, err, "Unexpected json.Marshal error")
	responseBytes, err := app.ProcessRequestBody(requestbytes)
	assert.Nil(t, err, "Unexpected ProcessRequestBody error")
	expectedResponse := "{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":1126,\"text\":\"There is not such dailyTask. Try another breach ;)\",\"reply_markup\":\"\"}"
	assert.Equal(t, responseBytes, []byte(expectedResponse), "Unexprected response bytes")
}

func TestProcessRequestTaskDifficulty(t *testing.T) {
	_, storageController, _, app := getTestApp()
	todayTaskID := common.GetDateIDForNow()
	storageController.tasks[todayTaskID] = &common.BotLeetCodeTask{
		DateID: todayTaskID,
		LeetCodeTask: leetcodeclient.LeetCodeTask{
			QuestionID: 1445,
			ItemID:     6534,
			Title:      "Test title",
			Content:    "Test content",
			Hints:      []string{"first hint", "Second Hint"},
			Difficulty: "Easy",
		},
	}
	request := TelegramRequest{}
	request.CallbackQuery.From.ID = 1126
	data, err := common.GetMarshalledCallbackData(todayTaskID, 2, common.DifficultyRequest)
	assert.Nil(t, err, "Unexpected GetMarshalledCallbackData error")
	request.CallbackQuery.Data = data
	requestbytes, err := json.Marshal(request)
	assert.Nil(t, err, "Unexpected json.Marshal error")
	responseBytes, err := app.ProcessRequestBody(requestbytes)
	assert.Nil(t, err, "Unexpected ProcessRequestBody error")
	expectedResponse := "{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":1126,\"text\":\"Task difficulty: Easy\",\"reply_markup\":\"\"}"
	assert.Equal(t, responseBytes, []byte(expectedResponse), "Unexprected response bytes")
}

func TestProcessRequestBrokenBody(t *testing.T) {
	_, _, _, app := getTestApp()

	requestbytes := []byte("{\"}")
	responseBytes, err := app.ProcessRequestBody(requestbytes)
	assert.Equal(t, err.(*json.SyntaxError).Offset, int64(3), "Unexpected ProcessRequestBody error")
	assert.Empty(t, responseBytes, "Unexprected response bytes")
}

func TestProcessRequestBrokenData(t *testing.T) {
	_, storageController, _, app := getTestApp()
	todayTaskID := common.GetDateIDForNow()
	storageController.tasks[todayTaskID] = &common.BotLeetCodeTask{
		DateID: todayTaskID,
		LeetCodeTask: leetcodeclient.LeetCodeTask{
			QuestionID: 1445,
			ItemID:     6534,
			Title:      "Test title",
			Content:    "Test content",
			Hints:      []string{"first hint", "Second Hint"},
			Difficulty: "Easy",
		},
	}
	request := TelegramRequest{}
	request.CallbackQuery.From.ID = 1126
	request.CallbackQuery.Data = "{\"}"
	requestbytes, err := json.Marshal(request)
	assert.Nil(t, err, "Unexpected json.Marshal error")
	responseBytes, err := app.ProcessRequestBody(requestbytes)
	assert.Equal(t, err.(*json.SyntaxError).Offset, int64(3), "Unexpected ProcessRequestBody error")
	assert.Empty(t, responseBytes, "Unexprected response bytes")
}

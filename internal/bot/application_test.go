package bot

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dartkron/leetcodeBot/v2/internal/common"
	"github.com/dartkron/leetcodeBot/v2/internal/storage"
	"github.com/dartkron/leetcodeBot/v2/pkg/leetcodeclient"
)

var ErrWorngContentType = errors.New("content type should be \"application/json\"")

var ErrBypassTest error = errors.New("test bypass error")

type httpResponse struct {
	response *http.Response
	err      error
}

var MockHTTPPostRequests []string

var MockHTTPPostknownRequests map[string]httpResponse

func MockHTTPPost(url string, contentType string, body io.Reader) (*http.Response, error) {
	request, err := io.ReadAll(body)
	if err != nil {
		return &http.Response{}, err
	}
	requestString := string(request)
	MockHTTPPostRequests = append(MockHTTPPostRequests, requestString)
	if contentType != "application/json" {
		return &http.Response{}, ErrWorngContentType
	}
	if response, ok := MockHTTPPostknownRequests[requestString]; ok {
		return response.response, response.err
	}
	return &http.Response{StatusCode: http.StatusInternalServerError}, http.ErrBodyNotAllowed
}

type MockLeetcodeClient struct {
	tasks        map[uint64]leetcodeclient.LeetCodeTask
	callsJournal []string
	failedTaskID uint64
}

func (m *MockLeetcodeClient) GetDailyTaskItemID(date time.Time) (string, error) {
	m.callsJournal = append(m.callsJournal, fmt.Sprintf("GetDailyTaskItemID %d", common.GetDateID(date)))
	return "1", nil
}

func (m *MockLeetcodeClient) GetQuestionDetailsByItemID(itemID string) (leetcodeclient.LeetCodeTask, error) {
	m.callsJournal = append(m.callsJournal, fmt.Sprintf("GetQuestionDetailsByItemID %s", itemID))
	return leetcodeclient.LeetCodeTask{}, nil
}

func (m *MockLeetcodeClient) GetDailyTask(date time.Time) (leetcodeclient.LeetCodeTask, error) {
	m.callsJournal = append(m.callsJournal, fmt.Sprintf("GetDailyTask %d", common.GetDateID(date)))
	dateID := common.GetDateID(date)
	if dateID == m.failedTaskID {
		return leetcodeclient.LeetCodeTask{}, ErrBypassTest
	}
	if resp, ok := m.tasks[dateID]; ok {
		return resp, nil
	}
	return leetcodeclient.LeetCodeTask{}, errors.New("Not such task")
}

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
		return common.BotLeetCodeTask{}, ErrBypassTest
	}
	if task, ok := controller.tasks[dateID]; ok {
		return *task, nil
	}
	return common.BotLeetCodeTask{}, storage.ErrNoSuchTask
}

func (controller *MockStorageController) SaveTask(task common.BotLeetCodeTask) error {
	controller.callsJournal = append(controller.callsJournal, fmt.Sprintf("SaveTask %d", task.DateID))
	if task.DateID == controller.failedTaskID {
		return ErrBypassTest
	}
	controller.tasks[task.DateID] = &task
	return nil
}

func (controller *MockStorageController) SubscribeUser(user common.User) error {
	controller.callsJournal = append(controller.callsJournal, fmt.Sprintf("SubscribeUser %d", user.ID))
	if user.ID == controller.failedUserID {
		return ErrBypassTest
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
		return ErrBypassTest
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
		return []common.User{}, ErrBypassTest
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
	if err != nil {
		t.Errorf("Got error from GetMainKeyboard %q", err)
	}
	if !reflect.DeepEqual(waitKeyboard, keyboard) {
		t.Errorf("Returned keyboard:\n%q\nnot equal with awaited:\n%q\n", keyboard, waitKeyboard)
	}
}

func getTestApp() (*MockStorageController, *MockLeetcodeClient, *Application) {
	MockHTTPPostRequests = []string{}
	MockHTTPPostknownRequests = map[string]httpResponse{
		"{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":1120,\"text\":\"\\u003cstrong\\u003eTest title\\u003c/strong\\u003e\\n\\nTest content\",\"reply_markup\":\"{\\\"inline_keyboard\\\":[[{\\\"text\\\":\\\"See task on LeetCode website\\\",\\\"url\\\":\\\"https://leetcode.com/explore/item/6534\\\"}],[{\\\"text\\\":\\\"Hint 1\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":0}\\\"},{\\\"text\\\":\\\"Hint 2\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":1}\\\"}],[{\\\"text\\\":\\\"Hint: Get the difficulty of the task\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":1,\\\\\\\"hint\\\\\\\":0}\\\"}]]}\"}": {
			response: &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("1120"))},
		},
		"{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":1126,\"text\":\"\\u003cstrong\\u003eTest title\\u003c/strong\\u003e\\n\\nTest content\",\"reply_markup\":\"{\\\"inline_keyboard\\\":[[{\\\"text\\\":\\\"See task on LeetCode website\\\",\\\"url\\\":\\\"https://leetcode.com/explore/item/6534\\\"}],[{\\\"text\\\":\\\"Hint 1\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":0}\\\"},{\\\"text\\\":\\\"Hint 2\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":1}\\\"}],[{\\\"text\\\":\\\"Hint: Get the difficulty of the task\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":1,\\\\\\\"hint\\\\\\\":0}\\\"}]]}\"}": {
			response: &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("1126"))},
		},
		"{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":1127,\"text\":\"\\u003cstrong\\u003eTest title\\u003c/strong\\u003e\\n\\nTest content\",\"reply_markup\":\"{\\\"inline_keyboard\\\":[[{\\\"text\\\":\\\"See task on LeetCode website\\\",\\\"url\\\":\\\"https://leetcode.com/explore/item/6534\\\"}],[{\\\"text\\\":\\\"Hint 1\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":0}\\\"},{\\\"text\\\":\\\"Hint 2\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":1}\\\"}],[{\\\"text\\\":\\\"Hint: Get the difficulty of the task\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":1,\\\\\\\"hint\\\\\\\":0}\\\"}]]}\"}": {
			response: &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("1127"))},
		},
	}
	app := NewApplication()
	leetcodeClient := &MockLeetcodeClient{
		tasks: map[uint64]leetcodeclient.LeetCodeTask{},
	}
	app.leetcodeAPIClient = leetcodeClient
	app.PostFunc = MockHTTPPost
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
	app.storageController = storageController
	return storageController, leetcodeClient, app
}

func TestSendDailyTaskToSubscribedUsers(t *testing.T) {
	storageController, _, app := getTestApp()
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
	if err != nil {
		t.Errorf("Got unexprected error:%q", err)
	}
}

func TestSendDailyTaskToSubscribedUsersError(t *testing.T) {
	storageController, _, app := getTestApp()
	loc, _ := time.LoadLocation("US/Pacific")
	now := time.Now().In(loc)
	taskDateID := common.GetDateID(now)
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
	if err != nil {
		t.Errorf("Got unexprected error:%q", err)
	}

	count := map[string]int{}
	for _, s := range MockHTTPPostRequests {
		count[s]++
	}

	awaited := map[string]int{
		"{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":1120,\"text\":\"\\u003cstrong\\u003eTest title\\u003c/strong\\u003e\\n\\nTest content\",\"reply_markup\":\"{\\\"inline_keyboard\\\":[[{\\\"text\\\":\\\"See task on LeetCode website\\\",\\\"url\\\":\\\"https://leetcode.com/explore/item/6534\\\"}],[{\\\"text\\\":\\\"Hint 1\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":0}\\\"},{\\\"text\\\":\\\"Hint 2\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":1}\\\"}],[{\\\"text\\\":\\\"Hint: Get the difficulty of the task\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":1,\\\\\\\"hint\\\\\\\":0}\\\"}]]}\"}": 1,
		"{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":1126,\"text\":\"\\u003cstrong\\u003eTest title\\u003c/strong\\u003e\\n\\nTest content\",\"reply_markup\":\"{\\\"inline_keyboard\\\":[[{\\\"text\\\":\\\"See task on LeetCode website\\\",\\\"url\\\":\\\"https://leetcode.com/explore/item/6534\\\"}],[{\\\"text\\\":\\\"Hint 1\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":0}\\\"},{\\\"text\\\":\\\"Hint 2\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":1}\\\"}],[{\\\"text\\\":\\\"Hint: Get the difficulty of the task\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":1,\\\\\\\"hint\\\\\\\":0}\\\"}]]}\"}": 1,
		"{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":1127,\"text\":\"\\u003cstrong\\u003eTest title\\u003c/strong\\u003e\\n\\nTest content\",\"reply_markup\":\"{\\\"inline_keyboard\\\":[[{\\\"text\\\":\\\"See task on LeetCode website\\\",\\\"url\\\":\\\"https://leetcode.com/explore/item/6534\\\"}],[{\\\"text\\\":\\\"Hint 1\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":0}\\\"},{\\\"text\\\":\\\"Hint 2\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":1}\\\"}],[{\\\"text\\\":\\\"Hint: Get the difficulty of the task\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":1,\\\\\\\"hint\\\\\\\":0}\\\"}]]}\"}": 3,
		"{\"method\":\"sendMessage\",\"parse_mode\":\"HTML\",\"chat_id\":1121,\"text\":\"\\u003cstrong\\u003eTest title\\u003c/strong\\u003e\\n\\nTest content\",\"reply_markup\":\"{\\\"inline_keyboard\\\":[[{\\\"text\\\":\\\"See task on LeetCode website\\\",\\\"url\\\":\\\"https://leetcode.com/explore/item/6534\\\"}],[{\\\"text\\\":\\\"Hint 1\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":0}\\\"},{\\\"text\\\":\\\"Hint 2\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":0,\\\\\\\"hint\\\\\\\":1}\\\"}],[{\\\"text\\\":\\\"Hint: Get the difficulty of the task\\\",\\\"callback_data\\\":\\\"{\\\\\\\"dateID\\\\\\\":\\\\\\\"20210927\\\\\\\",\\\\\\\"callback_type\\\\\\\":1,\\\\\\\"hint\\\\\\\":0}\\\"}]]}\"}": 3,
	}
	if !reflect.DeepEqual(count, awaited) {
		t.Errorf("awaited requests to TelegramAPI incorrect:\n%v\ncorrect one:\n%v\n", count, awaited)
	}
}

func TestSendDailyTaskToSubscribedUsersWithoutTasks(t *testing.T) {
	storageController, leetcodeClient, app := getTestApp()
	taskDateID := common.GetDateIDForNow()
	storageController.failedTaskID = taskDateID
	leetcodeClient.failedTaskID = taskDateID

	err := app.SendDailyTaskToSubscribedUsers()
	if err != ErrBypassTest {
		t.Errorf("Got unexprected error:%q", err)
	}
	awaitedCalls := []string{fmt.Sprintf("GetDailyTask %d", taskDateID)}
	if !reflect.DeepEqual(leetcodeClient.callsJournal, awaitedCalls) {
		t.Errorf("Wrong calls list of leetcodeclient:\n%q\nshould be:%q", leetcodeClient.callsJournal, awaitedCalls)
	}

	count := map[string]int{}
	for _, s := range MockHTTPPostRequests {
		fmt.Println("S: ", s)
		count[s]++
	}

	awaited := map[string]int{}
	if !reflect.DeepEqual(count, awaited) {
		t.Errorf("awaited requests to TelegramAPI incorrect:\n%v\ncorrect one:\n%v\n", count, awaited)
	}
}

func TestGetTodayTaskFromAllPossibleSourcesFromClient(t *testing.T) {
	storageController, leetcodeClient, app := getTestApp()
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
	leetcodeClient.tasks[todayDateID] = testTask.LeetCodeTask
	task, err := app.GetTodayTaskFromAllPossibleSources()
	if err != nil {
		t.Errorf("Got unexprected error:%q", err)
	}

	if !reflect.DeepEqual(task, testTask) {
		t.Errorf("Returned task:\n%q\nnot equal to expected:\n%q", task, testTask)
	}
	awaitedCalls := []string{fmt.Sprintf("GetTask %d", todayDateID), fmt.Sprintf("SaveTask %d", todayDateID)}
	if !reflect.DeepEqual(storageController.callsJournal, awaitedCalls) {
		t.Errorf("Storage controller calls:\n%q\nisn't equal to expected:\n%q\n", storageController.callsJournal, awaitedCalls)
	}

	awaitedCalls = []string{fmt.Sprintf("GetDailyTask %d", todayDateID)}
	if !reflect.DeepEqual(leetcodeClient.callsJournal, awaitedCalls) {
		t.Errorf("Wrong calls list of leetcodeclient:\n%q\nshould be:%q", leetcodeClient.callsJournal, awaitedCalls)
	}

}

func TestGetTodayTaskFromAllPossibleSourcesFromClientWithErrorFromStorage(t *testing.T) {
	storageController, leetcodeClient, app := getTestApp()
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
	leetcodeClient.tasks[todayDateID] = testTask.LeetCodeTask
	storageController.failedTaskID = todayDateID
	task, err := app.GetTodayTaskFromAllPossibleSources()
	if err != ErrBypassTest {
		t.Errorf("Got unexprected error:%q", err)
	}

	if !reflect.DeepEqual(task, testTask) {
		t.Errorf("Returned task:\n%q\nnot equal to expected:\n%q", task, testTask)
	}
	awaitedCalls := []string{fmt.Sprintf("GetTask %d", todayDateID), fmt.Sprintf("SaveTask %d", todayDateID)}
	if !reflect.DeepEqual(storageController.callsJournal, awaitedCalls) {
		t.Errorf("Storage controller calls:\n%q\nisn't equal to expected:\n%q\n", storageController.callsJournal, awaitedCalls)
	}

	awaitedCalls = []string{fmt.Sprintf("GetDailyTask %d", todayDateID)}
	if !reflect.DeepEqual(leetcodeClient.callsJournal, awaitedCalls) {
		t.Errorf("Wrong calls list of leetcodeclient:\n%q\nshould be:%q", leetcodeClient.callsJournal, awaitedCalls)
	}

}

func TestGetTodayTaskFromAllPossibleSourcesFromStorage(t *testing.T) {
	storageController, leetcodeClient, app := getTestApp()
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
	if err != nil {
		t.Errorf("Got unexprected error:%q", err)
	}

	if !reflect.DeepEqual(task, testTask) {
		t.Errorf("Returned task:\n%q\nnot equal to expected:\n%q", task, testTask)
	}

	awaitedCalls := []string{fmt.Sprintf("GetTask %d", todayDateID)}
	if !reflect.DeepEqual(storageController.callsJournal, awaitedCalls) {
		t.Errorf("Storage controller calls:\n%q\nisn't equal to expected:\n%q\n", storageController.callsJournal, awaitedCalls)
	}

	if len(leetcodeClient.callsJournal) != 0 {
		t.Errorf("Wrong calls list of leetcodeclient:\n%q\nshould be empty", leetcodeClient.callsJournal)
	}
}

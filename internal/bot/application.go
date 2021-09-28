package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/dartkron/leetcodeBot/v2/internal/common"
	"github.com/dartkron/leetcodeBot/v2/internal/storage"
	"github.com/dartkron/leetcodeBot/v2/pkg/leetcodeclient"
)

const (
	helpMessage = `You command "%s" isn't recognized =(
List of available commands:
/getDailyTask — get actual dailyTask
/Subscribe — start automatically sending of daily tasks
/Unsubscribe — stop automatically sending of daily tasks`

	unsubscribedMessage = `%s, you have <strong>sucessfully unsubscribed</strong>. You'll not automatically recieve daily tasks.
If you've found this bot useless and have ideas of possible improvements, please, add them to https://github.com/dartkron/leetcodeBot/issues`

	alreadyUnsubscribedMessage = "%s, you were <strong>not subscribed</strong>. No additonal actions required."
	subcribedMessage           = "%s, you have <strong>sucessfully subscribed</strong>. You'll automatically recieve daily tasks when they appear."
	alreadySubscribedMessage   = "%s, you have <strong>already subscribed</strong> for daily updates, nothing to do."
	getActualDailyTaskCommand  = "Get actual daily task"
	subscribeCommand           = "Subscribe"
	unsubscribeCommand         = "Unsubscribe"
	telegramAPIURL             = "https://api.telegram.org/bot%s/sendMessage"
)

// TelegramResponse is a short representation of fileds supported by Telegram
type TelegramResponse struct {
	Method      string `json:"method"`
	ParseMode   string `json:"parse_mode"`
	ChatID      uint64 `json:"chat_id"`
	Text        string `json:"text"`
	ReplyMarkup string `json:"reply_markup"`
}

// TelegramRequest represents part of possible fields in Telegram request JSON
type TelegramRequest struct {
	CallbackQuery struct {
		Data string `json:"data"`
		From struct {
			ID uint64 `json:"id"`
		} `json:"from"`
	} `json:"callback_query"`
	Message struct {
		Text string `json:"text"`
		Chat struct {
			ID uint64 `json:"id"`
		} `json:"chat"`
		From struct {
			ID        uint64 `json:"id"`
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		}
	} `json:"message"`
}

// KeyboardDef is a representation of Telegram keyboard, don't confuse it with inline keyboard
// Two dimensonal slice of Key: first dimension levels of keys and second is particular keys.
// So you can have different amount of keys on every level
type KeyboardDef struct {
	Keyboard              [][]Key `json:"keyboard"`
	InputFieldPlaceholder string  `json:"input_field_placeholder"`
	ResizeKeyboard        bool    `json:"resize_keyboard"`
}

// Key is one keyboard key
type Key struct {
	Text string `json:"text"`
}

// NewTelegramResponse TelegramResponse constructor with filling basic fileds
func NewTelegramResponse() *TelegramResponse {
	return &TelegramResponse{
		Method:    "sendMessage",
		ParseMode: "HTML",
	}
}

// Application holder for dependicies for easy injection
type Application struct {
	storageController storage.Controller
	leetcodeAPIClient leetcodeclient.LeetcodeClient
	PostFunc          func(string, string, io.Reader) (*http.Response, error)
}

// ProcessRequestBody parse body json and route request to handlers
func (app *Application) ProcessRequestBody(body []byte) ([]byte, error) {
	telegramRequest := TelegramRequest{}
	err := json.Unmarshal(body, &telegramRequest)
	if err != nil {
		return []byte{}, err
	}
	var response *TelegramResponse
	if len(telegramRequest.CallbackQuery.Data) != 0 {
		response, err = app.processCallback(telegramRequest)
	} else {
		response, err = app.processMessage(telegramRequest)
	}
	if err != nil {
		return []byte{}, err
	}
	bytes, err := json.Marshal(response)
	return bytes, err

}

func (app *Application) processCallback(request TelegramRequest) (*TelegramResponse, error) {
	response := NewTelegramResponse()
	callback := common.CallbackData{}
	err := json.Unmarshal([]byte(request.CallbackQuery.Data), &callback)
	if err != nil {
		return response, err
	}
	response.ChatID = request.CallbackQuery.From.ID
	task, err := app.storageController.GetTask(callback.DateID)
	if err != nil {
		if err == storage.ErrNoSuchTask {
			response.Text = "There is not such dailyTask. Try another breach ;)"
		} else {
			response.Text = "Something went completely wrong"
		}
		return response, nil
	}
	if callback.Type == common.HintReuqest {
		if callback.Hint > len(task.Hints)-1 {
			response.Text = fmt.Sprintf("There is no such hint for task %d", callback.DateID)
			return response, nil
		}
		response.Text = fmt.Sprintf("Hint #%d: %s", callback.Hint+1, task.Hints[callback.Hint])
	} else if callback.Type == common.DifficultyRequest {
		response.Text = fmt.Sprintf("Task difficulty: %s", task.Difficulty)
	}
	return response, nil
}

func (app *Application) processMessage(request TelegramRequest) (*TelegramResponse, error) {
	command := request.Message.Text
	response := NewTelegramResponse()
	response.ChatID = request.Message.Chat.ID
	keyboard, err := GetMainKeyboard()
	if err != nil {
		return response, err
	}
	response.ReplyMarkup = keyboard
	switch command {
	case getActualDailyTaskCommand, "/getDailyTask":
		err = app.getTaskAction(response)
	case subscribeCommand, "/Subscribe":
		err = app.subscribeAction(&request, response)
	case unsubscribeCommand, "/Unsubscribe":
		err = app.unsubscribeAction(&request, response)
	default:
		response.Text = fmt.Sprintf(helpMessage, command)
	}
	return response, err

}

func (app *Application) getTaskAction(response *TelegramResponse) error {
	task, err := app.GetTodayTaskFromAllPossibleSources()
	if err != nil {
		return err
	}
	response.Text = task.GetTaskText()
	response.ReplyMarkup = task.GetInlineKeyboard()
	return nil
}

func (app *Application) subscribeAction(request *TelegramRequest, response *TelegramResponse) error {
	user := common.User{
		ID:        request.Message.From.ID,
		ChatID:    request.Message.Chat.ID,
		Username:  request.Message.From.Username,
		FirstName: request.Message.From.FirstName,
		LastName:  request.Message.From.LastName,
	}
	err := app.storageController.SubscribeUser(user)
	if err == storage.ErrUserAlreadySubscribed {
		response.Text = fmt.Sprintf(alreadySubscribedMessage, user.FirstName)

	} else if err != nil {
		return err
	} else {
		response.Text = fmt.Sprintf(subcribedMessage, user.FirstName)
	}
	return nil
}

func (app *Application) unsubscribeAction(request *TelegramRequest, response *TelegramResponse) error {
	err := app.storageController.UnsubscribeUser(request.Message.From.ID)
	if err == storage.ErrUserAlreadyUnsubscribed {
		response.Text = fmt.Sprintf(alreadyUnsubscribedMessage, request.Message.From.FirstName)
	} else if err != nil {
		return err
	} else {
		response.Text = fmt.Sprintf(unsubscribedMessage, request.Message.From.FirstName)
	}
	return nil
}

// GetTodayTaskFromAllPossibleSources is the one func to rule them all
// It's apperared because of necessary to share logic between reminder and bot
func (app *Application) GetTodayTaskFromAllPossibleSources() (common.BotLeetCodeTask, error) {
	now := common.GetDateInRightTimeZone()
	taskDateID := common.GetDateID(now)
	task, err := app.storageController.GetTask(taskDateID)
	if err != nil {
		if err != storage.ErrNoSuchTask {
			fmt.Println("Got DB error:", err)
			fmt.Println("Fallback to Leetcode API")
		}
		lcTask, err := app.leetcodeAPIClient.GetDailyTask(now)
		if err != nil {
			return common.BotLeetCodeTask{}, err
		}
		task = common.BotLeetCodeTask{
			LeetCodeTask: lcTask,
			DateID:       taskDateID,
		}
		task.FixTagsAndImages()
		err = app.storageController.SaveTask(task)
		if err != nil {
			return task, err
		}
		return task, nil
	}
	return task, nil
}

// SendMessage sends message to particular user.
func (app *Application) SendMessage(requestBody []byte) error {
	tries := 0
	for tries < 3 {
		tries++
		buf := bytes.NewBuffer(requestBody)
		resp, err := app.PostFunc(fmt.Sprintf(telegramAPIURL, os.Getenv("SENDING_TOKEN")), "application/json", buf)
		if err != nil || resp.StatusCode >= 400 {
			if tries == 3 {
				return err
			}
			continue
		}
		// TODO: should we see what's inside the response?
		_, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		break
	}
	return nil
}

// SendDailyTaskToSubscribedUsers get subscribed users and send notifications with daily task to them
func (app *Application) SendDailyTaskToSubscribedUsers() error {
	usersSlice, err := app.storageController.GetSubscribedUsers()
	if err != nil {
		return err
	}

	telegramRequest := NewTelegramResponse()
	err = app.getTaskAction(telegramRequest)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, user := range usersSlice {
		telegramRequest.ChatID = user.ID
		bytes, err := json.Marshal(telegramRequest)
		if err != nil {
			return err
		}
		wg.Add(1)
		go func(bytes []byte, userID uint64) {
			err := app.SendMessage(bytes)
			if err != nil {
				fmt.Printf("Failed to send message to user %d, with error: %s\n", userID, err.Error())
			}
			wg.Done()
		}(bytes, user.ID)
	}
	wg.Wait()
	return nil
}

// GetMainKeyboard returns marshaled json for the main keyboard
func GetMainKeyboard() (string, error) {
	keyboard := KeyboardDef{
		ResizeKeyboard:        true,
		InputFieldPlaceholder: "Please, use buttons below:",
		Keyboard: [][]Key{
			{{Text: getActualDailyTaskCommand}},
			{{Text: subscribeCommand}, {Text: unsubscribeCommand}},
		},
	}
	bytes, err := json.Marshal(keyboard)
	if err != nil {
		return "", err
	}
	return string(bytes), err
}

// NewApplication Application constructor with default values
func NewApplication() *Application {
	return &Application{
		storageController: storage.NewYDBandFileCacheController(),
		leetcodeAPIClient: leetcodeclient.NewLeetCodeGraphQlClient(),
	}
}

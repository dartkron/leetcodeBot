package common

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	leetcodeclient "github.com/dartkron/leetcodeBot/v2/pkg/leetcodeclient"
)

const (
	easy = iota
	medium
	hard
	notSet
)

type inlineButton struct {
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
}

// CallbackType means type of CallbackData recieved from user or marshalled into inline buttons
type CallbackType uint8

const (
	// HintReuqest means callback is about hint, not difficulty
	HintReuqest CallbackType = iota
	// DifficultyRequest means that callback require only difficulty
	DifficultyRequest
)

// CallbackData stuct for unmarshal JSON in callbackRequests and marshal in inlineKeyboard building
type CallbackData struct {
	DateID uint64       `json:"dateID,string,omitempty"`
	Type   CallbackType `json:"callback_type"`
	Hint   int          `json:"hint"`
}

// BotLeetCodeTask is iternal LeetcodeTask representation with bot-related info: DateID
type BotLeetCodeTask struct {
	leetcodeclient.LeetCodeTask
	DateID uint64 `json:"dateID,string"`
}

// User struct for the whole application
type User struct {
	ID         uint64
	ChatID     uint64
	Username   string
	FirstName  string
	LastName   string
	Subscribed bool
}

// GetTaskText returns task text representation to couple login into class(struct, of course struct)
func (task *BotLeetCodeTask) GetTaskText() string {
	return fmt.Sprintf("<strong>%s</strong>\n\n%s", task.Title, task.Content)
}

func getMarshalledCallbackData(dateID uint64, hintID int, dataType CallbackType) (string, error) {
	callbackData := CallbackData{DateID: dateID, Type: dataType}
	if dataType == HintReuqest {
		callbackData.Hint = int(hintID)
	}
	callbackDataMarshaled, err := json.Marshal(callbackData)
	return string(callbackDataMarshaled), err
}

// GetInlineKeyboard returns inline keyboard for task marshalled into JSON string
func (task *BotLeetCodeTask) GetInlineKeyboard() string {
	listOfHints := [][]inlineButton{{}}
	level := 0
	for i := range task.Hints {
		callbackData, err := getMarshalledCallbackData(task.DateID, i, HintReuqest)
		if err != nil {
			fmt.Println("Got error on marshalling callback data:", err)
		}
		listOfHints[level] = append(
			listOfHints[level],
			inlineButton{
				Text:         fmt.Sprintf("Hint %d", i+1),
				CallbackData: string(callbackData),
			})
		// 5 hints in the row
		if (i+1)%5 == 0 {
			level++
			listOfHints = append(listOfHints, []inlineButton{})
		}
	}
	listOfHints = append([][]inlineButton{
		{
			{
				Text: "See task on LeetCode website",
				URL:  fmt.Sprintf("https://leetcode.com/explore/item/%d", task.ItemID),
			},
		},
	}, listOfHints...)

	// Append difficulty hint to task inline keyboard
	getDifficultyCallbackData, err := getMarshalledCallbackData(task.DateID, 0, DifficultyRequest)
	if err != nil {
		fmt.Println("Got error on marshalling callback data:", err)
	}
	listOfHints = append(
		listOfHints,
		[]inlineButton{
			{
				Text:         "Hint: Get the difficulty of the task",
				CallbackData: getDifficultyCallbackData,
			},
		},
	)

	inlineKeyboard, err := json.Marshal(map[string][][]inlineButton{"inline_keyboard": listOfHints})
	if err != nil {
		fmt.Println("Error during marshall inlineKeyboard:", err)
	}
	return string(inlineKeyboard)
}

// FixTagsAndImages fix Title and Content for tags unsupported by Telegram. Also process images into links.
func (task *BotLeetCodeTask) FixTagsAndImages() {
	task.Title = RemoveUnsupportedTags(task.Title)
	task.Content = ReplaceImgTagWithA(RemoveUnsupportedTags(task.Content))
	for i := range task.Hints {
		task.Hints[i] = ReplaceImgTagWithA(RemoveUnsupportedTags(task.Hints[i]))
	}
}

// GetDifficultyNum return uint8 value for task difficulty
func (task *BotLeetCodeTask) GetDifficultyNum() uint8 {
	switch task.Difficulty {
	case "Easy":
		return easy
	case "Medium":
		return medium
	case "Hard":
		return hard
	default:
		return notSet
	}
}

// SetDifficultyFromNum set task difficulty from uint8 value
func (task *BotLeetCodeTask) SetDifficultyFromNum(difficuty uint8) {
	switch difficuty {
	case easy:
		task.Difficulty = "Easy"
	case medium:
		task.Difficulty = "Medium"
	case hard:
		task.Difficulty = "Hard"
	case notSet:
		task.Difficulty = "Not known"
	}
}

// GetDateID right provider of ID based on task Date. Persistent key.
func GetDateID(date time.Time) uint64 {
	dateID, err := strconv.ParseUint(fmt.Sprintf("%02d%02d%02d", date.Year(), date.Month(), date.Day()), 10, 64)
	if err != nil {
		fmt.Println("Got error on dateID generation:", err)
	}
	return dateID
}

// RemoveUnsupportedTags remove from string all tags unsupported by Telegram and returns fixed string
func RemoveUnsupportedTags(source string) string {
	tagsToReplacement := map[string]string{
		"<p>":    "",
		"</p>":   "",
		"<ul>":   "",
		"</ul>":  "",
		"</li>":  "",
		"</sup>": "",
		"<em>":   "",
		"</em>":  "",
		"\n\n":   "",
		"<br>":   "",
		"</br>":  "",
		"&nbsp;": " ",
		"<sup>":  "**",
		"<sub>":  "(",
		"</sub>": ")",
		"<li>":   " — ",
	}
	for pattern, replacement := range tagsToReplacement {
		source = strings.ReplaceAll(source, pattern, replacement)
	}
	return source
}

// ReplaceImgTagWithA replace <img> tag which is unsupported by Telegram with <a href=https://address.of.img.
func ReplaceImgTagWithA(source string) string {
	imgCount := strings.Count(source, "<img")
	for i := 0; i < imgCount; i++ {
		start := strings.Index(source, "<img")
		end := start + strings.Index(source[start:], "/>") + 2
		urlStart := strings.Index(source[start:end], "src=") + 5
		urlEnd := strings.Index(source[start+urlStart+1:end], "\"")
		url := source[start+urlStart : start+urlStart+1+urlEnd]
		source = source[:start] + fmt.Sprintf("\n<a href=\"%s\">Picture %d</a>", url, i) + source[end:]
	}
	return source
}

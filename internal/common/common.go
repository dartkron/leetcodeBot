package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dartkron/leetcodeBot/v3/pkg/leetcodeclient"
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

// CallbackType means type of CallbackData received from user or marshalled into inline buttons
type CallbackType uint8

const (
	// HintReuqest means callback is about hint, not difficulty
	HintReuqest CallbackType = iota
	// DifficultyRequest means that callback require only difficulty
	DifficultyRequest
)

// ErrClosedContext universal error about closed context
var ErrClosedContext error = errors.New("context closed during execution")

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
	ID          uint64
	ChatID      uint64
	Username    string
	FirstName   string
	LastName    string
	Subscribed  bool
	SendingHour uint8
}

// GetTaskText returns task text representation to couple login into class(struct, of course struct)
func (task *BotLeetCodeTask) GetTaskText() string {
	return fmt.Sprintf("<strong>%s</strong>\n\n%s", task.Title, task.Content)
}

// GetMarshalledCallbackData shotcut to get ready data for callback
func GetMarshalledCallbackData(dateID uint64, hintID int, dataType CallbackType) (string, error) {
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
		callbackData, err := GetMarshalledCallbackData(task.DateID, i, HintReuqest)
		if err != nil {
			fmt.Println("Got error on marshalling callback data:", err)
		}

		// 5 hints in the row
		if len(listOfHints[level]) == 5 {
			level++
			listOfHints = append(listOfHints, []inlineButton{})
		}

		listOfHints[level] = append(
			listOfHints[level],
			inlineButton{
				Text:         fmt.Sprintf("Hint %d", i+1),
				CallbackData: string(callbackData),
			})
	}
	listOfHints = append([][]inlineButton{
		{
			{
				Text: "See task on LeetCode website",
				URL:  fmt.Sprintf("https://leetcode.com/problems/%s", task.TitleSlug),
			},
		},
	}, listOfHints...)

	// Append difficulty hint to task inline keyboard
	getDifficultyCallbackData, err := GetMarshalledCallbackData(task.DateID, 0, DifficultyRequest)
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
	default:
		task.Difficulty = "Not known"
	}
}

// GetDateInRightTimeZone returns time in corrent time zone for Leetcode
func GetDateInRightTimeZone() time.Time {
	return time.Now().UTC()
}

// GetDateIDForNow dateID for current time in correct time zone for Leetcode
func GetDateIDForNow() uint64 {
	return GetDateID(GetDateInRightTimeZone())
}

// GetDateID right provider of ID based on task Date. Persistent key.
func GetDateID(date time.Time) uint64 {
	dateID, err := strconv.ParseUint(fmt.Sprintf("%02d%02d%02d", date.Year(), date.Month(), date.Day()), 10, 64)
	if err != nil {
		fmt.Println("Got error on dateID generation:", err)
	}
	return dateID
}

// RemoveUnsupportedTags shortcut for removing all unsopurted tags and returns fixed string
func RemoveUnsupportedTags(source string) string {
	source = RemoveSimpleUnsupportedTags(source)
	source = ReplaceComplexOpenTags(source, "span", "")
	source = ReplaceComplexOpenTags(source, "div", "")
	return ReplaceComplexOpenTags(source, "font", "")
}

// RemoveSimpleUnsupportedTags remove from string all simple tags unsupported by Telegram and returns fixed string
func RemoveSimpleUnsupportedTags(source string) string {
	tagsToReplacement := map[string]string{
		"<p>&nbsp;</p>": "\n",
		"<p>":           "",
		"</p>":          "",
		"<ol>":          "",
		"</ol>":         "",
		"<ul>":          "",
		"</ul>":         "",
		"</li>":         "",
		"</sup>":        "",
		"<em>":          "",
		"</em>":         "",
		"\n\n":          "\n",
		"<br>":          "",
		"</br>":         "",
		"&nbsp;":        " ",
		"<sup>":         "**",
		"<sub>":         "(",
		"</sub>":        ")",
		"<li>":          " — ",
		"</span>":       "\n",
		"</font>":       "",
		"</div>":        "",
	}
	// In Golang map is truly unordered, so due to different order results were different
	// The sorted order is always the same
	keys := make([]string, len(tagsToReplacement))
	for key := range tagsToReplacement {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		source = strings.ReplaceAll(source, key, tagsToReplacement[key])
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

// ReplaceComplexOpenTags replaces any complex open tag like <tagName * > with pattern
func ReplaceComplexOpenTags(source string, tagName string, pattern string) string {
	startTag := "<" + tagName
	tagCount := strings.Count(source, startTag)
	for i := 0; i < tagCount; i++ {
		start := strings.Index(source, startTag)
		end := start + strings.Index(source[start:], ">") + 1
		source = source[:start] + pattern + source[end:]
	}
	return source
}

package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReplaceImgTagWithA(t *testing.T) {
	tests := map[string]string{
		"test<img src=\"http://mytest.com/test.png\"/>test<img src=\"http://anothermytest.org/pic.jpg\"/>test": "test\n<a href=\"http://mytest.com/test.png\">Picture 0</a>test\n<a href=\"http://anothermytest.org/pic.jpg\">Picture 1</a>test",
	}
	for source, result := range tests {
		calculatedResult := ReplaceImgTagWithA(source)
		assert.Equal(t, calculatedResult, result, "Unexpected ReplaceImgTagWithA transformation")
	}
}

type DateTestCase struct {
	year  int
	month int
	day   int
}

func TestGetDateId(t *testing.T) {
	loc, _ := time.LoadLocation("US/Pacific")
	testData := map[DateTestCase]uint64{
		{1970, 1, 1}:   19700101,
		{2021, 12, 31}: 20211231,
		{2020, 2, 29}:  20200229,
	}
	for dataPart, result := range testData {
		calculatedResult := GetDateID(time.Date(dataPart.year, time.Month(dataPart.month), dataPart.day, 0, 0, 0, 0, loc))
		assert.Equal(t, calculatedResult, result, "Unexpected GetDateID response")
	}
}

func TestRemoveUnsuppotedTags(t *testing.T) {
	cases := map[string]string{
		"test":                   "test",
		"te<p>st":                "test",
		"te</p>st":               "test",
		"t<p>e</p>st":            "test",
		"te<ul>st":               "test",
		"te</ul>st":              "test",
		"t<ul>e</ul>st":          "test",
		"te<li>st":               "te — st",
		"te</li>st":              "test",
		"t<li>e</li>st":          "t — est",
		"te&nbsp;st":             "te st",
		"t&nbsp;e&nbsp;s&nbsp;t": "t e s t",
		"te<sup>st":              "te**st",
		"te</sup>st":             "test",
		"t<sup>e</sup>st":        "t**est",
		"te<sub>st":              "te(st",
		"te</sub>st":             "te)st",
		"t<sub>e</sub>st":        "t(e)st",
		"te<em>st":               "test",
		"te</em>st":              "test",
		"t<em>e</em>st":          "test",
		"te\n\nst":               "test",
		"t<p>e</p>s<ul>t</ul>t<li>e</li>s&nbsp;t<sup>t</sup>e<sub>s</sub>tt<em>e</em>s\n\nt<br>t</br>e</strong>": "testt — es t**te(s)ttestte</strong>",
		"<p></p><ul></ul><li></li>&nbsp;<sup></sup><sub></sub><em></em>\n\n<br></br></strong>":                   " —  **()</strong>",
		"te<br>st":      "test",
		"te</br>st":     "test",
		"t<br>e</br>st": "test",
	}
	for testCase, result := range cases {
		calculatedResult := RemoveUnsupportedTags(testCase)
		assert.Equal(t, calculatedResult, result, "Unexpected RemoveUnsupportedTags transformation")
	}
}

func TestGetTaskText(t *testing.T) {
	task := BotLeetCodeTask{}
	task.Title = "Test TaSk title"
	task.Content = "VeRy tEsT TaSk CoNTEnt\r"
	waited := "<strong>Test TaSk title</strong>\n\nVeRy tEsT TaSk CoNTEnt\r"
	assert.Equal(t, waited, task.GetTaskText(), "Unexpected GetTaskText response")
}

type callbackTestCase struct {
	dateID         uint64
	hintID         int
	dataType       CallbackType
	awaitingResult string
}

func TestGetMarshalledCallbackData(t *testing.T) {
	testCases := []callbackTestCase{
		{dateID: 10, hintID: 22, dataType: HintReuqest, awaitingResult: "{\"dateID\":\"10\",\"callback_type\":0,\"hint\":22}"},
		{dateID: 10, hintID: 0, dataType: HintReuqest, awaitingResult: "{\"dateID\":\"10\",\"callback_type\":0,\"hint\":0}"},
		{dateID: 10, hintID: 22, dataType: DifficultyRequest, awaitingResult: "{\"dateID\":\"10\",\"callback_type\":1,\"hint\":0}"},
	}
	for _, testCase := range testCases {
		result, err := GetMarshalledCallbackData(testCase.dateID, testCase.hintID, testCase.dataType)
		assert.Nil(t, err, "Unexpected error from GetMarshalledCallbackData")
		assert.Equal(t, result, testCase.awaitingResult, "Unexpected GetMarshalledCallbackData response")
	}
}

type inlineKeyboardTestCase struct {
	DateID         uint64
	TitleSlug      string
	Hints          []string
	awaitingResult string
}

func TestGetInlineKeyboard(t *testing.T) {
	testCases := []inlineKeyboardTestCase{
		{20230101, "5567", []string{}, "{\"inline_keyboard\":[[{\"text\":\"See task on LeetCode website\",\"url\":\"https://leetcode.com/problems/5567\"}],[],[{\"text\":\"Hint: Get the difficulty of the task\",\"callback_data\":\"{\\\"dateID\\\":\\\"20230101\\\",\\\"callback_type\\\":1,\\\"hint\\\":0}\"}]]}"},
		{20230101, "10", []string{"First hint"}, "{\"inline_keyboard\":[[{\"text\":\"See task on LeetCode website\",\"url\":\"https://leetcode.com/problems/10\"}],[{\"text\":\"Hint 1\",\"callback_data\":\"{\\\"dateID\\\":\\\"20230101\\\",\\\"callback_type\\\":0,\\\"hint\\\":0}\"}],[{\"text\":\"Hint: Get the difficulty of the task\",\"callback_data\":\"{\\\"dateID\\\":\\\"20230101\\\",\\\"callback_type\\\":1,\\\"hint\\\":0}\"}]]}"},
		{20030101, "50", []string{"First hint", "Second hint", "Third hint", "Fourth hint", "Fifth hint"}, "{\"inline_keyboard\":[[{\"text\":\"See task on LeetCode website\",\"url\":\"https://leetcode.com/problems/50\"}],[{\"text\":\"Hint 1\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":0}\"},{\"text\":\"Hint 2\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":1}\"},{\"text\":\"Hint 3\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":2}\"},{\"text\":\"Hint 4\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":3}\"},{\"text\":\"Hint 5\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":4}\"}],[{\"text\":\"Hint: Get the difficulty of the task\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":1,\\\"hint\\\":0}\"}]]}"},
		{20030101, "65677", []string{"First hint", "Second hint", "Third hint", "Fourth hint", "Fifth hint", "Sixth hint"}, "{\"inline_keyboard\":[[{\"text\":\"See task on LeetCode website\",\"url\":\"https://leetcode.com/problems/65677\"}],[{\"text\":\"Hint 1\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":0}\"},{\"text\":\"Hint 2\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":1}\"},{\"text\":\"Hint 3\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":2}\"},{\"text\":\"Hint 4\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":3}\"},{\"text\":\"Hint 5\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":4}\"}],[{\"text\":\"Hint 6\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":5}\"}],[{\"text\":\"Hint: Get the difficulty of the task\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":1,\\\"hint\\\":0}\"}]]}"},
		{20030101, "563456", []string{"First hint", "Second hint", "Third hint", "Fourth hint", "Fifth hint", "Sixth hint", "First hint", "Second hint", "Third hint", "Fourth hint", "Fifth hint", "Sixth hint"},
			"{\"inline_keyboard\":[[{\"text\":\"See task on LeetCode website\",\"url\":\"https://leetcode.com/problems/563456\"}],[{\"text\":\"Hint 1\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":0}\"},{\"text\":\"Hint 2\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":1}\"},{\"text\":\"Hint 3\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":2}\"},{\"text\":\"Hint 4\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":3}\"},{\"text\":\"Hint 5\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":4}\"}],[{\"text\":\"Hint 6\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":5}\"},{\"text\":\"Hint 7\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":6}\"},{\"text\":\"Hint 8\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":7}\"},{\"text\":\"Hint 9\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":8}\"},{\"text\":\"Hint 10\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":9}\"}],[{\"text\":\"Hint 11\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":10}\"},{\"text\":\"Hint 12\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":11}\"}],[{\"text\":\"Hint: Get the difficulty of the task\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":1,\\\"hint\\\":0}\"}]]}"},
	}
	for _, testCase := range testCases {
		task := BotLeetCodeTask{DateID: testCase.DateID}
		task.TitleSlug = testCase.TitleSlug
		task.Hints = testCase.Hints
		result := task.GetInlineKeyboard()
		assert.Equal(t, result, testCase.awaitingResult, "Unexpected GetInlineKeyboard response")
	}
}

func TestFixTagsAndImages(t *testing.T) {
	task := BotLeetCodeTask{}
	task.Title = "<br>Test<em>Title"
	task.Content = "<ul> Test <img src=\"http://secret_image.com/image.png\"/>"
	task.Hints = []string{"First </br>hint", "Second <ul>hint", "Third <li>hint"}
	task.FixTagsAndImages()
	assert.Equal(t, task.Title, "TestTitle", "Unexpected FixTagsAndImages transformation")
	assert.Equal(t, task.Content, " Test \n<a href=\"http://secret_image.com/image.png\">Picture 0</a>", "Unexpected FixTagsAndImages transformation")
	assert.Equal(t, task.Hints, []string{"First hint", "Second hint", "Third  — hint"}, "Unexpected FixTagsAndImages transformation")
}

func TestGetDifficultyNum(t *testing.T) {
	difToNum := map[string]int{
		"Easy":                0,
		"Medium":              1,
		"Hard":                2,
		"Not known":           3,
		"Another strange key": 3,
	}
	task := BotLeetCodeTask{}
	for key, val := range difToNum {
		task.Difficulty = key
		assert.Equal(t, task.GetDifficultyNum(), uint8(val), "Wrong GetDifficultyNum response")
	}
}

func TestSetDifficultyFromNum(t *testing.T) {
	numToDif := map[uint8]string{
		0:   "Easy",
		1:   "Medium",
		2:   "Hard",
		3:   "Not known",
		127: "Not known",
	}
	task := BotLeetCodeTask{}
	for key, val := range numToDif {
		task.SetDifficultyFromNum(key)
		assert.Equal(t, task.Difficulty, val, "Unexpected SetDifficultyFromNum response")
	}
}

func TestDateIDFunctions(t *testing.T) {
	loc, _ := time.LoadLocation("US/Pacific")
	now := time.Now().In(loc)
	assert.Equal(t, GetDateID(now), GetDateIDForNow(), "GetDateIDForNow now equal to what it supposed to be")
}

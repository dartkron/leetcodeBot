package common

import (
	"testing"
	"time"
)

func TestReplaceImgTagWithA(t *testing.T) {
	tests := map[string]string{
		"test<img src=\"http://mytest.com/test.png\"/>test<img src=\"http://anothermytest.org/pic.jpg\"/>test": "test\n<a href=\"http://mytest.com/test.png\">Picture 0</a>test\n<a href=\"http://anothermytest.org/pic.jpg\">Picture 1</a>test",
	}
	for source, result := range tests {

		calculatedResult := ReplaceImgTagWithA(source)
		if calculatedResult != result {
			t.Fatalf("Wrong transform, got %s, but %s awaited", calculatedResult, result)
		}

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
		if calculatedResult != result {
			t.Errorf("%d awaited, but got %d", result, calculatedResult)
		}
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
		if calculatedResult != result {
			t.Errorf("%s case proceeded into %s but %s awaited", testCase, calculatedResult, result)
		}
	}
}

func TestGetTaskText(t *testing.T) {
	task := BotLeetCodeTask{}
	task.Title = "Test TaSk title"
	task.Content = "VeRy tEsT TaSk CoNTEnt\r"
	waited := "<strong>Test TaSk title</strong>\n\nVeRy tEsT TaSk CoNTEnt\r"
	if waited != task.GetTaskText() {
		t.Errorf("%q got from GetTaskText but %q awaited", task.GetTaskText(), waited)
	}
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
		result, _ := getMarshalledCallbackData(testCase.dateID, testCase.hintID, testCase.dataType)
		if result != testCase.awaitingResult {
			t.Errorf("Awaited %q but got %q instead", testCase.awaitingResult, result)
		}
	}
}

type inlineKeyboardTestCase struct {
	DateID         uint64
	ItemID         uint64
	Hints          []string
	awaitingResult string
}

func TestGetInlineKeyboard(t *testing.T) {
	testCases := []inlineKeyboardTestCase{
		{20230101, 5567, []string{}, "{\"inline_keyboard\":[[{\"text\":\"See task on LeetCode website\",\"url\":\"https://leetcode.com/explore/item/5567\"}],[],[{\"text\":\"Hint: Get the difficulty of the task\",\"callback_data\":\"{\\\"dateID\\\":\\\"20230101\\\",\\\"callback_type\\\":1,\\\"hint\\\":0}\"}]]}"},
		{20230101, 10, []string{"First hint"}, "{\"inline_keyboard\":[[{\"text\":\"See task on LeetCode website\",\"url\":\"https://leetcode.com/explore/item/10\"}],[{\"text\":\"Hint 1\",\"callback_data\":\"{\\\"dateID\\\":\\\"20230101\\\",\\\"callback_type\\\":0,\\\"hint\\\":0}\"}],[{\"text\":\"Hint: Get the difficulty of the task\",\"callback_data\":\"{\\\"dateID\\\":\\\"20230101\\\",\\\"callback_type\\\":1,\\\"hint\\\":0}\"}]]}"},
		{20030101, 50, []string{"First hint", "Second hint", "Third hint", "Fourth hint", "Fifth hint"}, "{\"inline_keyboard\":[[{\"text\":\"See task on LeetCode website\",\"url\":\"https://leetcode.com/explore/item/50\"}],[{\"text\":\"Hint 1\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":0}\"},{\"text\":\"Hint 2\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":1}\"},{\"text\":\"Hint 3\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":2}\"},{\"text\":\"Hint 4\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":3}\"},{\"text\":\"Hint 5\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":4}\"}],[{\"text\":\"Hint: Get the difficulty of the task\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":1,\\\"hint\\\":0}\"}]]}"},
		{20030101, 65677, []string{"First hint", "Second hint", "Third hint", "Fourth hint", "Fifth hint", "Sixth hint"}, "{\"inline_keyboard\":[[{\"text\":\"See task on LeetCode website\",\"url\":\"https://leetcode.com/explore/item/65677\"}],[{\"text\":\"Hint 1\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":0}\"},{\"text\":\"Hint 2\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":1}\"},{\"text\":\"Hint 3\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":2}\"},{\"text\":\"Hint 4\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":3}\"},{\"text\":\"Hint 5\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":4}\"}],[{\"text\":\"Hint 6\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":5}\"}],[{\"text\":\"Hint: Get the difficulty of the task\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":1,\\\"hint\\\":0}\"}]]}"},
		{20030101, 563456, []string{"First hint", "Second hint", "Third hint", "Fourth hint", "Fifth hint", "Sixth hint", "First hint", "Second hint", "Third hint", "Fourth hint", "Fifth hint", "Sixth hint"},
			"{\"inline_keyboard\":[[{\"text\":\"See task on LeetCode website\",\"url\":\"https://leetcode.com/explore/item/563456\"}],[{\"text\":\"Hint 1\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":0}\"},{\"text\":\"Hint 2\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":1}\"},{\"text\":\"Hint 3\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":2}\"},{\"text\":\"Hint 4\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":3}\"},{\"text\":\"Hint 5\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":4}\"}],[{\"text\":\"Hint 6\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":5}\"},{\"text\":\"Hint 7\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":6}\"},{\"text\":\"Hint 8\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":7}\"},{\"text\":\"Hint 9\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":8}\"},{\"text\":\"Hint 10\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":9}\"}],[{\"text\":\"Hint 11\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":10}\"},{\"text\":\"Hint 12\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":0,\\\"hint\\\":11}\"}],[{\"text\":\"Hint: Get the difficulty of the task\",\"callback_data\":\"{\\\"dateID\\\":\\\"20030101\\\",\\\"callback_type\\\":1,\\\"hint\\\":0}\"}]]}"},
	}

	for _, testCase := range testCases {
		task := BotLeetCodeTask{DateID: testCase.DateID}
		task.ItemID = testCase.ItemID
		task.Hints = testCase.Hints
		result := task.GetInlineKeyboard()
		if result != testCase.awaitingResult {
			t.Errorf("Awaited %q but got %q instead", testCase.awaitingResult, result)
		}
	}

}

func TestFixTagsAndImages(t *testing.T) {
	task := BotLeetCodeTask{}
	task.Title = "<br>Test<em>Title"
	task.Content = "<ul> Test <img src=\"http://secret_image.com/image.png\"/>"
	task.Hints = []string{"First </br>hint", "Second <ul>hint", "Third <li>hint"}
	task.FixTagsAndImages()
	awaitedTitle := "TestTitle"
	if task.Title != awaitedTitle {
		t.Errorf("Wrong fix got title:\n%q\nawaited:\n%q", task.Title, awaitedTitle)
	}
	awaitedContent := " Test \n<a href=\"http://secret_image.com/image.png\">Picture 0</a>"
	if task.Content != awaitedContent {
		t.Errorf("Wrong fix got content:\n%q\nawaited:\n%q", task.Content, awaitedContent)
	}
	awaitedHints := []string{"First hint", "Second hint", "Third  — hint"}
	for i := range task.Hints {
		if task.Hints[i] != awaitedHints[i] {
			t.Errorf("%d %q", i, task.Hints)
		}
	}
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
		if task.GetDifficultyNum() != uint8(val) {
			t.Errorf("%q difficuly awaited to be translated into %d number, but got %d instead", key, val, task.GetDifficultyNum())
		}
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
		if task.Difficulty != val {
			t.Errorf("%d num awaited to be translated into %s difficuly, but got %s instead", key, val, task.Difficulty)
		}
	}
}

package leetcodeclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var ErrBypassTest error = errors.New("test bypass error")

type mockerResponse struct {
	bytes []byte
	err   error
}

type MockRequester struct {
	responses map[string]mockerResponse
}

func (r *MockRequester) requestGraphQl(request graphQlRequest) ([]byte, error) {
	bytes, _ := json.Marshal(request)
	req := string(bytes)
	if _, ok := r.responses[req]; !ok {
		fmt.Printf("Request %q not found in requests map!\n", req)
		return []byte{}, nil
	}
	return r.responses[req].bytes, r.responses[req].err
}

type slugTest struct {
	date time.Time
	slug string
}

func TestGetSlug(t *testing.T) {
	loc, _ := time.LoadLocation("US/Pacific")
	testCases := []slugTest{
		{time.Date(1986, time.April, 26, 01, 23, 47, 0, loc), "april-leetcoding-challenge-1986"},
		{time.Date(1988, time.January, 31, 23, 59, 59, 0, loc), "january-leetcoding-challenge-1988"},
		{time.Date(1988, time.January, 31, 0, 0, 0, 0, loc), "january-leetcoding-challenge-1988"},
	}

	client := NewLeetCodeGraphQlClient()
	client.transport = &MockRequester{}
	for _, testCase := range testCases {
		slug := client.getSlug(testCase.date)
		if slug != testCase.slug {
			t.Errorf("Got slug:\n%q\nbut:\n%q\nawaited", slug, testCase.slug)
		}
	}
}

type dateAndSliceStringsWithError struct {
	date    time.Time
	strings []string
	err     error
}

func getSettedLeetcodeClient() *LeetCodeGraphQlClient {
	slugMap := "{\"operationName\":\"GetChaptersWithItems\",\"variables\":{\"cardSlug\":\"%s\"},\"query\":\"query GetChaptersWithItems($cardSlug: String!) { chapters(cardSlug: $cardSlug) { items {id title type }}}\"}"
	questionSlugMapReq := "{\"operationName\":\"GetItem\",\"variables\":{\"itemId\":\"%s\"},\"query\":\"query GetItem($itemId: String!) {item(id: $itemId) { question { titleSlug }}}\"}"
	questionSlugMapResp := "{\"data\":{\"item\":{\"question\":{\"titleSlug\":\"%s\"}}}}"
	taskRequestMap := "{\"operationName\":\"GetQuestion\",\"variables\":{\"titleSlug\":\"%s\"},\"query\":\"query GetQuestion($titleSlug: String!) {question(titleSlug: $titleSlug) { questionId questionTitle difficulty content hints }}\"}"
	client := newLeetCodeGraphQlClient(&MockRequester{
		responses: map[string]mockerResponse{
			fmt.Sprintf(slugMap, "april-leetcoding-challenge-1986"): {[]byte{}, ErrBypassTest},
			fmt.Sprintf(slugMap, "august-leetcoding-challenge-1995"): {
				[]byte("{\"data\":{\"chapters\":[{\"items\":[{\"id\":\"6655\",\"title\":\"Test title\",\"type\":1},{\"id\":\"2346\",\"title\":\"Test title2\",\"type\":1},{\"id\":\"5432\",\"title\":\"Test title3\",\"type\":1},{\"id\":\"7654\",\"title\":\"Test title4\",\"type\":1},{\"id\":\"7543\",\"title\":\"Test title5\",\"type\":1},{\"id\":\"7534\",\"title\":\"Test title6\",\"type\":1},{\"id\":\"12345\",\"title\":\"Test title6\",\"type\":1},{\"id\":\"9876\",\"title\":\"Test title7\",\"type\":1}]},{\"items\":[{\"id\":\"6574\",\"title\":\"Test title8\",\"type\":1},{\"id\":\"2345\",\"title\":\"Test title9\",\"type\":1},{\"id\":\"3567\",\"title\":\"Test title9\",\"type\":1},{\"id\":\"4565\",\"title\":\"Test title10\",\"type\":1},{\"id\":\"1245\",\"title\":\"Test title11\",\"type\":1},{\"id\":\"7656\",\"title\":\"Test title12\",\"type\":1},{\"id\":\"6754\",\"title\":\"Test title13\",\"type\":1},{\"id\":\"7654\",\"title\":\"Test title14\",\"type\":1}]},{\"items\":[{\"id\":\"4667\",\"title\":\"Test title15\",\"type\":1},{\"id\":\"6453\",\"title\":\"Test title16\",\"type\":1},{\"id\":\"6454\",\"title\":\"Test title17\",\"type\":1},{\"id\":\"8798\",\"title\":\"Test title18\",\"type\":1},{\"id\":\"6716\",\"title\":\"Test title19\",\"type\":1},{\"id\":\"8964\",\"title\":\"Test title20\",\"type\":1},{\"id\":\"8673\",\"title\":\"Test title21\",\"type\":1},{\"id\":\"5672\",\"title\":\"Test title22\",\"type\":1}]},{\"items\":[{\"id\":\"7654\",\"title\":\"Test title23\",\"type\":1},{\"id\":\"2345\",\"title\":\"Test title24\",\"type\":1},{\"id\":\"3565\",\"title\":\"Test title25\",\"type\":1},{\"id\":\"4567\",\"title\":\"Test title26\",\"type\":1},{\"id\":\"3567\",\"title\":\"Test title27\",\"type\":1},{\"id\":\"2345\",\"title\":\"Test title28\",\"type\":1}]},{\"items\":[]}]}}"),
				nil,
			},

			fmt.Sprintf(questionSlugMapReq, "2345"): {[]byte(fmt.Sprintf(questionSlugMapResp, "test-title28")), nil},
			fmt.Sprintf(questionSlugMapReq, "2346"): {[]byte{}, ErrBypassTest},
			fmt.Sprintf(questionSlugMapReq, "2347"): {[]byte(fmt.Sprintf(questionSlugMapResp, "test-title29")), nil},

			fmt.Sprintf(taskRequestMap, "test-title28"): {[]byte("{\"data\":{\"question\":{\"questionId\":\"6577\",\"questionTitle\":\"Test title28\",\"difficulty\":\"Easy\",\"content\":\"Very test content of test title28\",\"hints\":[\"First hint\", \"Last hint\"]}}}"), nil},
			fmt.Sprintf(taskRequestMap, "test-title29"): {[]byte{}, ErrBypassTest},
		},
	})
	return client
}

func TestGetDailyTaskItemsIdsForMonth(t *testing.T) {
	client := getSettedLeetcodeClient()
	loc, _ := time.LoadLocation("US/Pacific")

	testCases := []dateAndSliceStringsWithError{
		{time.Date(1986, time.April, 26, 01, 23, 47, 0, loc), []string{}, ErrBypassTest},
		{time.Date(1995, time.August, 26, 01, 23, 47, 0, loc), []string{"2346", "5432", "7654", "7543", "7534", "12345", "9876", "2345", "3567", "4565", "1245", "7656", "6754", "7654", "6453", "6454", "8798", "6716", "8964", "8673", "5672", "2345", "3565", "4567", "3567", "2345"}, nil},
	}
	for _, testCase := range testCases {
		tasks, err := client.getDailyTaskItemsIdsForMonth(testCase.date)
		if !reflect.DeepEqual(tasks, testCase.strings) {
			t.Errorf("Returned wrong tasks list:\n%q\nbut awaited:%q\n", tasks, testCase.strings)
		}
		if err != testCase.err {
			t.Errorf("Wrong error returned: %q but %q awaited", err, testCase.err)
		}
	}
}

type dateAndSliceStringWithError struct {
	date time.Time
	data string
	err  error
}

func TestGetDailyTaskItemID(t *testing.T) {
	client := getSettedLeetcodeClient()
	loc, _ := time.LoadLocation("US/Pacific")
	testCases := []dateAndSliceStringWithError{
		{time.Date(1986, time.April, 26, 01, 23, 47, 0, loc), "", ErrBypassTest},
		{time.Date(1995, time.August, 26, 01, 23, 47, 0, loc), "2345", nil},
		{time.Date(1995, time.August, 27, 01, 23, 47, 0, loc), "", fmt.Errorf("can't get 27 task for month August. Only 26 tasks isset")},
		{time.Date(1995, time.August, 4, 01, 23, 47, 0, loc), "7543", nil},
	}
	for _, testCase := range testCases {
		itemID, err := client.GetDailyTaskItemID(testCase.date)
		if itemID != testCase.data {
			t.Errorf("Returned wrong itemID: %q but %q awaited", itemID, testCase.data)
		}
		if err != testCase.err && err.Error() != testCase.err.Error() {
			t.Errorf("Wrong error returned: %q but %q awaited", err, testCase.err)
		}
	}
}

type taskAndErr struct {
	task LeetCodeTask
	err  error
}

func TestGetQuestionDetailsByItemID(t *testing.T) {
	client := getSettedLeetcodeClient()
	testCases := map[string]taskAndErr{
		"2346": {LeetCodeTask{}, ErrBypassTest},
		"2347": {LeetCodeTask{}, ErrBypassTest},
		"2345": {
			LeetCodeTask{
				QuestionID: 6577,
				ItemID:     2345,
				Title:      "Test title28",
				Content:    "Very test content of test title28",
				Hints:      []string{"First hint", "Last hint"},
				Difficulty: "Easy",
			},
			nil},
	}
	for itemID, testCase := range testCases {
		task, err := client.GetQuestionDetailsByItemID(itemID)
		// if err != testCase.err && (err == nil || err.Error() != testCase.err.Error()) {
		if err != testCase.err {
			t.Errorf("Wrong error returned: %q but %q awaited", err, testCase.err)
		}
		if !reflect.DeepEqual(task, testCase.task) {
			t.Errorf("Returned task:\n%q\nawaited task:\n%q", task, testCase.task)
		}
	}
}

type dateTaskErr struct {
	date time.Time
	task LeetCodeTask
	err  error
}

func TestGetDailyTask(t *testing.T) {
	loc, _ := time.LoadLocation("US/Pacific")
	client := getSettedLeetcodeClient()
	testCases := []dateTaskErr{
		{
			time.Date(1995, time.August, 26, 01, 23, 47, 0, loc),
			LeetCodeTask{
				QuestionID: 6577,
				ItemID:     2345,
				Title:      "Test title28",
				Content:    "Very test content of test title28",
				Hints:      []string{"First hint", "Last hint"},
				Difficulty: "Easy",
			},
			nil,
		},
		{
			time.Date(1995, time.August, 1, 01, 23, 47, 0, loc),
			LeetCodeTask{},
			ErrBypassTest,
		},
		{
			time.Date(1995, time.August, 29, 01, 23, 47, 0, loc),
			LeetCodeTask{},
			errors.New("can't get 29 task for month August. Only 26 tasks isset"),
		},
	}

	for _, testCase := range testCases {
		task, err := client.GetDailyTask(testCase.date)
		if !reflect.DeepEqual(task, testCase.task) {
			t.Errorf("Returned task:\n%q\nawaited task:\n%q", task, testCase.task)
		}
		if err != testCase.err {
			if err == nil || err.Error() != testCase.err.Error() {
				t.Errorf("Wrong error returned: %q but %q awaited", err, testCase.err)
			}
		}

	}

}

func TestNewLeetCodeGraphQlClient(t *testing.T) {
	client := NewLeetCodeGraphQlClient()
	assert.NotEmpty(t, client.getChaptersReq.OperationName, "getChaptersReq.OperationName must be configured in constructor")
	assert.NotEmpty(t, client.getSlugReq.OperationName, "getSlugReq.OperationName must be configured in constructor")
	assert.NotEmpty(t, client.getQuestionReq.OperationName, "getQuestionReq.OperationName must be configured in constructor")
	assert.NotNil(t, client.transport, "transport must be configured in constructor")
}

func TestNewLeetCodeGraphQlClientLocal(t *testing.T) {
	client := newLeetCodeGraphQlClient(nil)
	assert.Nil(t, client.transport, "transport must be set in private constructor")
}

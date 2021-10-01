package leetcodeclient

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dartkron/leetcodeBot/v2/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRequester struct {
	mock.Mock
}

func (r *MockRequester) requestGraphQl(ctx context.Context, request graphQlRequest) ([]byte, error) {
	args := r.Called(request)
	return args.Get(0).([]byte), args.Error(1)
}

type sliceDateChallengesError struct {
	date       time.Time
	challenges []challengeDesc
	err        error
}

type sliceDateStringError struct {
	date time.Time
	str  string
	err  error
}

func TestLeetcodeDateUnmarshal(t *testing.T) {
	testDate := LeetcodeDate{}
	err := testDate.UnmarshalJSON([]byte(""))
	assert.Equal(t, "parsing time \"\" as \"2006-01-02\": cannot parse \"\" as \"2006\"", err.Error(), "Unexpected parse error")
	err = testDate.UnmarshalJSON([]byte("1986-04-26"))
	assert.Nil(t, err, "Unexprecred error")
	loc, _ := time.LoadLocation("US/Pacific")
	assert.Equal(t, time.Date(1986, time.April, 26, 0, 0, 0, 0, loc), time.Time(testDate), "Unexpected date after parse")

	err = testDate.UnmarshalJSON([]byte("2029-12-31"))
	assert.Nil(t, err, "Unexprecred error")
	assert.Equal(t, time.Date(2029, time.December, 31, 0, 0, 0, 0, loc), time.Time(testDate), "Unexpected date after parse")
	err = testDate.UnmarshalJSON([]byte("2030-01-01"))
	assert.Nil(t, err, "Unexprecred error")
	assert.Equal(t, time.Date(2030, time.January, 01, 0, 0, 0, 0, loc), time.Time(testDate), "Unexpected date after parse")
}

func TestGetMonthlyQuestionsSlugs(t *testing.T) {
	client := NewLeetCodeGraphQlClient()
	mockRequester := &MockRequester{}
	client.transport = mockRequester
	loc, _ := time.LoadLocation("US/Pacific")
	chaptersReq := client.getDailyQuestionsSlugsReq
	chaptersReq.Variables = map[string]string{
		"year":  "1986",
		"month": "4",
	}
	mockRequester.On(
		"requestGraphQl",
		chaptersReq,
	).Return(
		[]byte(""),
		tests.ErrBypassTest,
	).Times(1)
	chaptersReq.Variables = map[string]string{
		"year":  "1995",
		"month": "8",
	}
	mockRequester.On(
		"requestGraphQl",
		chaptersReq,
	).Return(
		[]byte("{\"data\":{\"dailyCodingChallengeV2\":{\"challenges\":[{\"date\":\"1995-08-01\",\"question\":{\"titleSlug\":\"test-title\"}},{\"date\":\"1995-08-02\",\"question\":{\"titleSlug\":\"test-title2\"}},{\"date\":\"1995-08-03\",\"question\":{\"titleSlug\":\"test-title3\"}}]}}}"),
		nil,
	).Times(1)
	chaptersReq.Variables = map[string]string{
		"year":  "1986",
		"month": "5",
	}
	mockRequester.On(
		"requestGraphQl",
		chaptersReq,
	).Return(
		[]byte("{\"}"),
		nil,
	).Times(1)
	testCases := []sliceDateChallengesError{
		{time.Date(1986, time.April, 26, 01, 23, 47, 0, loc), []challengeDesc{}, tests.ErrBypassTest},
		{time.Date(1995, time.August, 26, 01, 23, 47, 0, loc), []challengeDesc{{Date: LeetcodeDate(time.Date(1995, time.August, 1, 0, 0, 0, 0, loc)), Question: struct {
			TitleSlug string "json:\"titleSlug\""
		}{TitleSlug: "test-title"}}, {Date: LeetcodeDate(time.Date(1995, time.August, 2, 0, 0, 0, 0, loc)), Question: struct {
			TitleSlug string "json:\"titleSlug\""
		}{TitleSlug: "test-title2"}}, {Date: LeetcodeDate(time.Date(1995, time.August, 3, 0, 0, 0, 0, loc)), Question: struct {
			TitleSlug string "json:\"titleSlug\""
		}{TitleSlug: "test-title3"}}}, nil},
		{time.Date(1986, time.May, 26, 01, 23, 47, 0, loc), []challengeDesc(nil), tests.ErrWrongJSON},
	}
	for _, testCase := range testCases {
		res, err := client.getMonthlyQuestionsSlugs(context.Background(), testCase.date)
		if testCase.err == nil {
			assert.Nil(t, err, "Unexpected error")
		} else {
			assert.Equal(t, testCase.err.Error(), err.Error(), "Unexpected error")
		}
		assert.Equal(t, testCase.challenges, res, "Unexpected response")
	}
	mockRequester.AssertExpectations(t)
}

func TestGetDailyQuestionSlug(t *testing.T) {
	client := NewLeetCodeGraphQlClient()
	mockRequester := &MockRequester{}
	client.transport = mockRequester
	loc, _ := time.LoadLocation("US/Pacific")
	chaptersReq := client.getDailyQuestionsSlugsReq
	chaptersReq.Variables = map[string]string{
		"year":  "1986",
		"month": "4",
	}
	mockRequester.On(
		"requestGraphQl",
		chaptersReq,
	).Return(
		[]byte(""),
		tests.ErrBypassTest,
	).Times(1)
	chaptersReq.Variables = map[string]string{
		"year":  "1995",
		"month": "8",
	}
	mockRequester.On(
		"requestGraphQl",
		chaptersReq,
	).Return(
		[]byte("{\"data\":{\"dailyCodingChallengeV2\":{\"challenges\":[{\"date\":\"1995-08-01\",\"question\":{\"titleSlug\":\"test-title\"}},{\"date\":\"1995-08-02\",\"question\":{\"titleSlug\":\"test-title2\"}},{\"date\":\"1995-08-03\",\"question\":{\"titleSlug\":\"test-title3\"}}]}}}"),
		nil,
	).Times(4)
	testCases := []sliceDateStringError{
		{time.Date(1986, time.April, 26, 01, 23, 47, 0, loc), "", tests.ErrBypassTest},
		{time.Date(1995, time.August, 1, 01, 23, 47, 0, loc), "test-title", nil},
		{time.Date(1995, time.August, 2, 01, 23, 47, 0, loc), "test-title2", nil},
		{time.Date(1995, time.August, 3, 01, 23, 47, 0, loc), "test-title3", nil},
		{time.Date(1995, time.August, 4, 01, 23, 47, 0, loc), "", errors.New("can't get 4 task for month August. Only 3 tasks isset")},
	}
	for _, testCase := range testCases {
		slug, err := client.GetDailyQuestionSlug(context.Background(), testCase.date)
		if testCase.err == nil {
			assert.Nil(t, err, "Unexpected error")
		} else {
			assert.Equal(t, testCase.err.Error(), err.Error(), "Unexpected error")
		}
		assert.Equal(t, testCase.str, slug, "Unexpected response")
	}
	mockRequester.AssertExpectations(t)
}

type sliceStringLeetcodeTaskError struct {
	str  string
	task LeetCodeTask
	err  error
}

func TestGetQuestionDetailsByTitleSlug(t *testing.T) {
	client := NewLeetCodeGraphQlClient()
	mockRequester := &MockRequester{}
	client.transport = mockRequester
	questionReq := client.getQuestionReq
	questionReq.Variables["titleSlug"] = "test-title0"
	mockRequester.On(
		"requestGraphQl",
		questionReq,
	).Return(
		[]byte{},
		tests.ErrBypassTest,
	).Times(1)
	questionReq.Variables["titleSlug"] = "test-title"
	mockRequester.On(
		"requestGraphQl",
		questionReq,
	).Return(
		[]byte("{\"data\":{\"question\":{\"questionId\":\"1254\",\"questionTitle\":\"Test title\",\"difficulty\":\"Easy\",\"content\":\"<p>My very test content <code>with code</code></p>\\n\\n\",\"hints\":[\"First hint\",\"Second hint\"]}}}"),
		nil,
	).Times(1)
	questionReq.Variables["titleSlug"] = "test-title1"
	mockRequester.On(
		"requestGraphQl",
		questionReq,
	).Return(
		[]byte("{\"}"),
		nil,
	).Times(1)

	testCases := []sliceStringLeetcodeTaskError{
		{"test-title0", LeetCodeTask{}, tests.ErrBypassTest},
		{"test-title", LeetCodeTask{
			QuestionID: 1254,
			Title:      "Test title",
			TitleSlug:  "test-title",
			Difficulty: "Easy",
			Content:    "<p>My very test content <code>with code</code></p>\n\n",
			Hints:      []string{"First hint", "Second hint"},
		}, nil},
		{"test-title1", LeetCodeTask{}, tests.ErrWrongJSON},
	}
	for _, testCase := range testCases {
		task, err := client.GetQuestionDetailsByTitleSlug(context.Background(), testCase.str)
		if testCase.err == nil {
			assert.Nil(t, err, "Unexpected error")
		} else {
			assert.Equal(t, testCase.err.Error(), err.Error(), "Unexpected error")
		}
		assert.Equal(t, testCase.task, task, "Unexpected response")
	}
	mockRequester.AssertExpectations(t)
}

type sliceDateLeetcodeTaskError struct {
	date time.Time
	task LeetCodeTask
	err  error
}

func TestGetDailyTask(t *testing.T) {
	client := NewLeetCodeGraphQlClient()
	mockRequester := &MockRequester{}
	client.transport = mockRequester
	loc, _ := time.LoadLocation("US/Pacific")
	chaptersReq := client.getDailyQuestionsSlugsReq
	chaptersReq.Variables = map[string]string{
		"year":  "1986",
		"month": "4",
	}
	mockRequester.On(
		"requestGraphQl",
		chaptersReq,
	).Return(
		[]byte(""),
		tests.ErrBypassTest,
	).Times(1)
	chaptersReq.Variables = map[string]string{
		"year":  "1995",
		"month": "8",
	}
	mockRequester.On(
		"requestGraphQl",
		chaptersReq,
	).Return(
		[]byte("{\"data\":{\"dailyCodingChallengeV2\":{\"challenges\":[{\"date\":\"1995-08-01\",\"question\":{\"titleSlug\":\"test-title\"}},{\"date\":\"1995-08-02\",\"question\":{\"titleSlug\":\"test-title2\"}},{\"date\":\"1995-08-03\",\"question\":{\"titleSlug\":\"test-title3\"}}]}}}"),
		nil,
	).Times(2)
	questionReq := client.getQuestionReq
	questionReq.Variables["titleSlug"] = "test-title2"
	mockRequester.On(
		"requestGraphQl",
		questionReq,
	).Return(
		[]byte("{\"data\":{\"question\":{\"questionId\":\"1254\",\"questionTitle\":\"Test title\",\"difficulty\":\"Easy\",\"content\":\"<p>My very test content <code>with code</code></p>\\n\\n\",\"hints\":[\"First hint\",\"Second hint\"]}}}"),
		nil,
	).Times(1)
	questionReq.Variables["titleSlug"] = "test-title3"
	mockRequester.On(
		"requestGraphQl",
		questionReq,
	).Return(
		[]byte("{\"}"),
		nil,
	).Times(1)
	testCases := []sliceDateLeetcodeTaskError{
		{time.Date(1986, time.April, 26, 01, 23, 47, 0, loc), LeetCodeTask{}, tests.ErrBypassTest},
		{time.Date(1995, time.August, 2, 01, 23, 47, 0, loc), LeetCodeTask{
			QuestionID: 1254,
			Title:      "Test title",
			TitleSlug:  "test-title2",
			Difficulty: "Easy",
			Content:    "<p>My very test content <code>with code</code></p>\n\n",
			Hints:      []string{"First hint", "Second hint"},
		}, nil},
		{time.Date(1995, time.August, 2, 01, 23, 47, 0, loc), LeetCodeTask{}, tests.ErrWrongJSON},
	}
	for _, testCase := range testCases {
		task, err := client.GetDailyTask(context.Background(), testCase.date)
		if testCase.err == nil {
			assert.Nil(t, err, "Unexpected error")
		} else {
			assert.Equal(t, testCase.err.Error(), err.Error(), "Unexpected error")
		}
		assert.Equal(t, testCase.task, task, "Unexpected response")
	}
	mockRequester.AssertExpectations(t)
}

func TestNewLeetCodeGraphQlClient(t *testing.T) {
	client := NewLeetCodeGraphQlClient()
	assert.NotEmpty(t, client.getDailyQuestionsSlugsReq.OperationName, "getChaptersReq.OperationName must be configured in constructor")
	assert.NotEmpty(t, client.getQuestionReq.OperationName, "getQuestionReq.OperationName must be configured in constructor")
	assert.NotNil(t, client.transport, "transport must be configured in constructor")
}

func TestNewLeetCodeGraphQlClientLocal(t *testing.T) {
	client := newLeetCodeGraphQlClient(nil)
	assert.Nil(t, client.transport, "transport must be set in private constructor")
}

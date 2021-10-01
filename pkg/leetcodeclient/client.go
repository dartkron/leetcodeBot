package leetcodeclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LeetCodeTask is a necessary information about task at LeetCode
type LeetCodeTask struct {
	QuestionID uint64   `json:"questionId,string"`
	TitleSlug  string   `json:"titleSlug"`
	Title      string   `json:"questionTitle"`
	Content    string   `json:"content"`
	Hints      []string `json:"hints"`
	Difficulty string   `json:"difficulty"`
}

// LeetcodeClient represents abstract set of methods required from any possible kind of Leetcode client
type LeetcodeClient interface {
	GetDailyQuestionSlug(context.Context, time.Time) (string, error)
	GetQuestionDetailsByTitleSlug(context.Context, string) (LeetCodeTask, error)
	GetDailyTask(context.Context, time.Time) (LeetCodeTask, error)
}

// LeetcodeDate time.Time with specific json unmarshaller to parse json dates
type LeetcodeDate time.Time

// UnmarshalJSON parsing date in format "2006-01-02" in json
func (j *LeetcodeDate) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	loc, _ := time.LoadLocation("US/Pacific")
	t, err := time.ParseInLocation("2006-01-02", s, loc)
	if err != nil {
		return err
	}
	*j = LeetcodeDate(t)
	return nil
}

type challengeDesc struct {
	Date     LeetcodeDate `json:"date"`
	Question struct {
		TitleSlug string `json:"titleSlug"`
	} `json:"question"`
}

type dailyCodingChallengeV2desc struct {
	Data struct {
		DailyCodingChallengeV2 struct {
			Challenges []challengeDesc `json:"challenges"`
		} `json:"dailyCodingChallengeV2"`
	} `json:"data"`
}

type questionDesc struct {
	Data struct {
		Question LeetCodeTask
	}
}

// LeetCodeGraphQlClient realization of GraphQL client.
// Potentially supports different requester types
type LeetCodeGraphQlClient struct {
	getDailyQuestionsSlugsReq graphQlRequest
	getQuestionReq            graphQlRequest
	transport                 graphQlRequester
}

type graphQlRequest struct {
	OperationName string            `json:"operationName"`
	Variables     map[string]string `json:"variables"`
	Query         string            `json:"query"`
}

type graphQlRequester interface {
	requestGraphQl(context.Context, graphQlRequest) ([]byte, error)
}

// GetDailyQuestionSlug provides slug for daily for the particular date
func (c *LeetCodeGraphQlClient) GetDailyQuestionSlug(ctx context.Context, date time.Time) (string, error) {
	monthlyChanngelgesSlugs, err := c.getMonthlyQuestionsSlugs(ctx, date)
	if err != nil {
		return "", err
	}
	if len(monthlyChanngelgesSlugs) < date.Day() {
		return "", fmt.Errorf("can't get %d task for month %s. Only %d tasks isset", date.Day(), date.Month().String(), len(monthlyChanngelgesSlugs))
	}
	return monthlyChanngelgesSlugs[date.Day()-1].Question.TitleSlug, nil
}

func (c *LeetCodeGraphQlClient) getMonthlyQuestionsSlugs(ctx context.Context, date time.Time) ([]challengeDesc, error) {
	monthlyQuestionsReq := c.getDailyQuestionsSlugsReq
	monthlyQuestionsReq.Variables["month"] = fmt.Sprintf("%d", int(date.Month()))
	monthlyQuestionsReq.Variables["year"] = fmt.Sprintf("%d", date.Year())
	responseBytes, err := c.transport.requestGraphQl(ctx, monthlyQuestionsReq)
	if err != nil {
		return []challengeDesc{}, err
	}
	parsed := dailyCodingChallengeV2desc{}
	err = json.Unmarshal(responseBytes, &parsed)
	return parsed.Data.DailyCodingChallengeV2.Challenges, err
}

// GetQuestionDetailsByTitleSlug provides all details of the question: title, text, hints, difficulty by provided titleSlug
func (c *LeetCodeGraphQlClient) GetQuestionDetailsByTitleSlug(ctx context.Context, titleSlug string) (LeetCodeTask, error) {
	questionReq := c.getQuestionReq
	questionReq.Variables["titleSlug"] = titleSlug

	responseBytes, err := c.transport.requestGraphQl(ctx, c.getQuestionReq)
	if err != nil {
		return LeetCodeTask{}, err
	}

	parsed := questionDesc{}
	err = json.Unmarshal(responseBytes, &parsed)
	if err != nil {
		return parsed.Data.Question, err
	}
	parsed.Data.Question.TitleSlug = titleSlug
	return parsed.Data.Question, err
}

// GetDailyTask shortcut of GetDailyTaskItemID and GetQuestionDetailsByTitleSlug
func (c *LeetCodeGraphQlClient) GetDailyTask(ctx context.Context, date time.Time) (LeetCodeTask, error) {
	questionSlug, err := c.GetDailyQuestionSlug(ctx, date)
	if err != nil {
		return LeetCodeTask{}, err
	}
	return c.GetQuestionDetailsByTitleSlug(ctx, questionSlug)
}

// NewLeetCodeGraphQlClient construct LeetCode client with default values
func NewLeetCodeGraphQlClient() *LeetCodeGraphQlClient {
	return newLeetCodeGraphQlClient(newHTTPGraphQlRequester(nil))
}

func newLeetCodeGraphQlClient(requester graphQlRequester) *LeetCodeGraphQlClient {
	client := LeetCodeGraphQlClient{
		getDailyQuestionsSlugsReq: graphQlRequest{
			OperationName: "dailyCodingQuestionRecords",
			Query: `query dailyCodingQuestionRecords($year: Int!, $month: Int!) { dailyCodingChallengeV2(year: $year, month: $month) { challenges {	date question { titleSlug } } } }`,
			Variables: make(map[string]string),
		},
		getQuestionReq: graphQlRequest{
			OperationName: "GetQuestion",
			Variables:     make(map[string]string),
			Query:         "query GetQuestion($titleSlug: String!) {question(titleSlug: $titleSlug) { questionId questionTitle difficulty content hints }}",
		},
		transport: requester,
	}
	return &client
}

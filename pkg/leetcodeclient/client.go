package leetcodeclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// LeetCodeTask is a necessary information about task at LeetCode
type LeetCodeTask struct {
	QuestionID uint64   `json:"questionId,string"`
	ItemID     uint64   `json:"itemId,string"`
	Title      string   `json:"questionTitle"`
	Content    string   `json:"content"`
	Hints      []string `json:"hints"`
	Difficulty string   `json:"difficulty"`
}

// LeetcodeClient represents abstract set of methods required from any possible kind of Leetcode client
type LeetcodeClient interface {
	GetDailyTaskItemID(context.Context, time.Time) (string, error)
	GetQuestionDetailsByItemID(context.Context, string) (LeetCodeTask, error)
	GetDailyTask(context.Context, time.Time) (LeetCodeTask, error)
}

type chaptersDesc struct {
	Data struct {
		Chapters []struct {
			Items []struct {
				ID    string
				Title string
			}
		}
	}
}

type questionDesc struct {
	Data struct {
		Question LeetCodeTask
	}
}

type titleSlugDesc struct {
	Data struct {
		Item struct {
			Question struct {
				QuestionID string
				Title      string
				TitleSlug  string
			}
		}
	}
}

// LeetCodeGraphQlClient realization of GraphQL client.
// Potentially supports different requester types
type LeetCodeGraphQlClient struct {
	getChaptersReq graphQlRequest
	getSlugReq     graphQlRequest
	getQuestionReq graphQlRequest
	transport      graphQlRequester
}

type graphQlRequest struct {
	OperationName string            `json:"operationName"`
	Variables     map[string]string `json:"variables"`
	Query         string            `json:"query"`
}

type graphQlRequester interface {
	requestGraphQl(context.Context, graphQlRequest) ([]byte, error)
}

func (c *LeetCodeGraphQlClient) getDailyTaskItemsIdsForMonth(ctx context.Context, date time.Time) ([]string, error) {
	chaptersReq := c.getChaptersReq

	cardSlug := c.getSlug(date)
	chaptersReq.Variables = map[string]string{"cardSlug": cardSlug}
	responseBytes, err := c.transport.requestGraphQl(ctx, chaptersReq)
	if err != nil {
		return []string{}, err
	}
	parsed := chaptersDesc{}
	err = json.Unmarshal(responseBytes, &parsed)
	resp := []string{}
	for _, week := range parsed.Data.Chapters {
		for i, item := range week.Items {
			if i == 0 {
				continue
			}
			resp = append(resp, item.ID)
		}
	}
	return resp, err
}

func (c *LeetCodeGraphQlClient) getDailyTaskItemIDForDate(ctx context.Context, date time.Time) (string, error) {
	forMonth, err := c.getDailyTaskItemsIdsForMonth(ctx, date)
	if err != nil {
		return "", err
	}
	if date.Day() > len(forMonth) {
		return "", fmt.Errorf("can't get %d task for month %s. Only %d tasks isset", date.Day(), date.Month().String(), len(forMonth))
	}
	return forMonth[date.Day()-1], nil
}

// GetDailyTaskItemID retrieve itemId for task of the day on provided date
func (c *LeetCodeGraphQlClient) GetDailyTaskItemID(ctx context.Context, date time.Time) (string, error) {
	return c.getDailyTaskItemIDForDate(ctx, date)
}

func (c *LeetCodeGraphQlClient) getSlug(date time.Time) string {
	return fmt.Sprintf("%s-leetcoding-challenge-%d", strings.ToLower(date.Month().String()), date.Year())
}

func (c *LeetCodeGraphQlClient) getQuestionSlug(ctx context.Context, itemID string) (string, error) {
	slugReq := c.getSlugReq
	slugReq.Variables["itemId"] = itemID
	responseBytes, err := c.transport.requestGraphQl(ctx, slugReq)
	if err != nil {
		return "", err
	}
	parsed := titleSlugDesc{}
	err = json.Unmarshal(responseBytes, &parsed)
	return parsed.Data.Item.Question.TitleSlug, err
}

// GetQuestionDetailsByItemID provides all details of the question: title, text, hints, difficulty by provided itemID
func (c *LeetCodeGraphQlClient) GetQuestionDetailsByItemID(ctx context.Context, itemID string) (LeetCodeTask, error) {
	questionReq := c.getQuestionReq
	questionSlug, err := c.getQuestionSlug(ctx, itemID)
	if err != nil {
		return LeetCodeTask{}, err
	}
	questionReq.Variables["titleSlug"] = questionSlug

	responseBytes, err := c.transport.requestGraphQl(ctx, c.getQuestionReq)
	if err != nil {
		return LeetCodeTask{}, err
	}

	parsed := questionDesc{}
	err = json.Unmarshal(responseBytes, &parsed)
	if err != nil {
		return parsed.Data.Question, err
	}
	parsed.Data.Question.ItemID, err = strconv.ParseUint(itemID, 10, 64)
	return parsed.Data.Question, err
}

// GetDailyTask shortcut of GetDailyTaskItemID and GetQuestionDetailsByItemID
func (c *LeetCodeGraphQlClient) GetDailyTask(ctx context.Context, date time.Time) (LeetCodeTask, error) {
	itemID, err := c.GetDailyTaskItemID(ctx, date)
	if err != nil {
		return LeetCodeTask{}, err
	}
	task, err := c.GetQuestionDetailsByItemID(ctx, itemID)
	return task, err
}

// NewLeetCodeGraphQlClient construct LeetCode client with default values
func NewLeetCodeGraphQlClient() *LeetCodeGraphQlClient {
	return newLeetCodeGraphQlClient(newHTTPGraphQlRequester(nil))
}

func newLeetCodeGraphQlClient(requester graphQlRequester) *LeetCodeGraphQlClient {
	client := LeetCodeGraphQlClient{
		getChaptersReq: graphQlRequest{
			OperationName: "GetChaptersWithItems",
			Query:         "query GetChaptersWithItems($cardSlug: String!) { chapters(cardSlug: $cardSlug) { items {id title type }}}",
		},
		getSlugReq: graphQlRequest{
			OperationName: "GetItem",
			Variables:     make(map[string]string),
			Query:         "query GetItem($itemId: String!) {item(id: $itemId) { question { titleSlug }}}",
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

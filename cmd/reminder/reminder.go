package main

import (
	"context"

	"github.com/dartkron/leetcodeBot/v2/internal/bot"
)

// Response type for simplyfied response
type Response struct {
	StatusCode int         `json:"statusCode"`
	Body       interface{} `json:"body"`
}

// Handler for Yandex.Function requests
func Handler(ctx context.Context) (*Response, error) {
	response := &Response{
		StatusCode: 200,
		Body:       "",
	}
	app := bot.NewApplication()
	err := app.SendDailyTaskToSubscribedUsers()
	if err != nil {
		response.StatusCode = 500
		response.Body = err.Error()
		return response, err
	}
	response.Body = "Finished"
	return response, nil
}

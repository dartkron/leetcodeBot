package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dartkron/leetcodeBot/v3/internal/bot"
)

// Handler for Yandex.Function requests
func Handler(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "application/json")
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Println("Error on reading request body:", err)
		return
	}
	app := bot.NewApplication(nil)
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(5)*time.Second)
	defer cancelFunc()
	responseBytes, err := app.ProcessRequestBody(ctx, bodyBytes)
	if err != nil {
		fmt.Println("Sending 500 error in response, because got error from bot.ProcessRequestBody", err)
		resp.WriteHeader(500)
		resp.Write([]byte(err.Error()))
	} else {
		resp.WriteHeader(200)
	}
	resp.Write(responseBytes)
}

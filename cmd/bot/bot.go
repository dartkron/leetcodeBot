package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/dartkron/leetcodeBot/v2/internal/bot"
)

// Handler for Yandex.Function requests
func Handler(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "application/json")
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Println("Error on reading request body:", err)
		return
	}
	app := bot.NewApplication()
	responseBytes, err := app.ProcessRequestBody(bodyBytes)
	if err != nil {
		fmt.Println("Sending 500 error in response, because got error from bot.ProcessRequestBody", err)
		resp.WriteHeader(500)
		resp.Write([]byte(err.Error()))
	} else {
		resp.WriteHeader(200)
	}
	resp.Write(responseBytes)
}

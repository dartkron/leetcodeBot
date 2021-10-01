package leetcodeclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

type httpGraphQlRequester struct {
	GraphQlURL string
	HTTPClient *http.Client
}

func (requester *httpGraphQlRequester) requestGraphQl(ctx context.Context, request graphQlRequest) ([]byte, error) {
	bodyBuffer := new(bytes.Buffer)
	err := json.NewEncoder(bodyBuffer).Encode(request)
	if err != nil {
		return []byte{}, err
	}

	req, _ := http.NewRequestWithContext(ctx, "POST", requester.GraphQlURL, bodyBuffer)
	req.Header.Add("content-type", "application/json")

	response, err := requester.HTTPClient.Do(req)

	if err != nil {
		return []byte{}, err
	}
	return io.ReadAll(response.Body)
}

func newHTTPGraphQlRequester(httpClient *http.Client) *httpGraphQlRequester {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &httpGraphQlRequester{
		GraphQlURL: "https://leetcode.com/graphql",
		HTTPClient: httpClient,
	}
}

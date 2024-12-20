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
	req.Header.Add("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0")
	req.Header.Add("x-csrftoken", "AhmTr4FHZa0sXagk4bEdWYRr50fK5BKa1a0F2ybPwBpoEWuNhKNKmMWlfaCjgCuq") // FIXME: (dartkron): This is a temporary solution and stays valid only for a year, needs to be replaced with getCsrfToken() function
	req.Header.Add("accept", "*/*")

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

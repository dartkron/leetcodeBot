package leetcodeclient

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type httpGraphQlRequester struct {
	GraphQlURL string
	PostFunc   func(string, string, io.Reader) (*http.Response, error)
}

func (requester *httpGraphQlRequester) requestGraphQl(request graphQlRequest) ([]byte, error) {
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(request)
	if err != nil {
		return []byte{}, err
	}
	r, err := requester.PostFunc(requester.GraphQlURL, "application/json", b)
	if err != nil {
		return []byte{}, err
	}
	return io.ReadAll(r.Body)
}

func newHTTPGraphQlRequester() *httpGraphQlRequester {
	return &httpGraphQlRequester{
		GraphQlURL: "https://leetcode.com/graphql",
		PostFunc:   http.Post,
	}
}

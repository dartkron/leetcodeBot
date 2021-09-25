package leetcodeclient

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type httpGraphQlRequester struct {
	GraphQlURL string
}

func (requester *httpGraphQlRequester) requestGraphQl(request graphQlRequest) ([]byte, error) {
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(request)
	if err != nil {
		return []byte{}, err
	}
	r, err := http.Post(requester.GraphQlURL, "application/json", b)
	if err != nil {
		return []byte{}, err
	}
	return ioutil.ReadAll(r.Body)
}

func newHTTPGraphQlRequester() *httpGraphQlRequester {
	return &httpGraphQlRequester{"https://leetcode.com/graphql"}
}

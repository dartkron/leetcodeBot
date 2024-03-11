package leetcodeclient

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/dartkron/leetcodeBot/v3/tests"
	"github.com/dartkron/leetcodeBot/v3/tests/mocks"
	"github.com/stretchr/testify/assert"
)

type testRequest struct {
	req  graphQlRequest
	resp string
	err  error
}

func TestRequestGraphQl(t *testing.T) {
	httpTransportMock := &mocks.MockHTTPTRansport{}
	requester := newHTTPGraphQlRequester(&http.Client{Transport: httpTransportMock})

	expected_headers := http.Header{
		"Content-Type": []string{"application/json"},
		"Accept":       []string{"*/*"},
		"User-Agent":   []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0"},
		"X-Csrftoken":  []string{"Lwf3m4FIFmw8a9u8RQ3u8vXQq7Zqra3aWfBJT8PvXAjhex3V8IQulH1EmXFmKEk8"},
	}

	httpTransportMock.On(
		"RoundTrip",
		"https://leetcode.com/graphql",
		expected_headers,
		"{\"operationName\":\"Test operation\",\"variables\":{\"testVariableName\":\"testVariableVal\"},\"query\":\"Some test query\"}\n",
	).Return(
		&http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("0"))},
		nil,
	).Times(1)
	httpTransportMock.On(
		"RoundTrip",
		"https://leetcode.com/graphql",
		expected_headers,
		"{\"operationName\":\"\",\"variables\":null,\"query\":\"Query for bypass error\"}\n",
	).Return(
		&http.Response{},
		tests.ErrBypassTest,
	).Times(1)
	httpTransportMock.On(
		"RoundTrip",
		"https://leetcode.com/graphql",
		expected_headers,
		"{\"operationName\":\"Test operation no variables\",\"variables\":{},\"query\":\"Some test query1\"}\n",
	).Return(
		&http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("1"))},
		nil,
	).Times(1)
	httpTransportMock.On(
		"RoundTrip",
		"https://leetcode.com/graphql",
		expected_headers,
		"{\"operationName\":\"Test operation two variables\",\"variables\":{\"testVariableName\":\"testVariableVal\",\"testVariableName1\":\"testVariableVal1\"},\"query\":\"Some test query2\"}\n",
	).Return(
		&http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("2"))},
		nil,
	).Times(1)
	httpTransportMock.On(
		"RoundTrip",
		"https://leetcode.com/graphql",
		expected_headers,
		"{\"operationName\":\"Test operation three variables\",\"variables\":{\"testVariableName\":\"testVariableVal\",\"testVariableName1\":\"testVariableVal1\",\"testVariableName2\":\"testVariableVal2\"},\"query\":\"Some test query3\"}\n",
	).Return(
		&http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("3"))},
		nil,
	).Times(1)

	testCases := []testRequest{
		{
			req: graphQlRequest{
				OperationName: "Test operation",
				Variables:     map[string]string{"testVariableName": "testVariableVal"},
				Query:         "Some test query",
			},
			resp: "0",
		},
		{
			req: graphQlRequest{
				Query: "Query for bypass error",
			},
			resp: "",
			err:  &url.Error{Op: "Post", URL: "https://leetcode.com/graphql", Err: tests.ErrBypassTest},
		},
		{
			req: graphQlRequest{
				OperationName: "Test operation no variables",
				Variables:     map[string]string{},
				Query:         "Some test query1",
			},
			resp: "1",
		},
		{
			req: graphQlRequest{
				OperationName: "Test operation two variables",
				Variables:     map[string]string{"testVariableName": "testVariableVal", "testVariableName1": "testVariableVal1"},
				Query:         "Some test query2",
			},
			resp: "2",
		},
		{
			req: graphQlRequest{
				OperationName: "Test operation three variables",
				Variables:     map[string]string{"testVariableName": "testVariableVal", "testVariableName1": "testVariableVal1", "testVariableName2": "testVariableVal2"},
				Query:         "Some test query3",
			},
			resp: "3",
		},
	}

	for _, testCase := range testCases {
		resp, err := requester.requestGraphQl(context.Background(), testCase.req)
		assert.Equal(t, testCase.err, err)
		assert.Equal(t, string(resp), testCase.resp)
	}
	httpTransportMock.AssertExpectations(t)
}

func TestNewHTTPGraphQlRequester(t *testing.T) {
	requester := newHTTPGraphQlRequester(nil)
	assert.NotNil(t, requester.HTTPClient, "PostFunc should be set in conctructor")
	assert.NotEmpty(t, requester.GraphQlURL, "GraphQlURL should be set in conctructor")
}

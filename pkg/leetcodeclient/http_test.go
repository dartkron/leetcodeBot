package leetcodeclient

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type httpResponse struct {
	response *http.Response
	err      error
}

var ErrWorngContentType = errors.New("content type should be \"application/json\"")

func MockHTTPPost(url string, contentType string, body io.Reader) (*http.Response, error) {
	if contentType != "application/json" {
		return &http.Response{}, ErrWorngContentType
	}
	request, err := io.ReadAll(body)
	requestString := string(request)
	if err != nil {
		return &http.Response{}, err
	}
	knownRequests := map[string]httpResponse{
		"{\"operationName\":\"Test operation\",\"variables\":{\"testVariableName\":\"testVariableVal\"},\"query\":\"Some test query\"}\n": {
			response: &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("0"))},
		},
		"{\"operationName\":\"\",\"variables\":null,\"query\":\"Query for bypass error\"}\n": {
			response: &http.Response{},
			err:      ErrBypassTest,
		},
		"{\"operationName\":\"Test operation no variables\",\"variables\":{},\"query\":\"Some test query1\"}\n": {
			response: &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("1"))},
		},
		"{\"operationName\":\"Test operation two variables\",\"variables\":{\"testVariableName\":\"testVariableVal\",\"testVariableName1\":\"testVariableVal1\"},\"query\":\"Some test query2\"}\n": {
			response: &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("2"))},
		},
		"{\"operationName\":\"Test operation three variables\",\"variables\":{\"testVariableName\":\"testVariableVal\",\"testVariableName1\":\"testVariableVal1\",\"testVariableName2\":\"testVariableVal2\"},\"query\":\"Some test query3\"}\n": {
			response: &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("3"))},
		},
	}
	if response, ok := knownRequests[requestString]; ok {
		return response.response, response.err
	}

	return &http.Response{StatusCode: http.StatusInternalServerError}, http.ErrBodyNotAllowed
}

type testRequest struct {
	req  graphQlRequest
	resp string
	err  error
}

func TestRequestGraphQl(t *testing.T) {
	requester := newHTTPGraphQlRequester()
	requester.PostFunc = MockHTTPPost
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
			err:  ErrBypassTest,
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
		resp, err := requester.requestGraphQl(testCase.req)
		if err != testCase.err {
			t.Errorf("Got unwaiter error %q, but %q awaited", err, testCase.err)
		}
		if string(resp) != testCase.resp {
			t.Errorf("Wrong response, awaited:\n%q\nbut recieved:\n%q\n", string(resp), testCase.resp)
		}
	}

}

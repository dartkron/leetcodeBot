package mocks

import (
	"io"
	"net/http"

	"github.com/stretchr/testify/mock"
)

// MockHTTPPost provides universal mock for http.Post function
type MockHTTPPost struct {
	mock.Mock
}

// Func of actual mock for http.Post function
func (m *MockHTTPPost) Func(url string, contentType string, body io.Reader) (*http.Response, error) {
	request, err := io.ReadAll(body)
	if err != nil {
		return &http.Response{}, err
	}
	requestString := string(request)
	args := m.Called(url, contentType, requestString)

	return args.Get(0).(*http.Response), args.Error(1)
}

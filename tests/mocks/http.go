package mocks

import (
	"io"
	"net/http"

	"github.com/stretchr/testify/mock"
)

// MockHTTPTRansport provides universal mock for http.Post function
type MockHTTPTRansport struct {
	mock.Mock
}

// RoundTrip doing fake transport
func (m *MockHTTPTRansport) RoundTrip(r *http.Request) (*http.Response, error) {
	request, err := io.ReadAll(r.Body)
	url := r.URL.String()
	headers := r.Header.Clone()
	if err != nil {
		return &http.Response{}, err
	}
	requestString := string(request)
	args := m.Called(url, headers, requestString)

	return args.Get(0).(*http.Response), args.Error(1)
}

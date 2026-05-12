package mocks

import (
	"io"
	"net/http"

	"github.com/stretchr/testify/mock"
)

// MockHTTPTransport provides universal mock for http.Post function.
type MockHTTPTransport struct {
	mock.Mock
}

// RoundTrip provides fake transport behavior.
func (m *MockHTTPTransport) RoundTrip(r *http.Request) (*http.Response, error) {
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

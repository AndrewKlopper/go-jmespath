package http

import (
	"github.com/AndrewKlopper/go-jmespath/internal/testify/mock"
	"net/http"
)

// TestRoundTripper DEPRECATED USE net/http/httptest
type TestRoundTripper struct {
	mock.Mock
}

// RoundTrip DEPRECATED USE net/http/httptest
func (t *TestRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	args := t.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

package netlox

import (
	"net/http"
)

// newnetloxClient returns a specific HTTP client used when communicating with the netlox API(s)
func newnetloxClient() *http.Client {
	return &http.Client{}
}

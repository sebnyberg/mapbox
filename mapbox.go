package mapbox

import (
	"errors"
	"fmt"
	"net/http"
)

const baseURL = "https://api.mapbox.com"

type Client struct {
	username    string
	accessToken string
	httpClient  *http.Client
}

var (
	ErrValidation = errors.New("validation")
	ErrOperation  = errors.New("upload")
	ErrParse      = errors.New("parse")
	ErrUnexpected = errors.New("unexpected")
)

// NewClient returns a new Mapbox client which interacts with the Mapbox API.
func NewClient(accessToken string, username string) (Client, error) {
	var c Client
	if len(username) == 0 {
		return c, fmt.Errorf("%w: username is required", ErrValidation)
	}
	if len(accessToken) == 0 {
		return c, fmt.Errorf("%w: access token is required", ErrValidation)
	}
	c.username = username
	c.accessToken = accessToken
	c.httpClient = http.DefaultClient
	return c, nil
}

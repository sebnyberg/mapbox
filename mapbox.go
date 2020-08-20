package mapbox

import (
	"errors"
	"fmt"
)

const baseURL = "https://api.mapbox.com"

type client struct {
	Username    string
	AccessToken string
}

var ErrValidation = errors.New("validation")

func (c *client) Validate() error {
	if len(c.Username) == 0 {
		return fmt.Errorf("%w: username is required", ErrValidation)
	}
	if len(c.AccessToken) == 0 {
		return fmt.Errorf("%w: username is required", ErrValidation)
	}
	return nil
}

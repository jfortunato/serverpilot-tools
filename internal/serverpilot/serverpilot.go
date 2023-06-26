package serverpilot

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidRuntime = errors.New("invalid runtime")
)

type Credentials struct {
	ClientId string
	ApiKey   string
}

type App struct {
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	Serverid    string   `json:"serverid"`
	Runtime     Runtime  `json:"runtime"`
	Domains     []string `json:"domains"`
	Datecreated int64    `json:"datecreated"`
}

type AppResponse struct {
	Data []App `json:"data"`
}

type Runtime string

func (r Runtime) Version() (string, error) {
	// Ensure the runtime is prefixed with "php".
	if !strings.HasPrefix(string(r), "php") {
		return "", fmt.Errorf("%w: %s", ErrInvalidRuntime, r)
	}

	// Remove the "php" prefix.
	return string(r[3:]), nil
}

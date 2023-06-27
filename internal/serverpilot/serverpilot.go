package serverpilot

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidRuntime    = errors.New("invalid runtime")
	ErrInvalidDateString = errors.New("invalid date string")
)

type Credentials struct {
	ClientId string
	ApiKey   string
}

type App struct {
	Id          string      `json:"id"`
	Name        string      `json:"name"`
	Serverid    string      `json:"serverid"`
	Runtime     Runtime     `json:"runtime"`
	Domains     []string    `json:"domains"`
	Datecreated DateCreated `json:"datecreated"`
}

type Server struct {
	Id          string      `json:"id"`
	Name        string      `json:"name"`
	Ipaddress   string      `json:"lastaddress"`
	Datecreated DateCreated `json:"datecreated"`
}

type AppResponse struct {
	Data []App `json:"data"`
}

type ServerResponse struct {
	Data []Server `json:"data"`
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

type DateCreated int64

func (d DateCreated) String() string {
	return time.Unix(int64(d), 0).Format("2006-01-02")
}

func DateCreatedFromDate(date string) (DateCreated, error) {
	// Just use a zero value if the date is empty.
	if date == "" {
		return DateCreated(0), nil
	}

	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidDateString, date)
	}

	// Add the time portion to the date
	date += "T00:00:00Z"

	t, _ := time.Parse(time.RFC3339, date)
	return DateCreated(t.Unix()), nil
}

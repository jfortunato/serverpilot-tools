package serverpilot

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	ErrInvalidRuntime = errors.New("invalid runtime")
)

type Credentials struct {
	ClientId string
	ApiKey   string
}

type App struct {
	Id       string  `json:"id"`
	Name     string  `json:"name"`
	Serverid string  `json:"serverid"`
	Runtime  Runtime `json:"runtime"`
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

type ServerPilotClient struct {
	credentials Credentials
}

func NewClient(user, key string) *ServerPilotClient {
	return &ServerPilotClient{
		credentials: Credentials{
			ClientId: user,
			ApiKey:   key,
		},
	}
}

func (c *ServerPilotClient) Get(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error while making request: %s", err)
	}
	req.SetBasicAuth(c.credentials.ClientId, c.credentials.ApiKey)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error while making request: %s", err)
	}
	defer resp.Body.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, resp.Body)

	return buf.String(), nil
}

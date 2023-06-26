package serverpilot

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Makes all the requests to the ServerPilot API. Since we don't want to hammer
// the API, we'll rate limit requests by default.
type serverPilotClient struct {
	credentials    Credentials
	g              httpGetter
	s              sleeper
	hasMadeRequest bool
}

func (c *serverPilotClient) Get(url string) (string, error) {
	// If this is not the first request, sleep for the configured duration.
	if c.hasMadeRequest {
		c.s.Sleep()
	}
	c.hasMadeRequest = true
	return c.g.Get(url)
}

// Constructor for creating our serverPilotClient. User/key are used to authenticate with the ServerPilot API.
func NewClient(user, key string) *serverPilotClient {
	return &serverPilotClient{
		credentials: Credentials{
			ClientId: user,
			ApiKey:   key,
		},
		g: &basicAuthHttpGetter{
			u: user,
			p: key,
		},
		s: &defaultSleeper{},
	}
}

// Interface for making HTTP requests. By using an interface it simplifies testing, because we can keep all the setup/bookkeeping for making a request out of our tests.
type httpGetter interface {
	Get(url string) (string, error)
}

type basicAuthHttpGetter struct {
	u string
	p string
}

func (g *basicAuthHttpGetter) Get(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error while making request: %s", err)
	}
	req.SetBasicAuth(g.u, g.p)
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

type sleeper interface {
	Sleep()
}

type defaultSleeper struct{}

func (s *defaultSleeper) Sleep() {
	// Sleep for 500 milliseconds.
	time.Sleep(500 * time.Millisecond)
}

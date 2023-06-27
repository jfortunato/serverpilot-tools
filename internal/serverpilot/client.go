package serverpilot

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	ErrCouldNotMakeRequest = fmt.Errorf("could not make request to ServerPilot API")
	ErrCouldNotCache       = fmt.Errorf("could not cache response")
)

// Makes all the requests to the ServerPilot API. Since we don't want to hammer
// the API, we'll rate limit requests by default.
type serverPilotClient struct {
	credentials    Credentials
	g              httpGetter
	s              sleeper
	c              cacher
	hasMadeRequest bool
}

func (c *serverPilotClient) Get(url string) (string, error) {
	// Check if we have a cached response for this url.
	if c.c != nil {
		resp, err := c.c.Get(url)
		if err == nil {
			return resp, nil
		}
	}

	// If this is not the first request, sleep for the configured duration.
	if c.hasMadeRequest {
		c.s.Sleep()
	}
	c.hasMadeRequest = true
	resp, err := c.g.Get(url)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrCouldNotMakeRequest, err)
	}

	// Cache the response.
	if c.c != nil {
		err = c.c.Set(url, resp)
		if err != nil {
			return "", fmt.Errorf("%w: %s", ErrCouldNotCache, err)
		}
	}

	return resp, nil
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
		c: &tmpFileCacher{},
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

type cacher interface {
	Get(key string) (string, error)
	Set(key string, value string) error
}

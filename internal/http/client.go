package http

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	ErrCouldNotMakeRequest = fmt.Errorf("could not make request")
	ErrCouldNotCache       = fmt.Errorf("could not cache response")
)

// CachingRateLimitedClient is an interface for making HTTP requests, caching the response, and rate limiting the requests.
type CachingRateLimitedClient interface {
	GetFromCacheOrFetchWithRateLimit(req Request) (string, error)
}

// Request is a struct that represents an HTTP request. It contains the URL and any headers that should be added to the request.
type Request struct {
	Url     string
	Headers map[string]string
}

// Client is a struct that implements the CachingRateLimitedClient interface. It will use the net.Http package to make HTTP requests.
type Client struct {
	*log.Logger
	s              sleeper
	c              cacher
	f              FetchForString
	hasMadeRequest bool
}

// NewClient returns a new Client, configured with objects to make HTTP requests, cache responses, and rate limit requests.
func NewClient(l *log.Logger) *Client {
	return &Client{
		Logger: l,
		s:      &defaultSleeper{},
		c:      &tmpFileCacher{},
		f:      makeHttpFetcher(),
	}
}

// GetFromCacheOrFetchWithRateLimit will check if we have a cached response for the given url. If we do, it will return the cached response.
// If we don't, it will make an HTTP request to the given url, cache the response, and return the response. When making additional requests,
// it will sleep for the configured duration to rate limit the requests.
func (c *Client) GetFromCacheOrFetchWithRateLimit(req Request) (string, error) {
	// Check if we have a cached response for this url.
	if c.c.Has(req.Url) {
		c.Println("cache hit")
		return c.c.Get(req.Url)
	}

	// If this is not the first request, sleep for the configured duration.
	if c.hasMadeRequest {
		c.s.Sleep()
	}

	c.Println("Making http request to", req.Url)
	resp, err := c.f(req)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrCouldNotMakeRequest, err)
	}

	c.hasMadeRequest = true

	// Cache the response.
	err = c.c.Set(req.Url, resp)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrCouldNotCache, err)
	}

	return resp, nil
}

type sleeper interface {
	Sleep()
}

type defaultSleeper struct{}

func (s *defaultSleeper) Sleep() {
	// Sleep for 200 milliseconds.
	time.Sleep(200 * time.Millisecond)
}

type cacher interface {
	Has(key string) bool
	Get(key string) (string, error)
	Set(key string, value string) error
}

// FetchForString will use the net.Http package to make an HTTP request.
type FetchForString func(req Request) (string, error)

func convertRequestToHttpRequest(req Request) (*http.Request, error) {
	// Convert our request into an http.Request.
	r, err := http.NewRequest("GET", req.Url, nil)
	if err != nil {
		return nil, err
	}

	// Add any headers to the request.
	for k, v := range req.Headers {
		r.Header.Add(k, v)
	}

	return r, nil
}

func makeHttpFetcher() FetchForString {
	return func(req Request) (string, error) {
		r, err := convertRequestToHttpRequest(req)
		if err != nil {
			return "", err
		}

		// Make the request.
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(r)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		// Read the response body into a string.
		buf := new(strings.Builder)
		_, err = io.Copy(buf, resp.Body)

		return buf.String(), err
	}
}

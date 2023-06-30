package http

import (
	"errors"
	"gotest.tools/v3/assert"
	"io"
	"log"
	"testing"
)

func TestClient(t *testing.T) {
	t.Run("it should make rate limited requests", func(t *testing.T) {
		var tests = []struct {
			name                         string
			makeGetCalls, wantSleepCalls int
		}{
			{"first request", 1, 0},
			{"second request", 2, 1},
			{"third request", 3, 2},
		}

		for _, tt := range tests {
			sleeper := &SpySleeper{}

			client := newClientWithStubs()
			client.s = sleeper

			// Make the desired number of Get calls.
			for i := 0; i < tt.makeGetCalls; i++ {
				client.GetFromCacheOrFetchWithRateLimit(Request{Url: "https://example.com"})
			}

			// Assert that we got the expected number of sleep calls.
			assert.Equal(t, sleeper.calls, tt.wantSleepCalls, tt.name)
		}
	})

	t.Run("it should not sleep when making 2 requests to 2 different APIs", func(t *testing.T) {
	})

	t.Run("it should return an error that occurs while fetching", func(t *testing.T) {
		client := newClientWithStubs()
		client.f = stubFetcher(errors.New("some http error"))

		_, err := client.GetFromCacheOrFetchWithRateLimit(Request{Url: "https://example.com"})

		assert.ErrorIs(t, err, ErrCouldNotMakeRequest)
	})

	t.Run("it should cache requests made to the same endpoint", func(t *testing.T) {
		var tests = []struct {
			name                       string
			makeGetCalls, wantApiCalls int
		}{
			{"first request", 1, 1},
			{"second request", 2, 1},
			{"third request", 3, 1},
		}

		for _, tt := range tests {
			spyCalls := 0

			client := newClientWithStubs()
			client.c = &InMemoryCacher{}
			client.f = func(req Request) (string, error) {
				spyCalls++
				return "response", nil
			}

			// Make the desired number of Get calls.
			for i := 0; i < tt.makeGetCalls; i++ {
				resp, _ := client.GetFromCacheOrFetchWithRateLimit(Request{Url: "https://example.com"})

				// Assert that we got the expected response each time
				assert.Equal(t, resp, "response", tt.name)
			}

			// Assert that we got the expected number of api calls.
			assert.Equal(t, spyCalls, tt.wantApiCalls, tt.name)
		}
	})

	t.Run("it should return an error if setting a cache value returns an error", func(t *testing.T) {
		client := newClientWithStubs()
		client.c = &InMemoryCacher{setErrStub: errors.New("some cache error")}

		_, err := client.GetFromCacheOrFetchWithRateLimit(Request{Url: "https://example.com"})

		assert.ErrorIs(t, err, ErrCouldNotCache)
	})

	t.Run("it should only cache 200 responses", func(t *testing.T) {
	})
}

func newClientWithStubs() *Client {
	client := NewClient(log.New(io.Discard, "", 0))
	client.s = &SpySleeper{}
	client.c = &NeverCacher{}
	client.f = stubFetcher(nil)
	return client
}

type SpySleeper struct {
	calls int
}

func (s *SpySleeper) Sleep() {
	s.calls++
}

type InMemoryCacher struct {
	cache      map[string]string
	setErrStub error
}

func (c *InMemoryCacher) Has(key string) bool {
	if _, ok := c.cache[key]; !ok {
		return false
	}
	return true
}

func (c *InMemoryCacher) Get(key string) (string, error) {
	if _, ok := c.cache[key]; !ok {
		return "", errors.New("key not found")
	}
	return c.cache[key], nil
}

func (c *InMemoryCacher) Set(key string, value string) error {
	// Initialize the cache if it's nil.
	if c.cache == nil {
		c.cache = make(map[string]string)
	}
	if c.setErrStub != nil {
		return c.setErrStub
	}
	c.cache[key] = value
	return nil
}

type NeverCacher struct{}

func (c *NeverCacher) Has(key string) bool                { return false }
func (c *NeverCacher) Get(key string) (string, error)     { return "", nil }
func (c *NeverCacher) Set(key string, value string) error { return nil }

func stubFetcher(errStub error) FetchForString {
	return func(req Request) (string, error) {
		return "response", errStub
	}
}

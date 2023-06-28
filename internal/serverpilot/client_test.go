package serverpilot

import (
	"errors"
	"gotest.tools/v3/assert"
	"testing"
)

func TestServerPilotClient(t *testing.T) {
	t.Run("it should make rate limited requests", func(t *testing.T) {
		var tests = []struct {
			name                                       string
			makeGetCalls, wantApiCalls, wantSleepCalls int
		}{
			{"first request", 1, 1, 0},
			{"second request", 2, 2, 1},
			{"third request", 3, 3, 2},
		}

		for _, tt := range tests {
			wrapped := &HttpGetterSpy{}
			sleeper := &SpySleeper{}

			client := NewClient("user", "pass")
			client.g = wrapped
			client.s = sleeper
			// Disable caching.
			client.c = nil
			// Make the desired number of Get calls.
			for i := 0; i < tt.makeGetCalls; i++ {
				client.Get("https://api.serverpilot.io/v1/apps")
			}

			// Assert that every Get call resulted in a call to the underlying http client.
			assert.Equal(t, wrapped.calls, tt.wantApiCalls, tt.name)

			// Assert that we got the expected number of sleep calls.
			assert.Equal(t, sleeper.calls, tt.wantSleepCalls, tt.name)
		}
	})

	t.Run("it should return an error if the underlying http client returns an error", func(t *testing.T) {
		client := NewClient("user", "pass")
		client.g = &HttpGetterSpy{errStub: errors.New("some http error")}
		client.s = &SpySleeper{}
		client.c = &InMemorySpyCacher{}

		_, err := client.Get("https://api.serverpilot.io/v1/apps")

		assert.ErrorIs(t, err, ErrCouldNotMakeRequest)
	})

	t.Run("it should cache some requests made to the api for some amount of time", func(t *testing.T) {
		var tests = []struct {
			name                                       string
			makeGetCalls, wantApiCalls, wantSleepCalls int
		}{
			{"first request", 1, 1, 0},
			{"second request", 2, 1, 0},
			{"third request", 3, 1, 0},
		}

		for _, tt := range tests {
			_ = tt
			wrapped := &HttpGetterSpy{}
			sleeper := &SpySleeper{}
			cacher := &InMemorySpyCacher{}

			client := NewClient("user", "pass")
			client.g = wrapped
			client.s = sleeper
			client.c = cacher

			// Make the desired number of Get calls.
			for i := 0; i < tt.makeGetCalls; i++ {
				client.Get("https://api.serverpilot.io/v1/apps")
			}

			// Assert that every Get call resulted in a call to the underlying http client.
			assert.Equal(t, wrapped.calls, tt.wantApiCalls, tt.name)

			// Assert that we got the expected number of sleep calls.
			assert.Equal(t, sleeper.calls, tt.wantSleepCalls, tt.name)
		}
	})

	t.Run("it should return an error if setting a cache value returns an error", func(t *testing.T) {
		client := NewClient("user", "pass")
		client.g = &HttpGetterSpy{}
		client.s = &SpySleeper{}
		client.c = &InMemorySpyCacher{setErrStub: errors.New("some cache error")}

		_, err := client.Get("https://api.serverpilot.io/v1/apps")

		assert.ErrorIs(t, err, ErrCouldNotCache)
	})
}

type HttpGetterSpy struct {
	calls   int
	errStub error
}

func (c *HttpGetterSpy) Get(url string) (string, error) {
	c.calls++
	return "", c.errStub
}

type SpySleeper struct {
	calls int
}

func (s *SpySleeper) Sleep() {
	s.calls++
}

type InMemorySpyCacher struct {
	cache              map[string]string
	getCalls, setCalls int
	setErrStub         error
}

func (s *InMemorySpyCacher) Get(key string) (string, error) {
	s.getCalls++
	if _, ok := s.cache[key]; !ok {
		return "", errors.New("key not found")
	}
	return s.cache[key], nil
}

func (s *InMemorySpyCacher) Set(key string, value string) error {
	s.setCalls++
	// Initialize the cache if it's nil.
	if s.cache == nil {
		s.cache = make(map[string]string)
	}
	if s.setErrStub != nil {
		return s.setErrStub
	}
	s.cache[key] = value
	return nil
}

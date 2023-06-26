package serverpilot

import (
	"testing"
)

func TestServerPilotClient(t *testing.T) {
	t.Run("it should make rate limited requests", func(t *testing.T) {
		var tests = []struct {
			name                         string
			wantGetCalls, wantSleepCalls int
		}{
			{"first request", 1, 0},
			{"second request", 2, 1},
			{"third request", 3, 2},
		}

		for _, tt := range tests {
			wrapped := &HttpGetterSpy{}
			sleeper := &SpySleeper{}

			client := NewClient("user", "pass")
			client.g = wrapped
			client.s = sleeper
			// Make a request for each expected wantGetCalls.
			for i := 0; i < tt.wantGetCalls; i++ {
				client.Get("https://api.serverpilot.io/v1/apps")
			}

			// Assert that every Get call resulted in a call to the underlying http client.
			if wrapped.calls != tt.wantGetCalls {
				t.Errorf("got %d calls to the wrapper client, want %d", wrapped.calls, tt.wantGetCalls)
			}

			// Assert that we got the expected number of sleep calls.
			if sleeper.calls != tt.wantSleepCalls {
				t.Errorf("got %d calls to the sleeper, want %d", sleeper.calls, tt.wantSleepCalls)
			}
		}
	})
}

type HttpGetterSpy struct {
	calls int
}

func (c *HttpGetterSpy) Get(url string) (string, error) {
	c.calls++
	return "", nil
}

type SpySleeper struct {
	calls int
}

func (s *SpySleeper) Sleep() {
	s.calls++
}

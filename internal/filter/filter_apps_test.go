package filter

import (
	"errors"
	"github.com/jfortunato/serverpilot-tools/internal/serverpilot"
	"reflect"
	"testing"
)

func TestFilterApps(t *testing.T) {
	t.Run("it lists all apps with no filter", func(t *testing.T) {
		client := &HttpClientStub{responses: map[string]string{
			"https://api.serverpilot.io/v1/apps": `{"data":[{"name":"app1","runtime":"php7.4"},{"name": "app2","runtime":"php8.2"}]}`,
		}}

		got, err := FilterApps(client, "", "")
		want := []serverpilot.App{
			{Name: "app1", Runtime: "php7.4"},
			{Name: "app2", Runtime: "php8.2"},
		}

		assertEqualSlice(t, got, want)
		assertNoError(t, err)
	})

	t.Run("it handles an error from the http client request", func(t *testing.T) {
		// No stubbed response results in an error.
		client := &HttpClientStub{}

		_, err := FilterApps(client, "", "")

		assertError(t, err, ErrInvalidRequest)
	})

	t.Run("it handles an error while decoding the json response", func(t *testing.T) {
		// The response is valid, but the json is nonsense.
		client := &HttpClientStub{responses: map[string]string{
			"https://api.serverpilot.io/v1/apps": `{nonsense}`,
		}}

		_, err := FilterApps(client, "", "")

		assertError(t, err, ErrInvalidJson)
	})

	t.Run("it filters apps by runtime", func(t *testing.T) {
		var tests = []struct {
			name                   string
			minRuntime, maxRuntime serverpilot.Runtime
			want                   []serverpilot.App
		}{
			{
				"exact",
				"php7.4",
				"php7.4",
				[]serverpilot.App{{Name: "app1", Runtime: "php7.4"}},
			},
			{
				"inside",
				"php7.0",
				"php8.0",
				[]serverpilot.App{{Name: "app1", Runtime: "php7.4"}},
			},
			{
				"greater than",
				"php8.2",
				"",
				[]serverpilot.App{{Name: "app2", Runtime: "php8.2"}},
			},
			{
				"greater than inclusive",
				"php8.2",
				"",
				[]serverpilot.App{{Name: "app2", Runtime: "php8.2"}},
			},
			{
				"less than",
				"",
				"php8.0",
				[]serverpilot.App{{Name: "app1", Runtime: "php7.4"}},
			},
			{
				"less than inclusive",
				"",
				"php7.4",
				[]serverpilot.App{{Name: "app1", Runtime: "php7.4"}},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				client := &HttpClientStub{responses: map[string]string{
					"https://api.serverpilot.io/v1/apps": `{"data":[{"name":"app1","runtime":"php7.4"},{"name": "app2","runtime":"php8.2"}]}`,
				}}

				got, err := FilterApps(client, tt.minRuntime, tt.maxRuntime)

				assertEqualSlice(t, got, tt.want)
				assertNoError(t, err)
			})
		}
	})

	t.Run("it returns an error when supplied an invalid runtime", func(t *testing.T) {
		var tests = []struct {
			name                   string
			minRuntime, maxRuntime serverpilot.Runtime
		}{
			{"invalid min runtime", "7.4", "php8.2"},
			{"invalid min runtime", "php7.4", "8.2"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				client := &HttpClientStub{}

				_, err := FilterApps(client, tt.minRuntime, tt.maxRuntime)

				assertError(t, err, serverpilot.ErrInvalidRuntime)
			})
		}
	})
}

func assertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Errorf("got an error, but didn't want one")
	}
}

func assertError(t *testing.T, got, want error) {
	t.Helper()

	if got == nil {
		t.Fatal("didn't get an error but wanted one")
	}

	if !errors.Is(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func assertEqualSlice(t *testing.T, got, want any) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

type HttpClientStub struct {
	responses map[string]string
}

func (c *HttpClientStub) Get(url string) (string, error) {
	response, ok := c.responses[url]
	if !ok {
		return "", errors.New("stubbed response not found")
	}
	return response, nil
}

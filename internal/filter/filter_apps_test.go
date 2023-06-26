package filter

import (
	"encoding/json"
	"errors"
	"github.com/jfortunato/serverpilot-tools/internal/serverpilot"
	"reflect"
	"testing"
)

func TestFilterApps(t *testing.T) {
	t.Run("it lists all apps with no filter", func(t *testing.T) {
		app1 := genApp(serverpilot.App{Name: "app1", Runtime: "php7.4"})
		app2 := genApp(serverpilot.App{Name: "app2", Runtime: "php8.2"})

		client := &HttpClientStub{responses: map[string]string{
			"https://api.serverpilot.io/v1/apps": responseWithApps([]serverpilot.App{app1, app2}),
		}}

		got, err := FilterApps(client, "", "", 0, 0)
		want := []serverpilot.App{app1, app2}

		assertEqualSlice(t, got, want)
		assertNoError(t, err)
	})

	t.Run("it handles an error from the http client request", func(t *testing.T) {
		// No stubbed response results in an error.
		client := &HttpClientStub{}

		_, err := FilterApps(client, "", "", 0, 0)

		assertError(t, err, ErrInvalidRequest)
	})

	t.Run("it handles an error while decoding the json response", func(t *testing.T) {
		// The response is valid, but the json is nonsense.
		client := &HttpClientStub{responses: map[string]string{
			"https://api.serverpilot.io/v1/apps": `{nonsense}`,
		}}

		_, err := FilterApps(client, "", "", 0, 0)

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
				[]serverpilot.App{genApp(serverpilot.App{Name: "app1", Runtime: "php7.4"})},
			},
			{
				"inside",
				"php7.0",
				"php8.0",
				[]serverpilot.App{genApp(serverpilot.App{Name: "app1", Runtime: "php7.4"})},
			},
			{
				"greater than",
				"php8.0",
				"",
				[]serverpilot.App{genApp(serverpilot.App{Name: "app2", Runtime: "php8.2"})},
			},
			{
				"greater than inclusive",
				"php8.2",
				"",
				[]serverpilot.App{genApp(serverpilot.App{Name: "app2", Runtime: "php8.2"})},
			},
			{
				"less than",
				"",
				"php8.0",
				[]serverpilot.App{genApp(serverpilot.App{Name: "app1", Runtime: "php7.4"})},
			},
			{
				"less than inclusive",
				"",
				"php7.4",
				[]serverpilot.App{genApp(serverpilot.App{Name: "app1", Runtime: "php7.4"})},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				app1 := genApp(serverpilot.App{Name: "app1", Runtime: "php7.4"})
				app2 := genApp(serverpilot.App{Name: "app2", Runtime: "php8.2"})

				client := &HttpClientStub{responses: map[string]string{
					"https://api.serverpilot.io/v1/apps": responseWithApps([]serverpilot.App{app1, app2}),
				}}

				got, err := FilterApps(client, tt.minRuntime, tt.maxRuntime, 0, 0)

				assertEqualSlice(t, got, tt.want)
				assertNoError(t, err)
			})
		}
	})

	t.Run("it filters apps by date created", func(t *testing.T) {
		var tests = []struct {
			name                   string
			minCreated, maxCreated serverpilot.DateCreated
			want                   []serverpilot.App
		}{
			{
				"exact",
				stringToDateCreated("2023-01-01"),
				stringToDateCreated("2023-01-01"),
				[]serverpilot.App{genApp(serverpilot.App{Name: "app1", Datecreated: stringToDateCreated("2023-01-01")})},
			},
			{
				"inside",
				stringToDateCreated("2022-12-01"),
				stringToDateCreated("2023-01-10"),
				[]serverpilot.App{genApp(serverpilot.App{Name: "app1", Datecreated: stringToDateCreated("2023-01-01")})},
			},
			{
				"greater than",
				stringToDateCreated("2023-01-10"),
				0,
				[]serverpilot.App{genApp(serverpilot.App{Name: "app2", Datecreated: stringToDateCreated("2023-02-01")})},
			},
			{
				"greater than inclusive",
				stringToDateCreated("2023-02-01"),
				0,
				[]serverpilot.App{genApp(serverpilot.App{Name: "app2", Datecreated: stringToDateCreated("2023-02-01")})},
			},
			{
				"less than",
				0,
				stringToDateCreated("2023-01-10"),
				[]serverpilot.App{genApp(serverpilot.App{Name: "app1", Datecreated: stringToDateCreated("2023-01-01")})},
			},
			{
				"less than inclusive",
				0,
				stringToDateCreated("2023-01-01"),
				[]serverpilot.App{genApp(serverpilot.App{Name: "app1", Datecreated: stringToDateCreated("2023-01-01")})},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				client := &HttpClientStub{responses: map[string]string{
					"https://api.serverpilot.io/v1/apps": responseWithApps([]serverpilot.App{
						genApp(serverpilot.App{Name: "app1", Datecreated: stringToDateCreated("2023-01-01")}),
						genApp(serverpilot.App{Name: "app2", Datecreated: stringToDateCreated("2023-02-01")}),
					}),
				}}

				got, err := FilterApps(client, "", "", tt.minCreated, tt.maxCreated)

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

				_, err := FilterApps(client, tt.minRuntime, tt.maxRuntime, 0, 0)

				assertError(t, err, serverpilot.ErrInvalidRuntime)
			})
		}
	})
}

// Generates a stubbed App with consistent defaults if not specified.
func genApp(app serverpilot.App) serverpilot.App {
	if app.Runtime == "" {
		app.Runtime = "php7.0"
	}
	if app.Datecreated == 0 {
		app.Datecreated = stringToDateCreated("2022-06-01")
	}
	return app
}

func responseWithApps(apps []serverpilot.App) string {
	j, err := json.Marshal(apps)
	if err != nil {
		panic("error while marshalling apps: " + err.Error())
	}

	return `{"data":` + string(j) + `}`
}

func stringToDateCreated(date string) serverpilot.DateCreated {
	d, _ := serverpilot.DateCreatedFromDate(date)
	return d
}

func assertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Errorf("got an error (%s), but didn't want one", err)
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

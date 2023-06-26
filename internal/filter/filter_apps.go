package filter

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jfortunato/serverpilot-tools/internal/serverpilot"
	"strings"
	"time"
)

var (
	ErrInvalidRequest = errors.New("error while making request")
	ErrInvalidJson    = errors.New("error while decoding json")
)

type HttpClient interface {
	Get(url string) (string, error)
}

func FilterApps(c HttpClient, minRuntime, maxRuntime serverpilot.Runtime, createdAfter, createdBefore serverpilot.DateCreated) ([]serverpilot.App, error) {
	if minRuntime == "" {
		minRuntime = "php0.0.0"
	}

	if maxRuntime == "" {
		maxRuntime = "php999999.9.9"
	}
	minR, err := minRuntime.Version()
	if err != nil {
		return nil, err
	}
	maxR, err := maxRuntime.Version()
	if err != nil {
		return nil, err
	}

	if createdBefore == 0 {
		createdBefore = serverpilot.DateCreated(time.Now().Unix())
	}

	resp, err := c.Get("https://api.serverpilot.io/v1/apps")
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRequest, err)
	}

	// Transform the JSON response into a slice of App structs.
	var appResponse serverpilot.AppResponse

	decoder := json.NewDecoder(strings.NewReader(resp))
	err = decoder.Decode(&appResponse)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidJson, err)
	}

	// Filter the apps by runtime.
	apps := filterByRuntime(appResponse.Data, minR, maxR)
	// Filter the apps by creation date.
	apps = filterByDate(apps, createdAfter, createdBefore)

	return apps, nil
}

func filterByRuntime(apps []serverpilot.App, minR, maxR string) []serverpilot.App {
	var filteredApps []serverpilot.App

	for _, app := range apps {
		appVersion, err := app.Runtime.Version()
		if err != nil {
			panic(err)
		}
		if appVersion >= minR && appVersion <= maxR {
			filteredApps = append(filteredApps, app)
		}
	}

	return filteredApps
}

func filterByDate(apps []serverpilot.App, createdAfter, createdBefore serverpilot.DateCreated) []serverpilot.App {
	var filteredApps []serverpilot.App

	for _, app := range apps {
		if app.Datecreated >= createdAfter && app.Datecreated <= createdBefore {
			filteredApps = append(filteredApps, app)
		}
	}

	return filteredApps
}

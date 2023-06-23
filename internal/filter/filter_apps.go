package filter

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jfortunato/serverpilot-tools/internal/serverpilot"
	"strings"
)

var (
	ErrInvalidRequest = errors.New("error while making request")
	ErrInvalidJson    = errors.New("error while decoding json")
)

type HttpClient interface {
	Get(url string) (string, error)
}

func FilterApps(c HttpClient, minRuntime, maxRuntime serverpilot.Runtime) ([]serverpilot.App, error) {
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
	var filteredApps []serverpilot.App
	for _, app := range appResponse.Data {
		appVersion, _ := app.Runtime.Version()
		if appVersion >= minR && appVersion <= maxR {
			filteredApps = append(filteredApps, app)
		}
	}
	return filteredApps, nil
}

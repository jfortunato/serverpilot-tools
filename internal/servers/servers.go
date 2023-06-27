package servers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jfortunato/serverpilot-tools/internal/filter"
	"github.com/jfortunato/serverpilot-tools/internal/serverpilot"
	"strings"
)

var (
	ErrInvalidRequest = errors.New("error while making request")
	ErrInvalidJson    = errors.New("error while decoding json")
)

func GetServers(c filter.HttpClient) ([]serverpilot.Server, error) {
	resp, err := c.Get("https://api.serverpilot.io/v1/servers")
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRequest, err)
	}

	// Transform the JSON response into a slice of App structs.
	var serverResponse serverpilot.ServerResponse

	decoder := json.NewDecoder(strings.NewReader(resp))
	err = decoder.Decode(&serverResponse)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidJson, err)
	}

	return serverResponse.Data, nil
}

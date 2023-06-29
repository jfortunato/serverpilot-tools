package serverpilot

import (
	"encoding/base64"
	"fmt"
	"github.com/jfortunato/serverpilot-tools/internal/http"
	"log"
)

// Makes all the requests to the ServerPilot API. Since we don't want to hammer
// the API, we'll rate limit requests by default.
type serverPilotClient struct {
	credentials Credentials
	c           http.CachingRateLimitedClient
}

func (c *serverPilotClient) Get(url string) (string, error) {
	basicAuth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.credentials.ClientId, c.credentials.ApiKey)))

	return c.c.GetFromCacheOrFetchWithRateLimit(http.Request{
		Url: url,
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Basic %s", basicAuth),
		},
	})
}

// Constructor for creating our serverPilotClient. User/key are used to authenticate with the ServerPilot API.
func NewClient(l *log.Logger, user, key string) *serverPilotClient {
	return &serverPilotClient{
		credentials: Credentials{
			ClientId: user,
			ApiKey:   key,
		},
		c: http.NewClient(l),
	}
}

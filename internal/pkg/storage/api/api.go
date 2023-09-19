package api

import (
	"bytes"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
)

type ApiConfig struct {
	ApiKey       string
	ApiSignature string
	ApiEndpoint  string
}

// Write content to API Endpoint added to config
func (api ApiConfig) Write(content []byte) (int, error) {
	client := &http.Client{}

	request, err := http.NewRequest(http.MethodPut, api.ApiEndpoint, bytes.NewBuffer(content))
	if err != nil {
		return 0, err
	}

	request.Header.Set("x-api-key", api.ApiKey)
	request.Header.Set("x-api-signature", api.ApiSignature)
	request.Header.Set("Content-Type", "application/json")

	res, err := client.Do(request)

	if err != nil {
		log.Error().Msgf("Error sending request: %s", err)
		return 0, err
	}

	if res.StatusCode != 200 {
		log.Error().Msgf("Error sending request, got StatusCode: %s", res.Status)
		return 0, fmt.Errorf("Got a Status '%s' instead of an '200 OK' response for API request", res.Status)
	}

	return len(content), nil
}

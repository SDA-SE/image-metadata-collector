package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

type ApiConfig struct {
	ApiKey       string
	ApiSignature string
	ApiEndpoint  string
	HTTPHeaders  []string
}

// NewApiStorage initializes and returns a new ApiConfig instance
func NewApi(cfg *ApiConfig) (io.Writer, error) {
	if cfg.ApiKey == "" {
		log.Info().Msg("Api Key not given, do not init ApiStorage")
		return nil, fmt.Errorf("Missing Api Key")
	}
	if cfg.ApiSignature == "" {
		log.Info().Msg("Api Signature not given, do not init ApiStorage")
		return nil, fmt.Errorf("Missing Api Signature")
	}
	if cfg.ApiEndpoint == "" {
		log.Info().Msg("Api Endpoint not given, do not init ApiStorage")
		return nil, fmt.Errorf("Missing Api Endpoint")
	}

	return &ApiConfig{
		ApiKey:       cfg.ApiKey,
		ApiSignature: cfg.ApiSignature,
		ApiEndpoint:  cfg.ApiEndpoint,
		HTTPHeaders:  cfg.HTTPHeaders,
	}, nil
}

// Write content to API Endpoint added to config
func (api ApiConfig) Write(content []byte) (int, error) {
	client := &http.Client{}

	request, err := http.NewRequest(http.MethodPut, api.ApiEndpoint, bytes.NewBuffer(content))
	if err != nil {
		return 0, err
	}

	//hashedKey := sha256.Sum256([]byte(api.ApiKey))
	//hashedKeyStr := hex.EncodeToString(hashedKey[:])

	request.Header.Set("x-api-key", api.ApiKey)
	request.Header.Set("x-api-signature", api.ApiSignature)
	request.Header.Set("Content-Type", "application/json")

	// Add headers to request
	for _, header := range api.HTTPHeaders {
		headerParts := strings.SplitN(header, ":", 2)
		if len(headerParts) != 2 {
			return 0, fmt.Errorf("invalid header format: %s", header)
		}
		request.Header.Set(headerParts[0], headerParts[1])
	}

	res, err := client.Do(request)

	if err != nil {
		log.Error().Msgf("Error sending request: %s", err)
		return 0, err
	}

	if res.StatusCode != 200 {
		log.Error().Msgf("Error sending request, got StatusCode: %s", res.Status)
		return 0, fmt.Errorf("got a Status '%s' instead of an '200 OK' response for API request", res.Status)
	} else {
		log.Info().Msgf("Upload Succeeded, Status: %s", res.Status)
	}

	return len(content), nil
}

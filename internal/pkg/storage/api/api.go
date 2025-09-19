package api

import (
	"bytes"
	"compress/gzip"
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
		return nil, fmt.Errorf("missing Api Key")
	}
	if cfg.ApiSignature == "" {
		log.Info().Msg("Api Signature not given, do not init ApiStorage")
		return nil, fmt.Errorf("missing Api Signature")
	}
	if cfg.ApiEndpoint == "" {
		log.Info().Msg("Api Endpoint not given, do not init ApiStorage")
		return nil, fmt.Errorf("missing Api Endpoint")
	}

	return &ApiConfig{
		ApiKey:       cfg.ApiKey,
		ApiSignature: cfg.ApiSignature,
		ApiEndpoint:  cfg.ApiEndpoint,
		HTTPHeaders:  append([]string(nil), cfg.HTTPHeaders...),
	}, nil
}

func CompressJSONBytes(content []byte) ([]byte, error) {
	// Create a buffer to hold compressed data
	var buf bytes.Buffer

	// Create gzip writer
	gzipWriter, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	// Write JSON data to gzip writer
	_, err = gzipWriter.Write(content)
	if err != nil {
		//nolint:errcheck // Close error not critical in this context
		gzipWriter.Close()
		return nil, fmt.Errorf("failed to write compressed data: %w", err)
	}

	// Close the gzip writer to flush data
	err = gzipWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// Write content to API Endpoint added to config
func (api ApiConfig) Write(content []byte) (int, error) {
	client := &http.Client{}

	contentToSend := content

	// API is max 6MB per request
	contentSize := len(content)
	addCompressHeader := false
	if contentSize > 6*1024*1024 {
		log.Info().Msgf("Content size is too large (%d bytes) compressing it", contentSize)
		compressedContent, err := CompressJSONBytes(content)
		if err != nil {
			return 0, err
		}
		addCompressHeader = true
		contentSize = len(compressedContent)
		contentToSend = compressedContent
		log.Debug().Msgf("Compressed content size is %d bytes", contentSize)
	}

	if contentSize > 6*1024*1024 {
		return 0, fmt.Errorf("content size is too large (%d bytes)", contentSize)
	}

	request, err := http.NewRequest(http.MethodPut, api.ApiEndpoint, bytes.NewBuffer(contentToSend))
	if err != nil {
		return 0, err
	}

	log.Debug().Msgf("Requests content size %d bytes", request.ContentLength)

	//hashedKey := sha256.Sum256([]byte(api.ApiKey))
	//hashedKeyStr := hex.EncodeToString(hashedKey[:])

	request.Header.Set("x-api-key", api.ApiKey)
	request.Header.Set("x-api-signature", api.ApiSignature)
	request.Header.Set("Content-Type", "application/json")
	if addCompressHeader {
		request.Header.Set("Content-Encoding", "gzip")
	}

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

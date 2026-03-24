package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

//const maxDirectUploadSize = 6 * 1024 * 1024
const maxDirectUploadSize = 6 * 1024

type ApiConfig struct {
	ApiKey       string
	ApiSignature string
	ApiEndpoint  string
	HTTPHeaders  []string
	HTTPClient   *http.Client
}

type preparedContent struct {
	body                []byte
	contentEncoding     string
	requiresCompression bool
	requiresMultipart   bool
}

type multipartInitResponse struct {
	UploadID         string `json:"upload_id"`
	Key              string `json:"key"`
	PartSize         int    `json:"part_size"`
	MaxParts         int    `json:"max_parts"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type multipartPartRequest struct {
	UploadID   string `json:"upload_id"`
	Key        string `json:"key"`
	PartNumber int    `json:"part_number"`
}

type multipartPartResponse struct {
	URL              string `json:"url"`
	PartNumber       int    `json:"part_number"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type multipartCompleteRequest struct {
	UploadID        string                `json:"upload_id"`
	Key             string                `json:"key"`
	ContentEncoding string                `json:"content_encoding"`
	Parts           []multipartUploadPart `json:"parts"`
}

type multipartAbortRequest struct {
	UploadID string `json:"upload_id"`
	Key      string `json:"key"`
}

type multipartUploadPart struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
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
		HTTPClient:   cfg.HTTPClient,
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
	prepared, err := api.prepareContent(content)
	if err != nil {
		return 0, err
	}

	if prepared.requiresMultipart {
		if err := api.uploadMultipart(prepared); err != nil {
			return 0, err
		}

		return len(content), nil
	}

	res, err := api.uploadDirect(prepared)
	if err != nil {
		log.Error().Msgf("Error sending request: %s", err)
		return 0, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode == http.StatusOK {
		log.Info().Msgf("Upload Succeeded, Status: %s", res.Status)
		return len(content), nil
	}

	if res.StatusCode == http.StatusRequestEntityTooLarge && prepared.requiresCompression {
		log.Info().Msg("Direct upload returned 413 for large payload, retrying via multipart upload")
		if err := api.uploadMultipart(prepared); err != nil {
			return 0, err
		}

		return len(content), nil
	}

	log.Error().Msgf("Error sending request, got StatusCode: %s", res.Status)
	return 0, fmt.Errorf("got a Status '%s' instead of an '200 OK' response for API request", res.Status)
}

func (api ApiConfig) prepareContent(content []byte) (preparedContent, error) {
	prepared := preparedContent{
		body:            content,
		contentEncoding: "identity",
	}

	if len(content) <= maxDirectUploadSize {
		return prepared, nil
	}

	log.Info().Msgf("Content size is too large (%d bytes) compressing it", len(content))
	compressedContent, err := CompressJSONBytes(content)
	if err != nil {
		return preparedContent{}, err
	}

	prepared.body = compressedContent
	prepared.contentEncoding = "gzip"
	prepared.requiresCompression = true
	log.Debug().Msgf("Compressed content size is %d bytes", len(compressedContent))

	if len(compressedContent) > maxDirectUploadSize {
		prepared.requiresMultipart = true
	}

	return prepared, nil
}

func (api ApiConfig) uploadDirect(prepared preparedContent) (*http.Response, error) {
	request, err := http.NewRequest(http.MethodPut, api.ApiEndpoint, bytes.NewReader(prepared.body))
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("Request content size %d bytes", request.ContentLength)
	if err := api.applyAPIHeaders(request, prepared.contentEncoding); err != nil {
		return nil, err
	}

	return api.httpClient().Do(request)
}

func (api ApiConfig) uploadMultipart(prepared preparedContent) error {
	endpoints, err := api.multipartEndpoints()
	if err != nil {
		return err
	}

	var initialized bool
	var initResponse multipartInitResponse

	abortUpload := func(cause error) error {
		if !initialized {
			return cause
		}

		if abortErr := api.abortMultipartUpload(endpoints.abort, initResponse); abortErr != nil {
			log.Warn().Err(abortErr).Msg("Failed to abort multipart upload after error")
		}

		return cause
	}

	if err := api.postAPIJSON(endpoints.init, nil, &initResponse); err != nil {
		return err
	}

	initialized = true
	if initResponse.UploadID == "" || initResponse.Key == "" {
		return abortUpload(fmt.Errorf("multipart init response missing upload metadata"))
	}
	if initResponse.PartSize <= 0 {
		return abortUpload(fmt.Errorf("multipart init response returned invalid part_size %d", initResponse.PartSize))
	}

	totalParts := (len(prepared.body) + initResponse.PartSize - 1) / initResponse.PartSize
	if totalParts == 0 {
		totalParts = 1
	}
	if initResponse.MaxParts > 0 && totalParts > initResponse.MaxParts {
		return abortUpload(fmt.Errorf("multipart upload requires %d parts, exceeding max_parts %d", totalParts, initResponse.MaxParts))
	}

	uploadedParts := make([]multipartUploadPart, 0, totalParts)
	for partNumber := 1; partNumber <= totalParts; partNumber++ {
		start := (partNumber - 1) * initResponse.PartSize
		end := start + initResponse.PartSize
		if end > len(prepared.body) {
			end = len(prepared.body)
		}

		var partResponse multipartPartResponse
		partRequest := multipartPartRequest{
			UploadID:   initResponse.UploadID,
			Key:        initResponse.Key,
			PartNumber: partNumber,
		}
		if err := api.postAPIJSON(endpoints.part, partRequest, &partResponse); err != nil {
			return abortUpload(err)
		}
		if partResponse.URL == "" {
			return abortUpload(fmt.Errorf("multipart part response missing url for part %d", partNumber))
		}

		etag, err := api.uploadPartToS3(partResponse.URL, prepared.body[start:end])
		if err != nil {
			return abortUpload(err)
		}

		uploadedParts = append(uploadedParts, multipartUploadPart{
			PartNumber: partNumber,
			ETag:       etag,
		})
	}

	completeRequest := multipartCompleteRequest{
		UploadID:        initResponse.UploadID,
		Key:             initResponse.Key,
		ContentEncoding: prepared.contentEncoding,
		Parts:           uploadedParts,
	}

	if err := api.postAPIJSON(endpoints.complete, completeRequest, nil); err != nil {
		return abortUpload(err)
	}

	log.Info().Int("parts", len(uploadedParts)).Msg("Multipart upload succeeded")
	return nil
}

func (api ApiConfig) uploadPartToS3(url string, content []byte) (string, error) {
	request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(content))
	if err != nil {
		return "", err
	}

	res, err := api.httpClient().Do(request)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("multipart S3 upload failed with status '%s'", res.Status)
	}

	etag := res.Header.Get("ETag")
	if etag == "" {
		return "", fmt.Errorf("multipart S3 upload response missing ETag header")
	}

	return etag, nil
}

func (api ApiConfig) abortMultipartUpload(endpoint string, initResponse multipartInitResponse) error {
	body, err := json.Marshal(multipartAbortRequest{
		UploadID: initResponse.UploadID,
		Key:      initResponse.Key,
	})
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodDelete, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	if err := api.applyAPIHeaders(request, "identity"); err != nil {
		return err
	}

	res, err := api.httpClient().Do(request)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("multipart abort failed with status '%s'", res.Status)
	}

	return nil
}

func (api ApiConfig) postAPIJSON(endpoint string, payload interface{}, responseTarget interface{}) error {
	var body io.Reader
	if payload != nil {
		jsonBody, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(jsonBody)
	}

	request, err := http.NewRequest(http.MethodPost, endpoint, body)
	if err != nil {
		return err
	}
	if err := api.applyAPIHeaders(request, "identity"); err != nil {
		return err
	}

	res, err := api.httpClient().Do(request)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("multipart API request to %s returned status '%s'", endpoint, res.Status)
	}

	if responseTarget == nil {
		return nil
	}

	if err := json.NewDecoder(res.Body).Decode(responseTarget); err != nil {
		return fmt.Errorf("failed to decode multipart API response: %w", err)
	}

	return nil
}

func (api ApiConfig) applyAPIHeaders(request *http.Request, contentEncoding string) error {
	request.Header.Set("x-api-key", api.ApiKey)
	request.Header.Set("x-api-signature", api.ApiSignature)
	request.Header.Set("Content-Type", "application/json")
	if contentEncoding != "" && contentEncoding != "identity" {
		request.Header.Set("Content-Encoding", contentEncoding)
	}

	for _, header := range api.HTTPHeaders {
		headerParts := strings.SplitN(header, ":", 2)
		if len(headerParts) != 2 {
			return fmt.Errorf("invalid header format: %s", header)
		}
		request.Header.Set(headerParts[0], headerParts[1])
	}

	return nil
}

func (api ApiConfig) httpClient() *http.Client {
	if api.HTTPClient != nil {
		return api.HTTPClient
	}

	return &http.Client{}
}

func (api ApiConfig) multipartEndpoints() (struct {
	init     string
	part     string
	complete string
	abort    string
}, error) {
	if !strings.HasSuffix(api.ApiEndpoint, "/images") {
		return struct {
			init     string
			part     string
			complete string
			abort    string
		}{}, fmt.Errorf("api endpoint must end with /images to derive multipart endpoints: %s", api.ApiEndpoint)
	}

	base := strings.TrimSuffix(api.ApiEndpoint, "/images")
	return struct {
		init     string
		part     string
		complete string
		abort    string
	}{
		init:     base + "/images/upload/init",
		part:     base + "/images/upload/part",
		complete: base + "/images/upload/complete",
		abort:    base + "/images/upload",
	}, nil
}

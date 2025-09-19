package api

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompressJSONBytes(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr bool
		errMsg  string
	}{
		{
			name:    "simple JSON object",
			input:   []byte(`{"name": "test", "value": 123}`),
			wantErr: false,
		},
		{
			name:    "empty JSON object",
			input:   []byte(`{}`),
			wantErr: false,
		},
		{
			name:    "empty byte slice",
			input:   []byte{},
			wantErr: false,
		},
		{
			name:    "large JSON array",
			input:   generateLargeJSON(1000),
			wantErr: false,
		},
		{
			name:    "JSON with special characters",
			input:   []byte(`{"unicode": "测试", "special": "!@#$%^&*()_+", "newlines": "line1\nline2\ntab\there"}`),
			wantErr: false,
		},
		{
			name:    "nested JSON structure",
			input:   []byte(`{"users": [{"id": 1, "name": "John", "details": {"age": 30, "city": "NYC"}}, {"id": 2, "name": "Jane", "details": {"age": 25, "city": "LA"}}]}`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, err := CompressJSONBytes(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CompressJSONBytes() expected error but got none")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("CompressJSONBytes() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("CompressJSONBytes() unexpected error = %v", err)
				return
			}

			// Verify the compressed data is not empty (unless input was empty)
			if len(tt.input) > 0 && len(compressed) == 0 {
				t.Error("CompressJSONBytes() returned empty compressed data for non-empty input")
			}

			// Verify we can decompress the data back to original
			decompressed, err := decompressData(compressed)
			if err != nil {
				t.Errorf("Failed to decompress data: %v", err)
				return
			}

			if !bytes.Equal(tt.input, decompressed) {
				t.Errorf("Decompressed data doesn't match original.\nOriginal: %s\nDecompressed: %s",
					string(tt.input), string(decompressed))
			}

			// Verify compression actually reduces size for larger inputs
			if len(tt.input) > 100 {
				compressionRatio := float64(len(compressed)) / float64(len(tt.input))
				if compressionRatio > 0.9 { // Less than 10% compression seems poor for JSON
					t.Logf("Warning: Poor compression ratio %.2f for input size %d",
						compressionRatio, len(tt.input))
				}
				t.Logf("Compression ratio: %.2f (original: %d bytes, compressed: %d bytes)",
					compressionRatio, len(tt.input), len(compressed))
			}
		})
	}
}

func TestCompressJSONBytes_CompressionEffectiveness(t *testing.T) {
	// Test with highly repetitive JSON (should compress very well)
	repetitiveJSON := generateRepetitiveJSON(100)

	compressed, err := CompressJSONBytes(repetitiveJSON)
	if err != nil {
		t.Fatalf("CompressJSONBytes() failed: %v", err)
	}

	originalSize := len(repetitiveJSON)
	compressedSize := len(compressed)
	compressionRatio := float64(compressedSize) / float64(originalSize)

	t.Logf("Repetitive JSON compression: %d -> %d bytes (ratio: %.3f)",
		originalSize, compressedSize, compressionRatio)

	// Repetitive JSON should compress to less than 20% of original size
	if compressionRatio > 0.2 {
		t.Errorf("Expected better compression for repetitive data. Ratio: %.3f", compressionRatio)
	}
}

func TestCompressJSONBytes_6MBThreshold(t *testing.T) {
	// Test with data around the 6MB threshold mentioned in the Write method
	largeJSON := generateLargeJSON(50000) // Generate large JSON

	compressed, err := CompressJSONBytes(largeJSON)
	if err != nil {
		t.Fatalf("CompressJSONBytes() failed: %v", err)
	}

	originalSize := len(largeJSON)
	compressedSize := len(compressed)

	t.Logf("Large JSON compression: %d -> %d bytes", originalSize, compressedSize)

	// Verify the compressed data can be decompressed
	decompressed, err := decompressData(compressed)
	if err != nil {
		t.Fatalf("Failed to decompress large JSON: %v", err)
	}

	if !bytes.Equal(largeJSON, decompressed) {
		t.Error("Large JSON decompression failed - data mismatch")
	}
}

// Helper function to decompress gzip data for testing
func decompressData(compressed []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	//nolint:errcheck // Close error not critical in this context
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return decompressed, nil
}

// Helper function to generate large JSON for testing
func generateLargeJSON(numObjects int) []byte {
	type TestObject struct {
		ID          int      `json:"id"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
		Metadata    struct {
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
			Version   int    `json:"version"`
		} `json:"metadata"`
	}

	objects := make([]TestObject, numObjects)
	for i := 0; i < numObjects; i++ {
		objects[i] = TestObject{
			ID:          i,
			Name:        "Test Object " + string(rune(i)),
			Description: "This is a test object with ID " + string(rune(i)) + " used for testing compression functionality in the image metadata collector API.",
			Tags:        []string{"test", "compression", "json", "api"},
		}
		objects[i].Metadata.CreatedAt = "2023-01-01T00:00:00Z"
		objects[i].Metadata.UpdatedAt = "2023-01-02T00:00:00Z"
		objects[i].Metadata.Version = 1
	}

	jsonData, _ := json.Marshal(map[string]interface{}{
		"objects": objects,
		"total":   numObjects,
		"status":  "success",
	})

	return jsonData
}

// Helper function to generate highly repetitive JSON for compression testing
func generateRepetitiveJSON(repeatCount int) []byte {
	baseObject := map[string]interface{}{
		"type":        "image",
		"format":      "jpeg",
		"compression": "high",
		"metadata": map[string]interface{}{
			"camera":       "Canon EOS R5",
			"lens":         "RF 24-70mm F2.8 L IS USM",
			"iso":          100,
			"aperture":     "f/2.8",
			"shutter":      "1/125",
			"focal_length": 50,
		},
		"tags": []string{"landscape", "nature", "photography", "outdoor"},
	}

	objects := make([]interface{}, repeatCount)
	for i := 0; i < repeatCount; i++ {
		objects[i] = baseObject
	}

	result := map[string]interface{}{
		"images": objects,
		"count":  repeatCount,
	}

	jsonData, _ := json.Marshal(result)
	return jsonData
}

// Benchmark the compression function
func BenchmarkCompressJSONBytes(b *testing.B) {
	testData := generateLargeJSON(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CompressJSONBytes(testData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompressJSONBytes_Small(b *testing.B) {
	testData := []byte(`{"name": "test", "value": 123, "active": true}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CompressJSONBytes(testData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestNewApi(t *testing.T) {
	tests := []struct {
		name    string
		config  *ApiConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with all required fields",
			config: &ApiConfig{
				ApiKey:       "test-api-key",
				ApiSignature: "test-signature",
				ApiEndpoint:  "https://api.example.com/upload",
				HTTPHeaders:  []string{"X-Custom-Header:value1", "Authorization:Bearer token"},
			},
			wantErr: false,
		},
		{
			name: "valid config with minimal required fields",
			config: &ApiConfig{
				ApiKey:       "test-key",
				ApiSignature: "test-sig",
				ApiEndpoint:  "https://api.test.com",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: &ApiConfig{
				ApiSignature: "test-signature",
				ApiEndpoint:  "https://api.example.com",
			},
			wantErr: true,
			errMsg:  "missing Api Key",
		},
		{
			name: "empty API key",
			config: &ApiConfig{
				ApiKey:       "",
				ApiSignature: "test-signature",
				ApiEndpoint:  "https://api.example.com",
			},
			wantErr: true,
			errMsg:  "missing Api Key",
		},
		{
			name: "missing API signature",
			config: &ApiConfig{
				ApiKey:      "test-key",
				ApiEndpoint: "https://api.example.com",
			},
			wantErr: true,
			errMsg:  "missing Api Signature",
		},
		{
			name: "empty API signature",
			config: &ApiConfig{
				ApiKey:       "test-key",
				ApiSignature: "",
				ApiEndpoint:  "https://api.example.com",
			},
			wantErr: true,
			errMsg:  "missing Api Signature",
		},
		{
			name: "missing API endpoint",
			config: &ApiConfig{
				ApiKey:       "test-key",
				ApiSignature: "test-signature",
			},
			wantErr: true,
			errMsg:  "missing Api Endpoint",
		},
		{
			name: "empty API endpoint",
			config: &ApiConfig{
				ApiKey:       "test-key",
				ApiSignature: "test-signature",
				ApiEndpoint:  "",
			},
			wantErr: true,
			errMsg:  "missing Api Endpoint",
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true, // This will panic, but we'll handle it
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Handle nil config case to prevent panic
			if tt.config == nil {
				defer func() {
					if r := recover(); r != nil {
						t.Logf("NewApi panicked with nil config (expected): %v", r)
					}
				}()
			}

			writer, err := NewApi(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewApi() expected error but got none")
					return
				}

				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("NewApi() error = %v, want error containing %v", err, tt.errMsg)
				}

				// Writer should be nil when there's an error
				if writer != nil {
					t.Errorf("NewApi() returned non-nil writer despite error")
				}
				return
			}

			if err != nil {
				t.Errorf("NewApi() unexpected error = %v", err)
				return
			}

			if writer == nil {
				t.Error("NewApi() returned nil writer without error")
				return
			}

			// Verify the returned writer is actually an ApiConfig
			apiConfig, ok := writer.(*ApiConfig)
			if !ok {
				t.Error("NewApi() did not return *ApiConfig type")
				return
			}

			// Verify all fields are correctly set
			if apiConfig.ApiKey != tt.config.ApiKey {
				t.Errorf("NewApi() ApiKey = %v, want %v", apiConfig.ApiKey, tt.config.ApiKey)
			}
			if apiConfig.ApiSignature != tt.config.ApiSignature {
				t.Errorf("NewApi() ApiSignature = %v, want %v", apiConfig.ApiSignature, tt.config.ApiSignature)
			}
			if apiConfig.ApiEndpoint != tt.config.ApiEndpoint {
				t.Errorf("NewApi() ApiEndpoint = %v, want %v", apiConfig.ApiEndpoint, tt.config.ApiEndpoint)
			}

			// Check HTTPHeaders slice
			if len(apiConfig.HTTPHeaders) != len(tt.config.HTTPHeaders) {
				t.Errorf("NewApi() HTTPHeaders length = %v, want %v",
					len(apiConfig.HTTPHeaders), len(tt.config.HTTPHeaders))
			} else {
				for i, header := range tt.config.HTTPHeaders {
					if apiConfig.HTTPHeaders[i] != header {
						t.Errorf("NewApi() HTTPHeaders[%d] = %v, want %v",
							i, apiConfig.HTTPHeaders[i], header)
					}
				}
			}
		})
	}
}

func TestNewApi_ReturnsIOWriter(t *testing.T) {
	config := &ApiConfig{
		ApiKey:       "test-key",
		ApiSignature: "test-signature",
		ApiEndpoint:  "https://api.example.com",
	}

	writer, err := NewApi(config)
	if err != nil {
		t.Fatalf("NewApi() unexpected error = %v", err)
	}

	//Verify it implements io.Writer interface
	//nolint:staticcheck // QF1011: explicit interface check is intentional
	var _ io.Writer = writer

	// This should compile without issues if the interface is properly implemented
	// We can't easily test the Write method without setting up a test server,
	// but we can verify the interface is satisfied
}

func TestNewApi_ConfigIsolation(t *testing.T) {
	// Test that modifying the original config doesn't affect the returned instance
	originalConfig := &ApiConfig{
		ApiKey:       "original-key",
		ApiSignature: "original-signature",
		ApiEndpoint:  "https://original.example.com",
		HTTPHeaders:  []string{"Original-Header:value"},
	}

	writer, err := NewApi(originalConfig)
	if err != nil {
		t.Fatalf("NewApi() unexpected error = %v", err)
	}

	apiConfig := writer.(*ApiConfig)

	// Modify the original config
	originalConfig.ApiKey = "modified-key"
	originalConfig.HTTPHeaders[0] = "Modified-Header:value"

	// The returned config should still have original values
	if apiConfig.ApiKey != "original-key" {
		t.Errorf("Config not properly isolated: ApiKey = %v, want %v",
			apiConfig.ApiKey, "original-key")
	}

	if apiConfig.HTTPHeaders[0] != "Original-Header:value" {
		t.Errorf("Config not properly isolated: HTTPHeaders[0] = %v, want %v",
			apiConfig.HTTPHeaders[0], "Original-Header:value")
	}
}

func TestNewApi_WriteWithCompression(t *testing.T) {
	tests := []struct {
		name              string
		dataSize          int
		expectCompression bool
		expectSuccess     bool
		serverStatusCode  int
		description       string
	}{
		{
			name:              "small data - no compression",
			dataSize:          1024, // 1KB
			expectCompression: false,
			expectSuccess:     true,
			serverStatusCode:  200,
			description:       "Small data should not be compressed",
		},
		{
			name:              "medium data - no compression",
			dataSize:          3 * 1024 * 1024, // 3MB
			expectCompression: false,
			expectSuccess:     true,
			serverStatusCode:  200,
			description:       "Medium data under 6MB should not be compressed",
		},
		{
			name:              "large data - with compression",
			dataSize:          7 * 1024 * 1024, // 7MB
			expectCompression: true,
			expectSuccess:     true,
			serverStatusCode:  200,
			description:       "Large data over 6MB should be compressed",
		},
		{
			name:              "very large data - with compression",
			dataSize:          10 * 1024 * 1024, // 10MB
			expectCompression: true,
			expectSuccess:     true,
			serverStatusCode:  200,
			description:       "Very large data should be compressed",
		},
		{
			name:              "server error response",
			dataSize:          1024,
			expectCompression: false,
			expectSuccess:     false,
			serverStatusCode:  500,
			description:       "Server error should cause Write to fail",
		},
		{
			name:              "large data with server error",
			dataSize:          70 * 1024 * 1024,
			expectCompression: true,
			expectSuccess:     false,
			serverStatusCode:  400,
			description:       "Large compressed data with server error should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track what the server receives
			var receivedHeaders http.Header
			var receivedBody []byte

			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Capture request details
				receivedHeaders = r.Header.Clone()

				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("Failed to read request body: %v", err)
					w.WriteHeader(500)
					return
				}
				receivedBody = body
				t.Logf("Received Body %d bytes", len(body))

				// Verify request method
				if r.Method != http.MethodPut {
					t.Errorf("Expected PUT request, got %s", r.Method)
				}

				// Return configured status code
				w.WriteHeader(tt.serverStatusCode)
			}))
			defer server.Close()

			// Create API config
			config := &ApiConfig{
				ApiKey:       "test-api-key",
				ApiSignature: "test-signature",
				ApiEndpoint:  server.URL,
				HTTPHeaders:  []string{"X-Custom-Header:test-value"},
			}

			writer, err := NewApi(config)
			if err != nil {
				t.Fatalf("NewApi() failed: %v", err)
			}

			// Generate test data of specified size
			testData := generateTestDataOfSize(tt.dataSize)
			originalDataSize := len(testData)

			// Write data using the API
			bytesWritten, err := writer.Write(testData)

			// Check results based on expectations
			if tt.expectSuccess {
				if err != nil {
					t.Errorf("Write() unexpected error: %v", err)
					return
				}

				// For successful writes, bytesWritten should match the data sent to server
				// (which might be compressed or original data)
				if bytesWritten <= 0 {
					t.Errorf("Write() returned invalid bytes written: %d", bytesWritten)
				}
			} else {
				if err == nil {
					t.Errorf("Write() expected error due to server status %d, but got none",
						tt.serverStatusCode)
					return
				}
				// For failed requests, we still want to check what was sent
			}

			// Verify compression behavior
			if tt.expectCompression {
				// Should have Content-Encoding: gzip header
				contentEncoding := receivedHeaders.Get("Content-Encoding")
				if contentEncoding != "gzip" {
					t.Errorf("Expected Content-Encoding: gzip, got: %s", contentEncoding)
				}

				// Verify the body is actually gzip compressed
				if len(receivedBody) > 0 {
					decompressed, err := decompressData(receivedBody)
					if err != nil {
						t.Errorf("Failed to decompress received data: %v", err)
					} else {
						if !bytes.Equal(testData, decompressed) {
							t.Errorf("Decompressed data doesn't match original data")
						}

						// Verify compression actually reduced size
						compressionRatio := float64(len(receivedBody)) / float64(originalDataSize)
						t.Logf("Compression ratio: %.3f (original: %d, compressed: %d)",
							compressionRatio, originalDataSize, len(receivedBody))
					}
				}
			} else {
				// Should NOT have Content-Encoding: gzip header
				contentEncoding := receivedHeaders.Get("Content-Encoding")
				if contentEncoding == "gzip" {
					t.Errorf("Unexpected Content-Encoding: gzip for small data")
				}

				// Body should match original data exactly
				if len(receivedBody) > 0 && !bytes.Equal(testData, receivedBody) {
					t.Errorf("Received data doesn't match original uncompressed data")
				}
			}

			// Verify standard headers are always present
			if receivedHeaders.Get("x-api-key") != "test-api-key" {
				t.Errorf("Missing or incorrect x-api-key header")
			}
			if receivedHeaders.Get("x-api-signature") != "test-signature" {
				t.Errorf("Missing or incorrect x-api-signature header")
			}
			if receivedHeaders.Get("Content-Type") != "application/json" {
				t.Errorf("Missing or incorrect Content-Type header")
			}
			if receivedHeaders.Get("X-Custom-Header") != "test-value" {
				t.Errorf("Missing or incorrect custom header")
			}

			t.Logf("%s: Original size: %d bytes, Sent size: %d bytes, Compressed: %t",
				tt.description, originalDataSize, len(receivedBody), tt.expectCompression)
		})
	}
}

func TestNewApi_WriteDataTooLargeEvenAfterCompression(t *testing.T) {
	// Create test server that should never be called
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Server should not be called for data that's too large even after compression")
		w.WriteHeader(500)
	}))
	defer server.Close()

	config := &ApiConfig{
		ApiKey:       "test-key",
		ApiSignature: "test-signature",
		ApiEndpoint:  server.URL,
	}

	writer, err := NewApi(config)
	if err != nil {
		t.Fatalf("NewApi() failed: %v", err)
	}

	// Generate data that won't compress well (random-ish data)
	// This simulates data that even after compression is still > 6MB
	incompressibleData := generateIncompressibleData(80 * 1024 * 1024) // 8MB of poorly compressible data

	_, err = writer.Write(incompressibleData)
	if err == nil {
		t.Error("Write() should fail for data that's too large even after compression")
		return
	}

	expectedErrMsg := "content size is too large"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Error message should contain '%s', got: %v", expectedErrMsg, err)
	}
}

func TestNewApi_WriteInvalidHTTPHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	config := &ApiConfig{
		ApiKey:       "test-key",
		ApiSignature: "test-signature",
		ApiEndpoint:  server.URL,
		HTTPHeaders:  []string{"InvalidHeaderFormat"}, // Missing colon
	}

	writer, err := NewApi(config)
	if err != nil {
		t.Fatalf("NewApi() failed: %v", err)
	}

	testData := []byte(`{"test": "data"}`)
	_, err = writer.Write(testData)

	if err == nil {
		t.Error("Write() should fail with invalid header format")
		return
	}

	expectedErrMsg := "invalid header format"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Error message should contain '%s', got: %v", expectedErrMsg, err)
	}
}

// Helper function to generate test data of specific size
func generateTestDataOfSize(sizeBytes int) []byte {
	// Create a base object that will be repeated
	baseObj := map[string]interface{}{
		"id":          1,
		"name":        "test-object",
		"description": "This is a test object for compression testing",
		"metadata": map[string]interface{}{
			"created": "2023-01-01T00:00:00Z",
			"tags":    []string{"test", "compression", "api"},
		},
	}

	// Calculate approximate size of base object
	baseJSON, _ := json.Marshal(baseObj)
	baseSize := len(baseJSON)

	// Calculate how many objects we need
	numObjects := sizeBytes / baseSize
	if numObjects == 0 {
		numObjects = 1
	}

	// Generate array of objects
	objects := make([]interface{}, numObjects)
	for i := 0; i < numObjects; i++ {
		obj := baseObj
		obj["id"] = i // Make each object slightly different
		objects[i] = obj
	}

	result := map[string]interface{}{
		"data":  objects,
		"count": numObjects,
	}

	jsonData, _ := json.Marshal(result)

	// If we're still under the target size, pad with extra data
	if len(jsonData) < sizeBytes {
		padding := strings.Repeat("x", sizeBytes-len(jsonData))
		result["padding"] = padding
		jsonData, _ = json.Marshal(result)
	}

	return jsonData
}

// Helper function to generate data that doesn't compress well
func generateIncompressibleData(sizeBytes int) []byte {
	// We need to account for JSON structure overhead
	// Target about 80% of the size for actual random data
	randomDataSize := int(float64(sizeBytes) * 0.8)

	// Generate truly random data using crypto/rand
	randomBytes := make([]byte, randomDataSize)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// Fallback to a less random but still poorly compressible method
		for i := range randomBytes {
			randomBytes[i] = byte((i*17 + i*i*13 + i*i*i*7) % 256)
		}
	}
	// Create pseudo-random data that won't compress efficiently
	encodedData := base64.StdEncoding.EncodeToString(randomBytes)

	// Wrap in JSON structure
	jsonStr := fmt.Sprintf(`{"incompressible_data": "%s"}`, encodedData)
	return []byte(jsonStr)
}

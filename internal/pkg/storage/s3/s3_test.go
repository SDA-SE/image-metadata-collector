package s3

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/smithy-go/logging"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// createTestBucket creates a bucket in the fake S3 for testing
func createTestBucket(t *testing.T, backend *s3mem.Backend, bucketName string) {
	err := backend.CreateBucket(bucketName)
	if err != nil {
		t.Fatalf("Failed to create test bucket: %v", err)
	}
}

func TestNewS3(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *S3Config
		fileName string
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid config with endpoint",
			cfg: &S3Config{
				S3BucketName: "test-bucket",
				S3Endpoint:   "http://localhost:9000",
				S3Region:     "us-east-1",
				S3Insecure:   true,
			},
			fileName: "test-file.txt",
			wantErr:  false,
		},
		{
			name: "valid config without endpoint",
			cfg: &S3Config{
				S3BucketName: "test-bucket",
				S3Region:     "us-west-2",
				S3Insecure:   false,
			},
			fileName: "test-file.txt",
			wantErr:  false,
		},
		{
			name: "empty bucket name",
			cfg: &S3Config{
				S3BucketName: "",
				S3Endpoint:   "http://localhost:9000",
				S3Region:     "us-east-1",
			},
			fileName: "test-file.txt",
			wantErr:  true,
			errMsg:   "S3_BUCKET is not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewS3(tt.cfg, tt.fileName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewS3() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("NewS3() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("NewS3() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify client fields
			if client.bucket != tt.cfg.S3BucketName {
				t.Errorf("client.bucket = %v, want %v", client.bucket, tt.cfg.S3BucketName)
			}
			if client.endpoint != tt.cfg.S3Endpoint {
				t.Errorf("client.endpoint = %v, want %v", client.endpoint, tt.cfg.S3Endpoint)
			}
			if client.insecure != tt.cfg.S3Insecure {
				t.Errorf("client.insecure = %v, want %v", client.insecure, tt.cfg.S3Insecure)
			}
			if client.region != tt.cfg.S3Region {
				t.Errorf("client.region = %v, want %v", client.region, tt.cfg.S3Region)
			}
			if client.fileName != tt.fileName {
				t.Errorf("client.fileName = %v, want %v", client.fileName, tt.fileName)
			}

			// Test forcePathStyle logic
			expectedForcePathStyle := tt.cfg.S3Endpoint != ""
			if client.forcePathStyle != expectedForcePathStyle {
				t.Errorf("client.forcePathStyle = %v, want %v", client.forcePathStyle, expectedForcePathStyle)
			}
		})
	}
}

func TestZerologLogger_Logf(t *testing.T) {
	// Capture log output for testing
	var buf bytes.Buffer
	oldLogger := log.Logger
	defer func() {
		log.Logger = oldLogger
	}()
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	logger := zerologLogger{}

	tests := []struct {
		name           string
		classification logging.Classification
		format         string
		args           []interface{}
		expectedLevel  string
	}{
		{
			name:           "debug level",
			classification: logging.Debug,
			format:         "debug message: %s",
			args:           []interface{}{"test"},
			expectedLevel:  "debug",
		},
		{
			name:           "warn level",
			classification: logging.Warn,
			format:         "warn message: %s",
			args:           []interface{}{"test"},
			expectedLevel:  "warn",
		},
		{
			name:           "default (info) level",
			classification: logging.Classification("unknown"),
			format:         "info message: %s",
			args:           []interface{}{"test"},
			expectedLevel:  "info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger.Logf(tt.classification, tt.format, tt.args...)

			output := buf.String()
			if !strings.Contains(output, tt.expectedLevel) {
				t.Errorf("Expected log level %s not found in output: %s", tt.expectedLevel, output)
			}
		})
	}
}

func TestS3Client_Write_Success(t *testing.T) {
	// Create backend to pre-create bucket
	backend := s3mem.New()
	createTestBucket(t, backend, "test-bucket")

	// Create new fake S3 server with the bucket
	faker := gofakes3.New(backend)
	ts := httptest.NewServer(faker.Server())
	defer ts.Close()

	cfg := &S3Config{
		S3BucketName: "test-bucket",
		S3Endpoint:   ts.URL,
		S3Region:     "us-east-1",
		S3Insecure:   true,
	}

	client, err := NewS3(cfg, "test-file.txt")
	if err != nil {
		t.Fatalf("NewS3() error = %v", err)
	}

	testContent := []byte("Hello, World!")
	n, err := client.Write(testContent)

	if err != nil {
		t.Errorf("Write() error = %v", err)
		return
	}

	if n != len(testContent) {
		t.Errorf("Write() returned %d bytes written, want %d", n, len(testContent))
	}

	// Verify the file was actually uploaded
	obj, err := backend.GetObject("test-bucket", "test-file.txt", nil)
	if err != nil {
		t.Errorf("Failed to get uploaded object: %v", err)
		return
	}

	uploadedContent := make([]byte, len(testContent))
	_, err = obj.Contents.Read(uploadedContent)
	if err != nil {
		t.Errorf("Failed to read uploaded content: %v", err)
		return
	}

	if !bytes.Equal(uploadedContent, testContent) {
		t.Errorf("Uploaded content = %v, want %v", uploadedContent, testContent)
	}
}

func TestS3Client_Write_EmptyContent(t *testing.T) {
	// Setup fake S3 server
	backend := s3mem.New()
	createTestBucket(t, backend, "test-bucket")

	faker := gofakes3.New(backend)
	ts := httptest.NewServer(faker.Server())
	defer ts.Close()

	cfg := &S3Config{
		S3BucketName: "test-bucket",
		S3Endpoint:   ts.URL,
		S3Region:     "us-east-1",
		S3Insecure:   true,
	}

	client, err := NewS3(cfg, "empty-file.txt")
	if err != nil {
		t.Fatalf("NewS3() error = %v", err)
	}

	emptyContent := []byte("")
	n, err := client.Write(emptyContent)

	if err != nil {
		t.Errorf("Write() error = %v", err)
		return
	}

	if n != 0 {
		t.Errorf("Write() returned %d bytes written for empty content, want 0", n)
	}

	// Verify empty file was uploaded
	obj, err := backend.GetObject("test-bucket", "empty-file.txt", nil)
	if err != nil {
		t.Errorf("Failed to get uploaded empty object: %v", err)
		return
	}

	uploadedContent := make([]byte, 1) // Try to read 1 byte
	bytesRead, err := obj.Contents.Read(uploadedContent)
	if err != nil && err.Error() != "EOF" {
		t.Errorf("Unexpected error reading empty content: %v", err)
		return
	}

	if bytesRead != 0 {
		t.Errorf("Expected 0 bytes in empty file, got %d", bytesRead)
	}
}

func TestS3Client_Write_LargeContent(t *testing.T) {
	// Setup fake S3 server
	backend := s3mem.New()
	createTestBucket(t, backend, "test-bucket")

	faker := gofakes3.New(backend)
	ts := httptest.NewServer(faker.Server())
	defer ts.Close()

	cfg := &S3Config{
		S3BucketName: "test-bucket",
		S3Endpoint:   ts.URL,
		S3Region:     "us-east-1",
		S3Insecure:   true,
	}

	client, err := NewS3(cfg, "large-file.txt")
	if err != nil {
		t.Fatalf("NewS3() error = %v", err)
	}

	// Create 1MB of test content
	largeContent := bytes.Repeat([]byte("Large content test data. "), 40000) // ~1MB
	n, err := client.Write(largeContent)

	if err != nil {
		t.Errorf("Write() error = %v", err)
		return
	}

	if n != len(largeContent) {
		t.Errorf("Write() returned %d bytes written, want %d", n, len(largeContent))
	}

	// Verify the large file was uploaded correctly
	obj, err := backend.GetObject("test-bucket", "large-file.txt", nil)
	if err != nil {
		t.Errorf("Failed to get uploaded large object: %v", err)
		return
	}

	// Read first 100 bytes to verify content
	uploadedContent := make([]byte, 100)
	_, err = obj.Contents.Read(uploadedContent)
	if err != nil {
		t.Errorf("Failed to read uploaded large content: %v", err)
		return
	}

	expectedStart := largeContent[:100]
	if !bytes.Equal(uploadedContent, expectedStart) {
		t.Errorf("Uploaded content start doesn't match expected")
	}
}

func TestS3Client_Write_NonExistentBucket(t *testing.T) {
	// Setup fake S3 server without creating bucket
	backend := s3mem.New()
	faker := gofakes3.New(backend)
	ts := httptest.NewServer(faker.Server())
	defer ts.Close()

	cfg := &S3Config{
		S3BucketName: "non-existent-bucket",
		S3Endpoint:   ts.URL,
		S3Region:     "us-east-1",
		S3Insecure:   true,
	}

	client, err := NewS3(cfg, "test-file.txt")
	if err != nil {
		t.Fatalf("NewS3() error = %v", err)
	}

	testContent := []byte("Hello, World!")
	n, err := client.Write(testContent)

	if err == nil {
		t.Error("Write() expected error for non-existent bucket, got nil")
		return
	}

	if n != 0 {
		t.Errorf("Write() returned %d bytes written on error, want 0", n)
	}

	// Error should mention the bucket doesn't exist
	if !strings.Contains(strings.ToLower(err.Error()), "nosuchbucket") &&
		!strings.Contains(strings.ToLower(err.Error()), "bucket") {
		t.Errorf("Expected bucket-related error, got: %v", err)
	}
}

func TestS3Client_Write_MultipleFiles(t *testing.T) {
	// Setup fake S3 server
	backend := s3mem.New()
	createTestBucket(t, backend, "multi-test-bucket")

	faker := gofakes3.New(backend)
	ts := httptest.NewServer(faker.Server())
	defer ts.Close()

	cfg := &S3Config{
		S3BucketName: "multi-test-bucket",
		S3Endpoint:   ts.URL,
		S3Region:     "us-east-1",
		S3Insecure:   true,
	}

	// Test uploading multiple files
	files := []struct {
		fileName string
		content  []byte
	}{
		{"file1.txt", []byte("Content of file 1")},
		{"file2.txt", []byte("Content of file 2")},
		{"data/file3.txt", []byte("Content of file 3 in subdirectory")},
	}

	for _, file := range files {
		client, err := NewS3(cfg, file.fileName)
		if err != nil {
			t.Fatalf("NewS3() error for %s = %v", file.fileName, err)
		}

		n, err := client.Write(file.content)
		if err != nil {
			t.Errorf("Write() error for %s = %v", file.fileName, err)
			continue
		}

		if n != len(file.content) {
			t.Errorf("Write() for %s returned %d bytes written, want %d", file.fileName, n, len(file.content))
		}

		// Verify each file was uploaded correctly
		obj, err := backend.GetObject("multi-test-bucket", file.fileName, nil)
		if err != nil {
			t.Errorf("Failed to get uploaded object %s: %v", file.fileName, err)
			continue
		}

		uploadedContent := make([]byte, len(file.content))
		_, err = obj.Contents.Read(uploadedContent)
		if err != nil {
			t.Errorf("Failed to read uploaded content for %s: %v", file.fileName, err)
			continue
		}

		if !bytes.Equal(uploadedContent, file.content) {
			t.Errorf("Uploaded content for %s = %v, want %v", file.fileName, uploadedContent, file.content)
		}
	}
}

// Benchmark for the Write method using fake S3
func BenchmarkS3Client_Write(b *testing.B) {
	// Setup fake S3 server
	backend := s3mem.New()
	createTestBucket(&testing.T{}, backend, "benchmark-bucket")

	faker := gofakes3.New(backend)
	ts := httptest.NewServer(faker.Server())
	defer ts.Close()

	cfg := &S3Config{
		S3BucketName: "benchmark-bucket",
		S3Endpoint:   ts.URL,
		S3Region:     "us-east-1",
		S3Insecure:   true,
	}

	client, err := NewS3(cfg, "benchmark-file.txt")
	if err != nil {
		b.Fatalf("NewS3() error = %v", err)
	}

	testContent := bytes.Repeat([]byte("benchmark data "), 1000) // ~15KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Write(testContent)
		if err != nil {
			b.Errorf("Write() error in benchmark: %v", err)
		}
	}
}

// Test with different AWS regions to ensure compatibility
func TestS3Client_Write_DifferentRegions(t *testing.T) {
	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}

	for _, region := range regions {
		t.Run("region_"+region, func(t *testing.T) {
			backend := s3mem.New()
			createTestBucket(t, backend, "region-test-bucket")

			faker := gofakes3.New(backend)
			ts := httptest.NewServer(faker.Server())
			defer ts.Close()

			cfg := &S3Config{
				S3BucketName: "region-test-bucket",
				S3Endpoint:   ts.URL,
				S3Region:     region,
				S3Insecure:   true,
			}

			client, err := NewS3(cfg, "region-test-file.txt")
			if err != nil {
				t.Fatalf("NewS3() error for region %s = %v", region, err)
			}

			testContent := []byte("Region test content for " + region)
			n, err := client.Write(testContent)

			if err != nil {
				t.Errorf("Write() error for region %s = %v", region, err)
				return
			}

			if n != len(testContent) {
				t.Errorf("Write() for region %s returned %d bytes written, want %d", region, n, len(testContent))
			}
		})
	}
}

// Example usage test
func ExampleNewS3() {
	cfg := &S3Config{
		S3BucketName: "my-bucket",
		S3Endpoint:   "https://s3.amazonaws.com",
		S3Region:     "us-west-2",
		S3Insecure:   false,
	}

	client, err := NewS3(cfg, "example-file.txt")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create S3 client")
	}

	content := []byte("Hello, S3!")
	bytesWritten, err := client.Write(content)
	if err != nil {
		log.Error().Err(err).Msg("Failed to write to S3")
		return
	}

	log.Info().Int("bytes", bytesWritten).Msg("Successfully uploaded to S3")
}

// Test S3Config struct
func TestS3Config(t *testing.T) {
	cfg := &S3Config{
		S3BucketName: "test-bucket",
		S3Endpoint:   "http://localhost:9000",
		S3Region:     "us-east-1",
		S3Insecure:   true,
	}

	// Verify all fields are accessible and correctly set
	if cfg.S3BucketName != "test-bucket" {
		t.Errorf("S3BucketName = %v, want test-bucket", cfg.S3BucketName)
	}
	if cfg.S3Endpoint != "http://localhost:9000" {
		t.Errorf("S3Endpoint = %v, want http://localhost:9000", cfg.S3Endpoint)
	}
	if cfg.S3Region != "us-east-1" {
		t.Errorf("S3Region = %v, want us-east-1", cfg.S3Region)
	}
	if cfg.S3Insecure != true {
		t.Errorf("S3Insecure = %v, want true", cfg.S3Insecure)
	}
}

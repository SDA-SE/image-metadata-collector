package s3

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/logging"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type S3Config struct {
	S3BucketName string
	S3Endpoint   string
	S3Region     string
	S3Insecure   bool
}

type s3Client struct {
	bucket         string
	endpoint       string
	insecure       bool
	region         string
	forcePathStyle bool
	fileName       string
}

// zerologLogger adapts zerolog to Smithy logging.Logger
type zerologLogger struct{}

func (l zerologLogger) Logf(classification logging.Classification, format string, v ...interface{}) {
	// Map AWS SDK log classes to zerolog levels
	switch classification {
	case logging.Debug:
		log.Debug().Msgf(format, v...)
	case logging.Warn:
		log.Warn().Msgf(format, v...)
	default:
		log.Info().Msgf(format, v...)
	}
}

// NewS3 creates a new S3Parameter instance.
func NewS3(cfg *S3Config, fileName string) (*s3Client, error) {
	forcePathStyle := cfg.S3Endpoint != ""

	s3c := &s3Client{
		bucket:         cfg.S3BucketName,
		endpoint:       cfg.S3Endpoint,
		insecure:       cfg.S3Insecure,
		region:         cfg.S3Region,
		forcePathStyle: forcePathStyle,
		fileName:       fileName,
	}

	if s3c.bucket == "" {
		return nil, fmt.Errorf("S3_BUCKET is not set")
	}

	return s3c, nil
}

// Write uploads the content to an S3 Bucket with a key consisting of the fileName.
func (s3c *s3Client) Write(content []byte) (int, error) {
	ctx := context.Background()

	insecureStr := strconv.FormatBool(s3c.insecure)
	log.Info().Str("s3.insecure", insecureStr).Msg("in Upload")

	var cfg aws.Config
	var err error

	// If we have a custom endpoint (like for testing or MinIO), use anonymous credentials
	if s3c.endpoint != "" {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(s3c.region),
			config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     "test-access-key",
					SecretAccessKey: "test-secret-key",
				},
			}),
		)
	} else {
		// For real AWS S3, load default config with credential chain
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(s3c.region),
		)
	}

	if err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to load AWS config: %v", err))
		return 0, err
	}

	// Enable debug logging if zerolog is set to DebugLevel
	if zerolog.GlobalLevel() == zerolog.DebugLevel {
		cfg.Logger = zerologLogger{}
		cfg.ClientLogMode = aws.LogRequest | aws.LogResponseWithBody | aws.LogRetries
	}

	// Build service client with service-specific endpoint override
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = s3c.forcePathStyle
		if s3c.endpoint != "" {
			o.BaseEndpoint = aws.String(s3c.endpoint) // <- replaces deprecated EndpointResolver
		}
	})

	tm := transfermanager.New(client)

	_, err = tm.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket: &s3c.bucket,
		Key:    &s3c.fileName,
		Body:   bytes.NewReader(content),
	})
	if err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to upload to S3 bucket %s, err: %v", s3c.bucket, err))
		return 0, err
	}

	log.Info().Str("fileName", s3c.fileName).Msg("Created new file in s3")
	return len(content), nil
}

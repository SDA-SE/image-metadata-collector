package s3

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	// "github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	// "os"
	// "path"
	"strconv"
)

type S3Config struct {
	S3BucketName string
	S3Endpoint   string
	S3Region     string
	S3Insecure   bool
}

type s3 struct {
	bucket         string
	endpoint       string
	insecure       bool
	region         string
	forcePathStyle bool
	fileName       string
}

// NewS3 creates a new S3Parameter instance.
func NewS3(cfg *S3Config, fileName string) (*s3, error) {

	forcePathStyle := false

	if cfg.S3Endpoint != "" && !forcePathStyle {
		forcePathStyle = true
	}

	s3 := &s3{
		bucket:         cfg.S3BucketName,
		endpoint:       cfg.S3Endpoint,
		insecure:       cfg.S3Insecure,
		region:         cfg.S3Region,
		forcePathStyle: forcePathStyle,
		fileName:       fileName,
	}

	if s3.bucket == "" {
		return nil, fmt.Errorf("S3_BUCKET is not set")
	}

	return s3, nil
}

// Upload uploads the content to an S3 Bucket with a key consisting of the environmentName and the fileName.
func (s3 s3) Write(content []byte) (int, error) {

	insecureStr := strconv.FormatBool(s3.insecure)
	log.Info().Str("s3.insecure", insecureStr).Msg("in Upload")

	sess, err := session.NewSession(&aws.Config{
		DisableSSL:       aws.Bool(s3.insecure),
		S3ForcePathStyle: aws.Bool(s3.forcePathStyle),
		Region:           aws.String(s3.region),
		LogLevel:         getAwsLoglevel(),
		Endpoint:         aws.String(s3.endpoint),
	})

	if err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to create an aws session err: %v", err))
		return len(content), err
	}

	// Setup the S3 Upload Manager. Also see the SDK doc for the Upload Manager
	// for more information on configuring part size, and concurrency.
	// http://docs.aws.amazon.com/sdk-for-go/api/service/s3/s3manager/#NewUploader
	uploader := s3manager.NewUploader(sess)

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s3.bucket),
		Key:    aws.String(s3.fileName),
		Body:   bytes.NewReader(content),
	})

	if err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to upload to S3 bucket %s, err: %v", s3.bucket, err))
		return 0, err
	}

	log.Info().Str("fileName", s3.fileName).Msg("Created new file in s3")

	return len(content), nil
}

func getAwsLoglevel() *aws.LogLevelType {
	logLevel := aws.LogLevel(aws.LogOff)
	if zerolog.GlobalLevel() == zerolog.DebugLevel {
		logLevel = aws.LogLevel(aws.LogDebug | aws.LogDebugWithHTTPBody | aws.LogDebugWithRequestRetries | aws.LogDebugWithRequestErrors | aws.LogDebugWithSigning)
	}
	return logLevel
}

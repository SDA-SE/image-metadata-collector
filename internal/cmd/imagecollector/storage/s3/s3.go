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
	"path"
	"strconv"
)

type s3Parameters struct {
	bucket         string
	endpoint       string
	insecure       bool
	region         string
	forcePathStyle bool
}

// NewS3 creates a new S3Parameter instance.
func NewS3(bucketName, endpoint, region string, insecure bool) (*s3Parameters, error) {

	forcePathStyle := false

	if endpoint != "" && !forcePathStyle {
		forcePathStyle = true
	}

	s3 := &s3Parameters{
		bucket:         bucketName,
		endpoint:       endpoint,
		insecure:       insecure,
		region:         region,
		forcePathStyle: forcePathStyle,
	}

	if s3.bucket == "" {
		return nil, fmt.Errorf("S3_BUCKET is not set")
	}

	return s3, nil
}

// Upload uploads the content to an S3 Bucket with a key consisting of the environmentName and the fileName.
func (s3 s3Parameters) Upload(content []byte, fileName string, environmentName string) error {

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
		return err
	}

	// Setup the S3 Upload Manager. Also see the SDK doc for the Upload Manager
	// for more information on configuring part size, and concurrency.
	// http://docs.aws.amazon.com/sdk-for-go/api/service/s3/s3manager/#NewUploader
	uploader := s3manager.NewUploader(sess)

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s3.bucket),
		Key:    aws.String(environmentName + "/imagecollector/" + path.Base(fileName)),
		Body:   bytes.NewReader(content),
	})

	if err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to upload to S3 bucket %s, err: %v", s3.bucket, err))
		return err
	}

	log.Info().Str("fileName", fileName).Msg("Created new file in s3")
	return nil
}

func getAwsLoglevel() *aws.LogLevelType {
	logLevel := aws.LogLevel(aws.LogOff)
	if zerolog.GlobalLevel() == zerolog.DebugLevel {
		logLevel = aws.LogLevel(aws.LogDebug | aws.LogDebugWithHTTPBody | aws.LogDebugWithRequestRetries | aws.LogDebugWithRequestErrors | aws.LogDebugWithSigning)
	}
	return logLevel
}

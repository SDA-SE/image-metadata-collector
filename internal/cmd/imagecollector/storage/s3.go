package storage

import (
	"bytes"
	"fmt"
	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"path"
	"strconv"
)

var s3ParameterEntry = model.S3parameterEntry{}

func Init(providedS3ParameterEntry model.S3parameterEntry) {
	s3ParameterEntry = providedS3ParameterEntry
	if s3ParameterEntry.Disabled {
		log.Info().Msg("S3 is disabled")
		return
	}
	if s3ParameterEntry.S3bucket == "" {
		log.Info().Msg("S3bucket not given")
		return
	}
	var validate = validator.New()
	validate.RegisterStructValidation(model.ValidateCollectorEntry, model.CollectorEntry{})

	err := validate.Struct(s3ParameterEntry)
	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			log.Fatal().Stack().Err(err).Msg("Could not validate struct")
		}

		for _, err := range err.(validator.ValidationErrors) {
			log.Fatal().Stack().Err(err).Msg("Validation Errors")
		}
	}
}

const serviceAccountTokenLocation = "/var/run/secrets/kubernetes.io/serviceaccount/token"

func Upload(content []byte, fileName string, environmentName string) error {
	if s3ParameterEntry.Disabled {
		log.Info().Msg("S3 is disabled")
		return nil
	}
	if s3ParameterEntry.S3bucket == "" {
		log.Info().Msg("S3bucket not given")
		return nil
	}
	insecureStr := strconv.FormatBool(s3ParameterEntry.S3insecure)
	log.Info().Str("s3ParameterEntry.S3insecure", insecureStr).Msg("in Upload")

	awsConfig := aws.Config{
		DisableSSL:       aws.Bool(s3ParameterEntry.S3insecure),
		S3ForcePathStyle: aws.Bool(s3ParameterEntry.S3ForcePathStyle),
		Region:           aws.String(s3ParameterEntry.S3region),
		LogLevel:         getAwsLoglevel(),
	}
	sess, _ := session.NewSession(&awsConfig)
	awsConfig = getAwsConfigWithCredentials(awsConfig, sess)
	awsConfig = *awsConfig.WithEndpoint(*aws.String(s3ParameterEntry.S3endpoint))
	log.Info().Str("s3ParameterEntry.S3accessKey", s3ParameterEntry.S3accessKey).Msg("in Upload")
	var size int64 = int64(len(content))
	fileType := http.DetectContentType(content)

	var res, uploadError = s3.New(sess, &awsConfig).PutObject(&s3.PutObjectInput{
		Bucket:             aws.String(s3ParameterEntry.S3bucket),
		Key:                aws.String(environmentName + "/imagecollector/" + path.Base(fileName)),
		Body:               bytes.NewReader(content),
		ContentDisposition: aws.String("attachment"),
		ContentLength:      aws.Int64(size),
		ContentType:        aws.String(fileType),
	})
	if uploadError != nil {
		log.Error().Msg(fmt.Sprintf("Failed to upload to S3 bucket %s, err: %v", s3ParameterEntry.S3bucket, uploadError))
		return uploadError
	}
	log.Info().Str("res", res.String()).Str("fileName", fileName).Msg("Created new file in s3")
	return nil
}

func getAwsConfigWithCredentials(awsConfig aws.Config, sess *session.Session) aws.Config {
	var creds *credentials.Credentials
	S3accessKey := s3ParameterEntry.S3accessKey
	S3secretKey := s3ParameterEntry.S3secretKey
	sessionToken := ""

	if S3accessKey == "" {
		stsSTS := sts.New(sess)
		roleARN := os.Getenv("AWS_ROLE_ARN")
		roleProvider := stscreds.NewWebIdentityRoleProviderWithOptions(stsSTS, roleARN, "image-collector", stscreds.FetchTokenPath(os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")))

		creds = credentials.NewCredentials(roleProvider)
		credValue, _ := roleProvider.Retrieve()
		S3accessKey = credValue.AccessKeyID
		S3secretKey = credValue.SecretAccessKey
		sessionToken = credValue.SessionToken
		log.Info().Str("S3accessKey", S3accessKey).Msg("in getAwsConfigWithCredentials")
	}
	creds = credentials.NewStaticCredentials(S3accessKey, S3secretKey, sessionToken)
	awsConfig = *awsConfig.WithCredentials(creds).WithEndpoint(*aws.String(s3ParameterEntry.S3endpoint))
	return awsConfig
}

func getAwsLoglevel() *aws.LogLevelType {
	logLevel := aws.LogLevel(aws.LogOff)
	if zerolog.GlobalLevel() == zerolog.DebugLevel {
		logLevel = aws.LogLevel(aws.LogDebug | aws.LogDebugWithHTTPBody | aws.LogDebugWithRequestRetries | aws.LogDebugWithRequestErrors | aws.LogDebugWithSigning)
	}
	return logLevel
}

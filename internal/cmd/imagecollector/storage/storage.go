package storage

import (
	"fmt"

	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/storage/fs"
	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/storage/git"
	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/storage/s3"
)

type Storager interface {
	Upload(content []byte, fileName, environmentName string) error
}

type StorageConfig struct {
	StorageFlag          string
	S3bucketName         string
	S3endpoint           string
	S3region             string
	S3insecure           bool
	FsBaseDir            string
	GitUrl               string
	GitDirectory         string
	GitPrivateKeyFile    string
	GitPassword          string
	GithubAppId          int64
	GithubInstallationId int64
}

func NewStorage(cfg *StorageConfig) (Storager, error) {

	var s Storager
	var err error

	switch cfg.StorageFlag {
	case "s3":
		s, err = s3.NewS3(cfg.S3bucketName, cfg.S3endpoint, cfg.S3region, cfg.S3insecure)
	case "git":
		s, err = git.NewGit(cfg.GitUrl, cfg.GitDirectory, cfg.GitPrivateKeyFile, cfg.GitPassword, cfg.GithubAppId, cfg.GithubInstallationId)
	case "fs":
		s, err = fs.NewFs(cfg.FsBaseDir)
	default:
		s = nil
		err = fmt.Errorf("Storage flag %s is not supported", cfg.StorageFlag)
	}

	return s, err
}

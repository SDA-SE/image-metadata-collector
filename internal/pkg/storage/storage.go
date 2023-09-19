package storage

import (
	"fmt"

	"github.com/SDA-SE/image-metadata-collector/internal/pkg/storage/fs"
	"github.com/SDA-SE/image-metadata-collector/internal/pkg/storage/git"
	"github.com/SDA-SE/image-metadata-collector/internal/pkg/storage/s3"
)

// Storager is implemented by different storage options (e.g., S3, Git, Local FS)
// It defines everything needed to store the collector output to the chosen
// storage option.
type Storager interface {
	// Upload takes the content of the collector output, the filename, and the environment name
	// of the scanned cluster and writes the content to the chosen storage option
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

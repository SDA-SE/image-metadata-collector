package storage

import (
	"fmt"
	"io"
	"os"

	"github.com/SDA-SE/image-metadata-collector/internal/pkg/storage/api"
	"github.com/SDA-SE/image-metadata-collector/internal/pkg/storage/git"
	"github.com/SDA-SE/image-metadata-collector/internal/pkg/storage/s3"
)

type StorageConfig struct {
	s3.S3Config
	git.GitConfig
	api.ApiConfig

	StorageFlag string
	FileName    string
}

func NewStorage(cfg *StorageConfig, environment string) (io.Writer, error) {

	var w io.Writer
	var err error

	filename := cfg.FileName

	if filename == "" {
		filename = environment + "-output.json"
	}

	switch cfg.StorageFlag {
	case "s3":
		w, err = s3.NewS3(&cfg.S3Config, filename)
	case "api":
		//w = cfg.ApiConfig
		w, err = api.NewApi(&cfg.ApiConfig)
	case "git":
		w, err = git.NewGit(&cfg.GitConfig, filename)
	case "fs":
		var file *os.File
		file, err = os.Create(filename)
		w = file
	case "stdout":
		w = os.Stdout
	default:
		w = nil
		err = fmt.Errorf("Storage flag %s is not supported", cfg.StorageFlag)
	}

	return w, err
}

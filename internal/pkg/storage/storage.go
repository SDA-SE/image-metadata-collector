package storage

import (
	"fmt"
	"io"
	"os"

	"github.com/SDA-SE/image-metadata-collector/internal/pkg/storage/git"
	"github.com/SDA-SE/image-metadata-collector/internal/pkg/storage/s3"
)

type StorageConfig struct {
	s3.S3Config
	git.GitConfig

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
	case "git":
		w, err = git.NewGit(&cfg.GitConfig, filename)
	case "fs":
		file, err_ := os.Create(filename)
		defer file.Close()
		err = err_
		w = file
	case "stdout":
		w = os.Stdout
	default:
		w = nil
		err = fmt.Errorf("Storage flag %s is not supported", cfg.StorageFlag)
	}

	return w, err
}

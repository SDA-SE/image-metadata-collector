package config

import (
	"github.com/SDA-SE/image-metadata-collector/internal/collector"
	"github.com/SDA-SE/image-metadata-collector/internal/pkg/kubeclient"
	"github.com/SDA-SE/image-metadata-collector/internal/pkg/storage"
)

type Config struct {
	collector.AnnotationNames
	collector.CollectorImage
	kubeclient.KubeConfig
	storage.StorageConfig
	collector.RunConfig

	Debug bool
}

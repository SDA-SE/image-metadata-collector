package config

import (
	"time"

	"github.com/SDA-SE/image-metadata-collector/internal/collector"
	"github.com/SDA-SE/image-metadata-collector/internal/pkg/storage"
	"github.com/SDA-SE/image-metadata-collector/internal/pkg/kubeclient"
)

type Config struct {
	collector.AnnotationNames
	collector.CollectorImage
	kubeclient.KubeConfig
	storage.StorageConfig

	ScanInterval  time.Duration
	Debug         bool
	ExposeMetrics bool
}

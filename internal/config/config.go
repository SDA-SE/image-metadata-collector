package config

import (
	"time"

	"github.com/SDA-SE/sdase-image-collector/internal/collector"
	"github.com/SDA-SE/sdase-image-collector/internal/pkg/storage"
	"github.com/SDA-SE/sdase-image-collector/internal/pkg/kubeclient"
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

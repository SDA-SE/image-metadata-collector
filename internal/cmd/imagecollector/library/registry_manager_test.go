package library

import (
	"strings"
	"testing"

	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/model"
)

func TestPrepareImageNameForPullableRemoval(t *testing.T) {
	image := "docker-pullable://abc.def/fgh/ji:123"
	image = PrepareImageName(image)
	if strings.Contains(image, "docker-pullable://") {
		t.Fatal("docker-pullable is still present in image name", image)
	}
}
func TestPrepareImageNameRegistryReplacement(t *testing.T) {
	registryReplacements = []model.RegistyReplacment{{Replacement: "new-registry.com", Original: "original-registry.com"}}
	image := "docker-pullable://original-registry.com/fgh/ji:123"
	image = PrepareImageName(image)
	if !strings.EqualFold(image, "new-registry.com/fgh/ji:123") {
		t.Fatal("RegistryReplacement didn't work ", image)
	}
}

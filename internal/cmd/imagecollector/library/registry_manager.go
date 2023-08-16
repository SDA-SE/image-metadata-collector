package library

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/model"
	"github.com/rs/zerolog/log"
)

var registryReplacements []model.RegistyReplacment

func InitRegistryRename(configBasePath string) error {
	var registryRenameFile = configBasePath + "/registry-rename.json"
	if _, err := os.Stat(registryRenameFile); err == nil {
		log.Info().Str("registryRenameFile", registryRenameFile).Msg("Found registryRenameFile")
		file, _ := os.ReadFile(registryRenameFile)

		if err := json.Unmarshal([]byte(file), &registryReplacements); err != nil {
			log.Fatal().Stack().Err(err).Str("registryRenameFile", registryRenameFile).Msg("Couldn't transform string")
			return err
		}
	} else {
		log.Info().Str("registryRenameFile", registryRenameFile).Msg("registryRenameFile doesn't exists")
	}
	return nil
}

func PrepareImageName(image string) string {
	image = strings.Replace(image, "docker-pullable://", "", -1)
	for i := 0; i < len(registryReplacements); i++ {
		image = strings.Replace(image, registryReplacements[i].Original, registryReplacements[i].Replacement, -1)
	}
	return image
}

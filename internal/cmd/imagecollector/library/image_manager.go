package library

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/model"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
)

type ImageManager struct {
	imageNegativeList []string
}

func InitImageNegativeList(configBasePath string) (ImageManager, error) {
	var imageManager = ImageManager{}
	var imageNegativeListFile = configBasePath + "/imageNegativeList.json"
	if _, err := os.Stat(imageNegativeListFile); err == nil {
		log.Info().Str("imageNegativeListFile", imageNegativeListFile).Msg("Found imageNegativeListFile")
		file, _ := os.ReadFile(imageNegativeListFile)

		if err := json.Unmarshal([]byte(file), &imageManager.imageNegativeList); err != nil {
			log.Fatal().Stack().Err(err).Msg("Couldn't Unmarshal")
			return imageManager, err
		}
	} else {
		log.Info().Str("imageNegativeListFile", imageNegativeListFile).Msg("imageNegativeListFile doesn't exists")
	}
	return imageManager, nil
}

func CheckAndSetSkipByImageNegativeList(imageManager ImageManager, container v1.ContainerStatus, collectorEntry *model.CollectorEntry) {
	for _, imageToSkip := range imageManager.imageNegativeList {
		if strings.Index(container.Image, imageToSkip) == 0 {
			log.Info().Str("imageToSkip", imageToSkip).Msg("Skipping image due to imageNegativeList")
			collectorEntry.Skip = true
		}
	}
}

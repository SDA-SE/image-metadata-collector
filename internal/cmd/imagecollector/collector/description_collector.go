package collector

import (
	"context"
	"encoding/json"
	"os"
	"sort"

	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/library"

	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/model"
	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/storage"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var imageCollectorDefaults model.ImageCollectorDefaults

func ClusterImageScannerDescriptionCollectorRun(givenImageCollectorDefaults model.ImageCollectorDefaults) error {
	imageCollectorDefaults = givenImageCollectorDefaults
	descriptionEntries := []model.DescriptionEntry{}
	namespaces, err := imageCollectorDefaults.Client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("failed to get namespaces")
		return err
	}

	for _, namespace := range namespaces.Items {
		log.Info().Str("namespace", namespace.GetName()).Msg("In namespace")
		descriptionEntries = append(descriptionEntries, getDescriptionEntry(&namespace))
	}
	storeDescriptionFiles(descriptionEntries)
	return nil
}

func getDescriptionEntry(namespace *v1.Namespace) model.DescriptionEntry {
	var descriptionEntry = model.DescriptionEntry{
		Team:        os.Getenv("DEFAULT_TEAM_NAME"),
		Environment: imageCollectorDefaults.Environment,
		Namespace:   namespace.GetName(),
		Description: namespace.GetAnnotations()["sdase.org/description"],
	}
	library.SetStringFromAnnotationAndLabel(namespace, "contact.sdase.org/team", &descriptionEntry.Team)

	if descriptionEntry.Description == "" {
		library.SetStringFromAnnotationAndLabel(namespace, "sdase.org/description", &descriptionEntry.Description)
	}

	return descriptionEntry
}

func storeDescriptionFiles(descriptionEntries []model.DescriptionEntry) {
	saveFilesPath := "/tmp"
	sort.Slice(descriptionEntries, func(i, j int) bool {
		return descriptionEntries[i].Namespace < descriptionEntries[j].Namespace
	})
	dataDescriptionEntries, err := json.Marshal(descriptionEntries)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Could not marshal json descriptionEntries")
	}

	if imageCollectorDefaults.IsSaveFiles {
		library.SaveFile(saveFilesPath+"/service-description.json", []byte(dataDescriptionEntries))
	}
	storage.Upload([]byte(dataDescriptionEntries), saveFilesPath+"/"+imageCollectorDefaults.Environment+"-service-description.json", imageCollectorDefaults.Environment)
	storage.GitUpload([]byte(dataDescriptionEntries), imageCollectorDefaults.Environment+"service-description.json")
	sort.Slice(descriptionEntries, func(i, j int) bool {
		return descriptionEntries[i].Namespace < descriptionEntries[j].Namespace
	})
	missingDescriptionText := "Missing description on namespace in environment " + os.Getenv("ENVIRONMENT_NAME")
	for _, entry := range descriptionEntries {
		if entry.Description == "" {
			missingDescriptionText += entry.Namespace + "\n"
		}
	}
	if imageCollectorDefaults.IsSaveFiles {
		library.SaveFile(saveFilesPath+"/"+imageCollectorDefaults.Environment+"/missing-service-description.txt", []byte(missingDescriptionText))
	}
	storage.Upload([]byte(missingDescriptionText), saveFilesPath+"/"+imageCollectorDefaults.Environment+"-missing-service-description.txt", imageCollectorDefaults.Environment)
	storage.GitUpload([]byte(missingDescriptionText), imageCollectorDefaults.Environment+"-missing-service-description.txt")
}

package collector

import (
	// "sort"
	// "strings"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/SDA-SE/image-metadata-collector/internal/pkg/kubeclient"
	"github.com/stretchr/testify/assert"
)

func TestIsSkip(t *testing.T) {
	testCases := []struct {
		name           string
		targetImage    CollectorImage
		expectedResult bool
	}{
		{
			name: "NoSkipConditionSet",
			targetImage: CollectorImage{
				Namespace:              "name",
				Skip:                   false,
				NamespaceFilter:        "",
				NamespaceFilterNegated: "",
			},
			expectedResult: false,
		},
		{
			name: "SkipIsSetExpectSkip",
			targetImage: CollectorImage{
				Namespace:              "name",
				Skip:                   true,
				NamespaceFilter:        "",
				NamespaceFilterNegated: "",
			},
			expectedResult: true,
		},
		{
			name: "CatchAllNamespaceFilterExpectSkip",
			targetImage: CollectorImage{
				Namespace:              "name",
				Skip:                   false,
				NamespaceFilter:        ".*",
				NamespaceFilterNegated: "",
			},
			expectedResult: true,
		},
		{
			name: "CatchAllNamespaceFilterAndSkipSetExpectSkip",
			targetImage: CollectorImage{
				Namespace:              "name",
				Skip:                   true,
				NamespaceFilter:        ".*",
				NamespaceFilterNegated: "",
			},
			expectedResult: true,
		},
		{
			name: "NoMatchingNamespaceFilterSetExpectNoSkip",
			targetImage: CollectorImage{
				Namespace:              "name",
				Skip:                   false,
				NamespaceFilter:        "^other$",
				NamespaceFilterNegated: "",
			},
			expectedResult: false,
		},
		{
			name: "NoMatchingNegatedNamespaceFilterSetExpectSkip",
			targetImage: CollectorImage{
				Namespace:              "name",
				Skip:                   false,
				NamespaceFilter:        "",
				NamespaceFilterNegated: "^other$",
			},
			expectedResult: false,
		},
		{
			name: "MultipleMatchingFilterSetExpectSkip",
			targetImage: CollectorImage{
				Namespace:              "name",
				Skip:                   false,
				NamespaceFilter:        "^name$",
				NamespaceFilterNegated: "^other$",
			},
			expectedResult: true,
		},
		{
			name: "AllFilterSetExpectSkip",
			targetImage: CollectorImage{
				Namespace:              "name",
				Skip:                   true,
				NamespaceFilter:        "^name$",
				NamespaceFilterNegated: "^other$",
			},
			expectedResult: true,
		},
		{
			name: "PRExpectSkip",
			targetImage: CollectorImage{
				Namespace:              "test-pr-xx",
				Skip:                   false,
				NamespaceFilterNegated: "\\-pr\\-",
			},
			expectedResult: true,
		},
		{
			name: "PRExpectSkipX",
			targetImage: CollectorImage{
				Namespace:              "test-pr-xx",
				Skip:                   false,
				NamespaceFilterNegated: "-pr-",
			},
			expectedResult: true,
		},
		{
			name: "PRExpectNoSkip",
			targetImage: CollectorImage{
				Namespace:              "test-pr-xx",
				Skip:                   false,
				NamespaceFilterNegated: "1234567890",
			},
			expectedResult: false,
		},
	}

	runConfig := RunConfig{
		ImageFilter: []string{},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isSkipImageByNamespace(&tc.targetImage)

			assert.Equal(t, result, tc.expectedResult, "Expected %v, got %v, with Namespace=%s, Skip=%v, NamespaceFilter=%v, NamespaceFilterNegated=%v, imageFilter=\"%v\"",
				tc.expectedResult,
				result,
				tc.targetImage.Namespace,
				tc.targetImage.Skip,
				tc.targetImage.NamespaceFilter,
				tc.targetImage.NamespaceFilterNegated,
				runConfig.ImageFilter)
		})
	}
}

func TestIsSkipByImageFilter(t *testing.T) {
	testCases := []struct {
		name           string
		targetImage    CollectorImage
		imageFilter    []string
		expectedResult bool
	}{
		{
			name: "NoSkipConditionSet",
			targetImage: CollectorImage{
				Namespace: "name",
				Skip:      false,
			},
			imageFilter:    []string{},
			expectedResult: false,
		},
		{
			name:        "SkipIsSetExpectSkip",
			imageFilter: []string{".*amazonaws.com/.*"},
			targetImage: CollectorImage{
				Image:     "333.dkr.ecr.eu-central-1.amazonaws.com/eks/kube-proxy@sha256:5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03",
				Namespace: "name",
				Skip:      false,
			},
			expectedResult: true,
		},
		{
			name:        "SkipIsSetExpectSkipWithoutSpecialRegex",
			imageFilter: []string{"amazonaws.com/"},
			targetImage: CollectorImage{
				Image: "333.dkr.ecr.eu-central-1.amazonaws.com/eks/kube-proxy@sha256:5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03",
				Skip:  false,
			},
			expectedResult: true,
		},
		{
			name:        "SkipIsSetExpectSkipWithList",
			imageFilter: []string{"amazonaws.com", "aws.com"},
			targetImage: CollectorImage{
				Image:     "333.dkr.ecr.eu-central-1.amazonaws.com/eks/kube-proxy@sha256:5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03",
				Namespace: "name",
				Skip:      false,
			},
			expectedResult: true,
		},
		{
			name:        "NoMatchingNamespaceFilterSetExpectNoSkip",
			imageFilter: []string{"^other$"},
			targetImage: CollectorImage{
				Namespace: "name",
				Skip:      true,
			},
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runConfig := RunConfig{
				ImageFilter: tc.imageFilter,
			}
			result := isSkipImageByImageFilter(&tc.targetImage, &runConfig)

			assert.Equal(t, result, tc.expectedResult, "Expected %v, got %v, with Namespace=%s, Skip=%v, NamespaceFilter=%v, NamespaceFilterNegated=%v, imageFilter=\"%v\"",
				tc.expectedResult,
				result,
				tc.targetImage.Namespace,
				tc.targetImage.Skip,
				tc.targetImage.NamespaceFilter,
				tc.targetImage.NamespaceFilterNegated,
				tc.imageFilter)
		})
	}
}

func TestCleanCollectorImageSkipSet(t *testing.T) {
	testCases := []struct {
		name            string
		targetImage     CollectorImage
		expectedChanged bool
		expectedResult  bool
	}{
		{
			name: "NoSkipConditionSet",
			targetImage: CollectorImage{
				Namespace:              "name",
				Skip:                   false,
				NamespaceFilter:        "",
				NamespaceFilterNegated: "",
			},
			expectedChanged: false,
			expectedResult:  false,
		},
		{
			name: "SkipIsSetExpectSkip",
			targetImage: CollectorImage{
				Namespace:              "name",
				Skip:                   true,
				NamespaceFilter:        "",
				NamespaceFilterNegated: "",
			},
			expectedChanged: false,
			expectedResult:  true,
		},
		{
			name: "CatchAllNamespaceFilterExpectSkip",
			targetImage: CollectorImage{
				Namespace:              "name",
				Skip:                   false,
				NamespaceFilter:        ".*",
				NamespaceFilterNegated: "",
			},
			expectedChanged: true,
			expectedResult:  true,
		},
		{
			name: "CatchAllNamespaceFilterAndSkipSetExpectSkip",
			targetImage: CollectorImage{
				Namespace:              "name",
				Skip:                   true,
				NamespaceFilter:        ".*",
				NamespaceFilterNegated: "",
			},
			expectedChanged: false,
			expectedResult:  true,
		},
		{
			name: "NoMatchingNamespaceFilterSetExpectNoSkip",
			targetImage: CollectorImage{
				Namespace:              "name",
				Skip:                   false,
				NamespaceFilter:        "^other$",
				NamespaceFilterNegated: "",
			},
			expectedChanged: false,
			expectedResult:  false,
		},
		{
			name: "NoMatchingNegatedNamespaceFilterSetExpectSkip",
			targetImage: CollectorImage{
				Namespace:              "name",
				Skip:                   false,
				NamespaceFilter:        "",
				NamespaceFilterNegated: "^other$",
			},
			expectedChanged: false,
			expectedResult:  false,
		},
		{
			name: "MultipleMatchingFilterSetExpectSkip",
			targetImage: CollectorImage{
				Namespace:              "name",
				Skip:                   false,
				NamespaceFilter:        "^name$",
				NamespaceFilterNegated: "^other$",
			},
			expectedChanged: true,
			expectedResult:  true,
		},
		{
			name: "AllFilterSetExpectSkip",
			targetImage: CollectorImage{
				Namespace:              "name",
				Skip:                   true,
				NamespaceFilter:        "^name$",
				NamespaceFilterNegated: "^other$",
			},
			expectedChanged: false,
			expectedResult:  true,
		},
	}
	runConfig := RunConfig{
		ImageFilter: []string{},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initialSkip := tc.targetImage.Skip
			cleanCollectorImage(&tc.targetImage, &runConfig)

			if tc.expectedChanged {
				assert.NotEqual(t, tc.targetImage.Skip, initialSkip, "Expected Skip to change but it did not change")
			} else {
				assert.Equal(t, tc.targetImage.Skip, initialSkip, "Expected Skip not to change but it did change")
			}

			assert.Equal(t,
				tc.targetImage.Skip,
				tc.expectedResult,
				"Expected %v, got %v, with Namespace=%s, Skip=%v, NamespaceFilter=%v, NamespaceFilterNegated=%v",
				tc.expectedResult,
				tc.targetImage.Skip,
				tc.targetImage.Namespace,
				tc.targetImage.Skip,
				tc.targetImage.NamespaceFilter,
				tc.targetImage.NamespaceFilterNegated)
		})
	}
}

func TestCleanCollectorImageImageNameAndID(t *testing.T) {
	testCases := []struct {
		name                 string
		targetImage          CollectorImage
		expectedImgChanged   bool
		expectedImgIdChanged bool
		expectedImage        string
		expectedImageId      string
	}{
		{
			name: "NothingToChangeResultsInNoChange",
			targetImage: CollectorImage{
				Image:   "quay.io/name:tag",
				ImageId: "quay.io/name@sha256:1234567890",
			},
			expectedImage:        "quay.io/name:tag",
			expectedImageId:      "quay.io/name@sha256:1234567890",
			expectedImgChanged:   false,
			expectedImgIdChanged: false,
		},
		{
			name: "RemoveDockerPullableInfoFromID",
			targetImage: CollectorImage{
				Image:   "quay.io/name:tag",
				ImageId: "docker-pullable://quay.io/name@sha256:1234567890",
			},
			expectedImage:        "quay.io/name:tag",
			expectedImageId:      "quay.io/name@sha256:1234567890",
			expectedImgChanged:   false,
			expectedImgIdChanged: true,
		},
		{
			name: "RemoveDockerPullableInfoFromImage",
			targetImage: CollectorImage{
				Image:   "docker-pullable://quay.io/name:tag",
				ImageId: "quay.io/name@sha256:1234567890",
			},
			expectedImage:        "quay.io/name:tag",
			expectedImageId:      "quay.io/name@sha256:1234567890",
			expectedImgChanged:   true,
			expectedImgIdChanged: false,
		},
		{
			name: "RemoveDockerPullableInfoFromImageAndID",
			targetImage: CollectorImage{
				Image:   "docker-pullable://quay.io/name:tag",
				ImageId: "docker-pullable://quay.io/name@sha256:1234567890",
			},
			expectedImage:        "quay.io/name:tag",
			expectedImageId:      "quay.io/name@sha256:1234567890",
			expectedImgChanged:   true,
			expectedImgIdChanged: true,
		},
		{
			name: "DontRemoveDockerPullableFromTag",
			targetImage: CollectorImage{
				Image:   "quay.io/name:docker-pullable",
				ImageId: "quay.io/name@sha256:1234567890",
			},
			expectedImage:        "quay.io/name:docker-pullable",
			expectedImageId:      "quay.io/name@sha256:1234567890",
			expectedImgChanged:   false,
			expectedImgIdChanged: false,
		},
	}
	runConfig := RunConfig{
		ImageFilter: []string{},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initialImage := tc.targetImage.Image
			initialImageId := tc.targetImage.ImageId

			cleanCollectorImage(&tc.targetImage, &runConfig)

			if tc.expectedImgChanged {
				assert.NotEqual(t, tc.targetImage.Image, initialImage, "Expected Image to change but it did not change")
			} else {
				assert.Equal(t, tc.targetImage.Image, initialImage, "Expected Image not to change but it did change")
			}

			if tc.expectedImgIdChanged {
				assert.NotEqual(t, tc.targetImage.ImageId, initialImageId, "Expected ImageId to change but it did not change")
			} else {
				assert.Equal(t, tc.targetImage.ImageId, initialImageId, "Expected ImageId not to change but it did change")
			}

			assert.Equal(t, tc.targetImage.Image, tc.expectedImage, "Expected %v, got %v,", tc.expectedImage, tc.targetImage.Image)
			assert.Equal(t, tc.targetImage.ImageId, tc.expectedImageId, "Expected %v, got %v,", tc.expectedImageId, tc.targetImage.ImageId)
		})
	}
}

func TestConvert(t *testing.T) {
	defaults := CollectorImage{
		Environment: "myEnv",
		// Destination: "Lorem Ipsum Dolor Sit Amet",
		ContainerType:  "myContainerType",
		Team:           "myTeam",
		Owners:         []Owner{},
		EngagementTags: []string{"defaultTag"},

		IsScanBaseimageLifetime: true,
		IsScanDependencyCheck:   true,
		IsScanDependencyTrack:   true,
		IsScanDistroless:        true,
		IsScanLifetime:          true,
		IsScanMalware:           true,
	}

	annotationNames := AnnotationNames{
		Base:       "sda.se/",
		Scans:      "scans.sda.se/",
		Contact:    "contact.sda.se/",
		DefectDojo: "dd.sda.se/",
	}

	testCases := []struct {
		name                   string
		defaults               *CollectorImage
		annotationNames        *AnnotationNames
		runConfig              *RunConfig
		targetK8Image          *[]kubeclient.Image
		expectedCollectorImage *[]CollectorImage
	}{
		{
			name:                   "EmptyInputsResultsInEmptyOutput",
			defaults:               &CollectorImage{},
			annotationNames:        &AnnotationNames{},
			targetK8Image:          &[]kubeclient.Image{{}},
			expectedCollectorImage: &[]CollectorImage{{}},
		},
		{
			name:            "EmptyInputResultsInEmptyOutput",
			defaults:        &CollectorImage{},
			annotationNames: &AnnotationNames{},
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				NamespaceName: "myNamespace",
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:tag",
			}},
		},
		{
			name:            "ImageTypeIsMapped",
			defaults:        &CollectorImage{},
			annotationNames: &AnnotationNames{},
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				NamespaceName: "myNamespace",
				ImageType:     kubeclient.ImageTypeCronJob,
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:tag",
				ImageType: kubeclient.ImageTypeCronJob,
			}},
		},
		{
			name:                   "EmptyInputWithDefaultsResultsInDefaults",
			defaults:               &defaults,
			annotationNames:        &annotationNames,
			targetK8Image:          &[]kubeclient.Image{{}},
			expectedCollectorImage: &[]CollectorImage{defaults},
		},
		{
			name:            "MergeK8InfoWithDefaults",
			defaults:        &defaults,
			annotationNames: &annotationNames,
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				NamespaceName: "myNamespace",
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:tag",

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           defaults.Team,
				Owners:         defaults.Owners,
				EngagementTags: defaults.EngagementTags,

				IsScanBaseimageLifetime: defaults.IsScanBaseimageLifetime,
				IsScanDependencyCheck:   defaults.IsScanDependencyCheck,
				IsScanDependencyTrack:   defaults.IsScanDependencyTrack,
				IsScanDistroless:        defaults.IsScanDistroless,
				IsScanLifetime:          defaults.IsScanLifetime,
				IsScanMalware:           defaults.IsScanMalware,
			}},
		},
		{
			name:            "MergeK8InfoWithDefaultsForMultipleImages",
			defaults:        &defaults,
			annotationNames: &annotationNames,
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag1",
				NamespaceName: "myNamespace",
			}, {
				Image:         "quay.io/name:tag2",
				NamespaceName: "myNamespace",
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag1",
				ImageId:   "quay.io/name:tag1",

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           defaults.Team,
				Owners:         defaults.Owners,
				EngagementTags: defaults.EngagementTags,

				IsScanBaseimageLifetime: defaults.IsScanBaseimageLifetime,
				IsScanDependencyCheck:   defaults.IsScanDependencyCheck,
				IsScanDependencyTrack:   defaults.IsScanDependencyTrack,
				IsScanDistroless:        defaults.IsScanDistroless,
				IsScanLifetime:          defaults.IsScanLifetime,
				IsScanMalware:           defaults.IsScanMalware,
			}, {
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag2",
				ImageId:   "quay.io/name:tag2",

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           defaults.Team,
				Owners:         defaults.Owners,
				EngagementTags: defaults.EngagementTags,

				IsScanBaseimageLifetime: defaults.IsScanBaseimageLifetime,
				IsScanDependencyCheck:   defaults.IsScanDependencyCheck,
				IsScanDependencyTrack:   defaults.IsScanDependencyTrack,
				IsScanDistroless:        defaults.IsScanDistroless,
				IsScanLifetime:          defaults.IsScanLifetime,
				IsScanMalware:           defaults.IsScanMalware,
			}},
		},
		{
			name:            "ParseLabels",
			defaults:        &defaults,
			annotationNames: &annotationNames,
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				NamespaceName: "myNamespace",
				Labels:        map[string]string{"contact.sda.se/team": "some-none-default-team"},
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:tag",

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           "some-none-default-team",
				Owners:         defaults.Owners,
				EngagementTags: defaults.EngagementTags,

				IsScanBaseimageLifetime: defaults.IsScanBaseimageLifetime,
				IsScanDependencyCheck:   defaults.IsScanDependencyCheck,
				IsScanDependencyTrack:   defaults.IsScanDependencyTrack,
				IsScanDistroless:        defaults.IsScanDistroless,
				IsScanLifetime:          defaults.IsScanLifetime,
				IsScanMalware:           defaults.IsScanMalware,
			}},
		},
		{
			name:            "ParseAnnotations",
			defaults:        &defaults,
			annotationNames: &annotationNames,
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				NamespaceName: "myNamespace",
				Annotations:   map[string]string{"contact.sda.se/team": "some-none-default-team"},
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:tag",

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           "some-none-default-team",
				Owners:         defaults.Owners,
				EngagementTags: defaults.EngagementTags,

				IsScanBaseimageLifetime: defaults.IsScanBaseimageLifetime,
				IsScanDependencyCheck:   defaults.IsScanDependencyCheck,
				IsScanDependencyTrack:   defaults.IsScanDependencyTrack,
				IsScanDistroless:        defaults.IsScanDistroless,
				IsScanLifetime:          defaults.IsScanLifetime,
				IsScanMalware:           defaults.IsScanMalware,
			}},
		},
		{
			name:            "ParseAnnotationsAndLabelsWithAnnotationsTakingPrecedence",
			defaults:        &defaults,
			annotationNames: &annotationNames,
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				NamespaceName: "myNamespace",

				Labels:      map[string]string{"contact.sda.se/team": "team-from-label"},
				Annotations: map[string]string{"contact.sda.se/team": "team-from-annotations"},
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:tag",

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           "team-from-annotations",
				Owners:         defaults.Owners,
				EngagementTags: defaults.EngagementTags,

				IsScanBaseimageLifetime: defaults.IsScanBaseimageLifetime,
				IsScanDependencyCheck:   defaults.IsScanDependencyCheck,
				IsScanDependencyTrack:   defaults.IsScanDependencyTrack,
				IsScanDistroless:        defaults.IsScanDistroless,
				IsScanLifetime:          defaults.IsScanLifetime,
				IsScanMalware:           defaults.IsScanMalware,
			}},
		},
		{
			name:            "ParseMultipleAnnotationsAndLabels",
			defaults:        &defaults,
			annotationNames: &annotationNames,
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				ImageId:       "quay.io/name@sha256:1234",
				NamespaceName: "myNamespace",
				Annotations:   map[string]string{"scans.sda.se/is-scan-malware": "false", "scans.sda.se/is-scan-distroless": "false"},
				Labels:        map[string]string{"contact.sda.se/team": "some-none-default-team"},
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name@sha256:1234",

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           "some-none-default-team",
				Owners:         defaults.Owners,
				EngagementTags: defaults.EngagementTags,

				IsScanBaseimageLifetime: defaults.IsScanBaseimageLifetime,
				IsScanDependencyCheck:   defaults.IsScanDependencyCheck,
				IsScanDependencyTrack:   defaults.IsScanDependencyTrack,
				IsScanDistroless:        false,
				IsScanLifetime:          defaults.IsScanLifetime,
				IsScanMalware:           false,
			}},
		},
		{
			name:            "ParseEngagementTags",
			defaults:        &defaults,
			annotationNames: &annotationNames,
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				ImageId:       "quay.io/name@sha256:1234",
				NamespaceName: "myNamespace",
				Annotations:   map[string]string{"dd.sda.se/engagement-tags": "first,second,third"},
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name@sha256:1234",

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           defaults.Team,
				Owners:         defaults.Owners,
				EngagementTags: []string{"first", "second", "third"},

				IsScanBaseimageLifetime: defaults.IsScanBaseimageLifetime,
				IsScanDependencyCheck:   defaults.IsScanDependencyCheck,
				IsScanDependencyTrack:   defaults.IsScanDependencyTrack,
				IsScanDistroless:        defaults.IsScanDistroless,
				IsScanLifetime:          defaults.IsScanLifetime,
				IsScanMalware:           defaults.IsScanMalware,
			}},
		},
		{
			name:            "WrongAnnotationNameHasNoEffect",
			defaults:        &defaults,
			annotationNames: &annotationNames,
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				NamespaceName: "myNamespace",
				Annotations:   map[string]string{"wrong-name.sda.se/team": "team-from-annotations"},
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:tag",

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           defaults.Team,
				Owners:         defaults.Owners,
				EngagementTags: defaults.EngagementTags,

				IsScanBaseimageLifetime: defaults.IsScanBaseimageLifetime,
				IsScanDependencyCheck:   defaults.IsScanDependencyCheck,
				IsScanDependencyTrack:   defaults.IsScanDependencyTrack,
				IsScanDistroless:        defaults.IsScanDistroless,
				IsScanLifetime:          defaults.IsScanLifetime,
				IsScanMalware:           defaults.IsScanMalware,
			}},
		},
		{
			name:            "DescriptionAnnotationIsParsed",
			defaults:        &defaults,
			annotationNames: &annotationNames,
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				ImageId:       "quay.io/name:sha",
				NamespaceName: "myNamespace",
				Annotations:   map[string]string{"sda.se/description": "Lorem Ipsum Dolor Sit Amet"},
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:sha",

				Environment:    defaults.Environment,
				Description:    "Lorem Ipsum Dolor Sit Amet",
				ContainerType:  defaults.ContainerType,
				Team:           defaults.Team,
				Owners:         []Owner{},
				EngagementTags: defaults.EngagementTags,

				IsScanBaseimageLifetime: defaults.IsScanBaseimageLifetime,
				IsScanDependencyCheck:   defaults.IsScanDependencyCheck,
				IsScanDependencyTrack:   defaults.IsScanDependencyTrack,
				IsScanDistroless:        defaults.IsScanDistroless,
				IsScanLifetime:          defaults.IsScanLifetime,
				IsScanMalware:           defaults.IsScanMalware,
			}},
		},
		{
			name:            "MergedNotificationAnnotation",
			defaults:        &CollectorImage{},
			annotationNames: &annotationNames,
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				NamespaceName: "myNamespace",
				ImageType:     kubeclient.ImageTypeCronJob,
				Annotations: map[string]string{"contact.sda.se/notifications": "{" +
					"\"slack\":[\"channel1\",\"channel2\"]," +
					"\"emails\":[\"admin@company.de\",\"super-admin+devops@company.de\"]," +
					"\"ms_teams\":[\"1234689745631@teams.microsoft.ms\"]" +
					"}"},
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:tag",
				ImageType: kubeclient.ImageTypeCronJob,
				Notifications: Notifications{
					Slack:   []string{"channel1", "channel2"},
					Emails:  []string{"admin@company.de", "super-admin+devops@company.de"},
					MSTeams: []string{"1234689745631@teams.microsoft.ms"},
				},
			}},
		},
		{
			name:            "CliDefaultNotificationsUsedWhenMetadataMissing",
			defaults:        &CollectorImage{Notifications: Notifications{Slack: []string{"#default"}}},
			annotationNames: &annotationNames,
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				NamespaceName: "myNamespace",
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:tag",
				Notifications: Notifications{
					Slack: []string{"#default"},
				},
			}},
		},
		{
			name:            "ImageNotificationRuleOverridesDefaultNotifications",
			defaults:        &CollectorImage{Notifications: Notifications{Slack: []string{"#default"}}},
			annotationNames: &annotationNames,
			runConfig: &RunConfig{ImageNotificationRules: []ImageNotificationRule{{
				Image: "^quay\\.io/name:.*$",
				Notifications: Notifications{
					Slack: []string{"#rule"},
				},
			}}},
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				NamespaceName: "myNamespace",
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:tag",
				Notifications: Notifications{
					Slack: []string{"#rule"},
				},
			}},
		},
		{
			name:            "ImageNotificationRuleOverridesAnnotationNotifications",
			defaults:        &CollectorImage{},
			annotationNames: &annotationNames,
			runConfig: &RunConfig{ImageNotificationRules: []ImageNotificationRule{{
				Image: "^quay\\.io/name:.*$",
				Notifications: Notifications{
					Emails: []string{"rule@example.com"},
				},
			}}},
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				NamespaceName: "myNamespace",
				Annotations: map[string]string{"contact.sda.se/notifications": "{" +
					"\"slack\":[\"channel1\"]" +
					"}"},
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:tag",
				Notifications: Notifications{
					Emails: []string{"rule@example.com"},
				},
			}},
		},
		{
			name:            "FirstMatchingImageNotificationRuleWins",
			defaults:        &CollectorImage{},
			annotationNames: &annotationNames,
			runConfig: &RunConfig{ImageNotificationRules: []ImageNotificationRule{
				{
					Image: "^quay\\.io/name:.*$",
					Notifications: Notifications{
						Slack: []string{"#first"},
					},
				},
				{
					Image: "^quay\\.io/name:tag$",
					Notifications: Notifications{
						Slack: []string{"#second"},
					},
				},
			}},
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				NamespaceName: "myNamespace",
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:tag",
				Notifications: Notifications{
					Slack: []string{"#first"},
				},
			}},
		},
		{
			name:            "NonMatchingImageNotificationRuleKeepsExistingNotifications",
			defaults:        &CollectorImage{Notifications: Notifications{Slack: []string{"#default"}}},
			annotationNames: &annotationNames,
			runConfig: &RunConfig{ImageNotificationRules: []ImageNotificationRule{{
				Image: "^ghcr\\.io/acme/.*$",
				Notifications: Notifications{
					Slack: []string{"#rule"},
				},
			}}},
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				NamespaceName: "myNamespace",
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:tag",
				Notifications: Notifications{
					Slack: []string{"#default"},
				},
			}},
		},

		{
			name:            "OwnerAnnotation",
			defaults:        &CollectorImage{},
			annotationNames: &annotationNames,
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				NamespaceName: "myNamespace",
				ImageType:     kubeclient.ImageTypeCronJob,
				Annotations:   map[string]string{"contact.sda.se/owners": "[{\"role\":\"ADMIN\",\"uuid\":\"550e8400-e29b-41d4-a716-446655440000\",\"name\":\"Alice\"}]"},
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:tag",
				ImageType: kubeclient.ImageTypeCronJob,
				Owners:    []Owner{{Role: "ADMIN", Uuid: "550e8400-e29b-41d4-a716-446655440000", Name: "Alice"}},
			}},
		},
		{
			name:            "OwnerAnnotationMultipleFullOwners",
			defaults:        &CollectorImage{},
			annotationNames: &annotationNames,
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				NamespaceName: "myNamespace",
				ImageType:     kubeclient.ImageTypeCronJob,
				Annotations: map[string]string{"contact.sda.se/owners": "[" +
					"{\"role\":\"ADMIN\",\"uuid\":\"123456-e29b-41d4-a716-446655440000\",\"name\":\"Alice1\"}," +
					"{\"role\":\"VIEWER\",\"uuid\":\"789456-e29b-41d4-a716-446655440000\",\"name\":\"Alice2\"}," +
					"{\"role\":\"UNKNOWN\",\"uuid\":\"123789-e29b-41d4-a716-446655440000\",\"name\":\"Alice3\"}" +
					"]"},
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:tag",
				ImageType: kubeclient.ImageTypeCronJob,
				Owners: []Owner{
					{Role: "ADMIN", Uuid: "123456-e29b-41d4-a716-446655440000", Name: "Alice1"},
					{Role: "VIEWER", Uuid: "789456-e29b-41d4-a716-446655440000", Name: "Alice2"},
					{Role: "UNKNOWN", Uuid: "123789-e29b-41d4-a716-446655440000", Name: "Alice3"},
				},
			}},
		},
		{
			name:            "OwnerAnnotationNonFullAttributes",
			defaults:        &CollectorImage{},
			annotationNames: &annotationNames,
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag",
				NamespaceName: "myNamespace",
				ImageType:     kubeclient.ImageTypeCronJob,
				Annotations: map[string]string{"contact.sda.se/owners": "[" +
					"{\"role\":\"ADMIN\",\"name\":\"Alice1\"}," +
					"{\"uuid\":\"789456-e29b-41d4-a716-446655440000\",\"name\":\"Alice2\"}," +
					"{\"role\":\"UNKNOWN\",\"uuid\":\"123789-e29b-41d4-a716-446655440000\"}" +
					"]"},
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",
				ImageId:   "quay.io/name:tag",
				ImageType: kubeclient.ImageTypeCronJob,
				Owners: []Owner{
					{Role: "ADMIN", Name: "Alice1"},
					{Uuid: "789456-e29b-41d4-a716-446655440000", Name: "Alice2"},
					{Role: "UNKNOWN", Uuid: "123789-e29b-41d4-a716-446655440000"},
				},
			}},
		},
		{
			name:            "MultipleImagesFromMultipleNamespaces",
			defaults:        &defaults,
			annotationNames: &annotationNames,
			targetK8Image: &[]kubeclient.Image{{
				Image:         "quay.io/name:tag-1",
				ImageId:       "quay.io/name@sha256:1234",
				NamespaceName: "myNamespace-1",
				Annotations:   map[string]string{"scans.sda.se/is-scan-malware": "false", "scans.sda.se/is-scan-distroless": "false"},
				Labels:        map[string]string{"contact.sda.se/team": "team-1"},
			}, {
				Image:         "quay.io/name:tag-2",
				ImageId:       "quay.io/name@sha256:2222",
				NamespaceName: "myNamespace-1",
				Annotations:   map[string]string{"scans.sda.se/is-scan-malware": "true", "scans.sda.se/is-scan-distroless": "false"},
				Labels:        map[string]string{"contact.sda.se/team": "team-2"},
			}, {
				Image:         "quay.io/name:tag-3",
				ImageId:       "quay.io/name@sha256:3333",
				NamespaceName: "myNamespace-2",
				Annotations:   map[string]string{"scans.sda.se/is-scan-malware": "false", "scans.sda.se/is-scan-distroless": "true"},
				Labels:        map[string]string{"contact.sda.se/team": "team-3"},
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace-1",
				Image:     "quay.io/name:tag-1",
				ImageId:   "quay.io/name@sha256:1234",

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           "team-1",
				Owners:         []Owner{},
				EngagementTags: defaults.EngagementTags,

				IsScanBaseimageLifetime: defaults.IsScanBaseimageLifetime,
				IsScanDependencyCheck:   defaults.IsScanDependencyCheck,
				IsScanDependencyTrack:   defaults.IsScanDependencyTrack,
				IsScanDistroless:        false,
				IsScanLifetime:          defaults.IsScanLifetime,
				IsScanMalware:           false,
			}, {
				Namespace: "myNamespace-1",
				Image:     "quay.io/name:tag-2",
				ImageId:   "quay.io/name@sha256:2222",

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           "team-2",
				Owners:         []Owner{},
				EngagementTags: defaults.EngagementTags,

				IsScanBaseimageLifetime: defaults.IsScanBaseimageLifetime,
				IsScanDependencyCheck:   defaults.IsScanDependencyCheck,
				IsScanDependencyTrack:   defaults.IsScanDependencyTrack,
				IsScanDistroless:        false,
				IsScanLifetime:          defaults.IsScanLifetime,
				IsScanMalware:           true,
			}, {
				Namespace: "myNamespace-2",
				Image:     "quay.io/name:tag-3",
				ImageId:   "quay.io/name@sha256:3333",

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           "team-3",
				Owners:         []Owner{},
				EngagementTags: defaults.EngagementTags,

				IsScanBaseimageLifetime: defaults.IsScanBaseimageLifetime,
				IsScanDependencyCheck:   defaults.IsScanDependencyCheck,
				IsScanDependencyTrack:   defaults.IsScanDependencyTrack,
				IsScanDistroless:        true,
				IsScanLifetime:          defaults.IsScanLifetime,
				IsScanMalware:           false,
			}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runConfig := tc.runConfig
			if runConfig == nil {
				runConfig = &RunConfig{
					ImageFilter: []string{},
				}
			}
			results, err := ConvertImages(tc.targetK8Image, tc.defaults, tc.annotationNames, runConfig)
			ensureSchemaVersion(tc.expectedCollectorImage)

			assert.NoError(t, err, "Expected no error, got %v", err)
			assert.Len(t, *results, len(*tc.expectedCollectorImage), "Lengths does not match. Expected %v, got %v,", len(*tc.expectedCollectorImage), len(*results))
			assert.True(t, reflect.DeepEqual(results, tc.expectedCollectorImage), "Expected %v, got %v,", *tc.expectedCollectorImage, *results)
		})
	}
}

func TestStore(t *testing.T) {
	defaults := CollectorImage{
		Environment: "myEnv",
		// Destination: "Lorem Ipsum Dolor Sit Amet",
		ContainerType:  "myContainerType",
		Team:           "myTeam",
		EngagementTags: []string{"defaultTag"},
		Owners:         []Owner{{}},

		IsScanBaseimageLifetime: true,
		IsScanDependencyCheck:   true,
		IsScanDependencyTrack:   true,
		IsScanDistroless:        true,
		IsScanLifetime:          true,
		IsScanMalware:           true,
	}

	fixtures := []CollectorImage{
		{
			Namespace: "myNamespace",
			Image:     "quay.io/name:tag",

			Environment:    defaults.Environment,
			ContainerType:  defaults.ContainerType,
			Team:           defaults.Team,
			Owners:         defaults.Owners,
			EngagementTags: defaults.EngagementTags,

			IsScanBaseimageLifetime: defaults.IsScanBaseimageLifetime,
			IsScanDependencyCheck:   defaults.IsScanDependencyCheck,
			IsScanDependencyTrack:   defaults.IsScanDependencyTrack,
			IsScanDistroless:        defaults.IsScanDistroless,
			IsScanLifetime:          defaults.IsScanLifetime,
			IsScanMalware:           defaults.IsScanMalware,
		},
		{
			Namespace: "myNamespace",
			Image:     "quay.io/name:tag1",

			Environment:    defaults.Environment,
			ContainerType:  defaults.ContainerType,
			Team:           defaults.Team,
			Owners:         defaults.Owners,
			EngagementTags: defaults.EngagementTags,

			IsScanBaseimageLifetime: defaults.IsScanBaseimageLifetime,
			IsScanDependencyCheck:   defaults.IsScanDependencyCheck,
			IsScanDependencyTrack:   defaults.IsScanDependencyTrack,
			IsScanDistroless:        defaults.IsScanDistroless,
			IsScanLifetime:          defaults.IsScanLifetime,
			IsScanMalware:           defaults.IsScanMalware,
		},
		{
			Namespace: "myNamespace-1",
			Image:     "quay.io/name:tag-2",
			ImageId:   "quay.io/name@sha256:2222",

			Environment:    defaults.Environment,
			ContainerType:  defaults.ContainerType,
			Team:           "team-2",
			Owners:         defaults.Owners,
			EngagementTags: defaults.EngagementTags,

			IsScanBaseimageLifetime: defaults.IsScanBaseimageLifetime,
			IsScanDependencyCheck:   defaults.IsScanDependencyCheck,
			IsScanDependencyTrack:   defaults.IsScanDependencyTrack,
			IsScanDistroless:        false,
			IsScanLifetime:          defaults.IsScanLifetime,
			IsScanMalware:           true,
		},
	}
	ensureSchemaVersion(&fixtures)
	jsonResult, _ := JsonIndentMarshal(fixtures)

	cases := []struct {
		name         string
		fixtures     *[]CollectorImage
		expectResult any
		expectError  bool
	}{
		{
			name:         "Test valid input",
			fixtures:     &fixtures,
			expectResult: jsonResult,
			expectError:  false,
		},
		{
			name:         "Test empty input",
			fixtures:     &[]CollectorImage{},
			expectResult: []byte("[]"),
			expectError:  false,
		},
		{
			name:         "Test nil input",
			fixtures:     nil,
			expectResult: []byte{},
			expectError:  true,
		},
	}

	for _, tc := range cases {
		var mockWriter bytes.Buffer

		t.Run(tc.name, func(t *testing.T) {
			err := Store(tc.fixtures, &mockWriter, JsonIndentMarshal)
			if tc.expectError {
				assert.Error(t, err, "Expected error but got none")
			} else {
				writtenData := mockWriter.Bytes()
				assert.Equal(t, writtenData, tc.expectResult, "Marshaling failed. Expected %v, got %v", tc.expectResult, writtenData)
			}
		})
	}

}

func TestSchemaContract(t *testing.T) {
	schemaBytes, err := os.ReadFile("../../schema/image-metadata-collector-report-v1.schema.json")
	assert.NoError(t, err)

	var schema map[string]any
	err = json.Unmarshal(schemaBytes, &schema)
	assert.NoError(t, err)

	payload := []CollectorImage{
		{
			SchemaVersion:          SchemaVersionV1,
			Namespace:              "myNamespace",
			Image:                  "quay.io/name:tag",
			ImageId:                "quay.io/name@sha256:1234",
			ImageType:              kubeclient.ImageTypeCronJob,
			Environment:            "dev",
			Product:                "image-metadata-collector",
			Description:            "test payload",
			AppKubernetesIoName:    "collector",
			AppKubernetesIoVersion: "1.0.0",
			ContainerType:          "application",
			Skip:                   false,
			NamespaceFilter:        "",
			NamespaceFilterNegated: "",
			EngagementTags:         []string{"defaultTag"},
			Team:                   "team-a",
			Owners: []Owner{
				{Role: "ADMIN", Uuid: "550e8400-e29b-41d4-a716-446655440000", Name: "Alice"},
			},
			Notifications: Notifications{
				Slack:   []string{"channel-a"},
				Emails:  []string{"team@example.com"},
				MSTeams: []string{"team-id"},
			},
			IsScanBaseimageLifetime:          true,
			IsScanDependencyCheck:            true,
			IsScanDependencyTrack:            false,
			IsScanDistroless:                 true,
			IsScanLifetime:                   true,
			IsScanMalware:                    true,
			IsScanNewVersion:                 true,
			IsScanRunAsRoot:                  true,
			IsPotentiallyRunningAsRoot:       false,
			IsScanRunAsPrivileged:            false,
			IsPotentiallyRunningAsPrivileged: false,
			ScanLifetimeMaxDays:              14,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	assert.NoError(t, err)

	var instance any
	err = json.Unmarshal(payloadBytes, &instance)
	assert.NoError(t, err)

	assert.NoError(t, validateAgainstSchemaSubset(schema, instance))
}

func validateAgainstSchemaSubset(schema map[string]any, instance any) error {
	return validateNode(schema, instance, "$")
}

func validateNode(schema map[string]any, instance any, path string) error {
	if schemaType, ok := schema["type"].(string); ok {
		switch schemaType {
		case "array":
			items, ok := instance.([]any)
			if !ok {
				return fmt.Errorf("%s: expected array", path)
			}

			itemSchema, ok := schema["items"].(map[string]any)
			if !ok {
				return fmt.Errorf("%s: schema missing items", path)
			}

			for idx, item := range items {
				if err := validateNode(itemSchema, item, fmt.Sprintf("%s[%d]", path, idx)); err != nil {
					return err
				}
			}

		case "object":
			objectValue, ok := instance.(map[string]any)
			if !ok {
				return fmt.Errorf("%s: expected object", path)
			}

			requiredFields := toStringSlice(schema["required"])
			for _, field := range requiredFields {
				if _, ok := objectValue[field]; !ok {
					return fmt.Errorf("%s: missing required field %q", path, field)
				}
			}

			properties, _ := schema["properties"].(map[string]any)
			additionalProperties, hasAdditionalProperties := schema["additionalProperties"].(bool)
			if hasAdditionalProperties && !additionalProperties {
				for field := range objectValue {
					if _, ok := properties[field]; !ok {
						return fmt.Errorf("%s: unexpected field %q", path, field)
					}
				}
			}

			for fieldName, propertySchema := range properties {
				fieldValue, ok := objectValue[fieldName]
				if !ok {
					continue
				}

				propertySchemaMap, ok := propertySchema.(map[string]any)
				if !ok {
					return fmt.Errorf("%s.%s: invalid property schema", path, fieldName)
				}

				if err := validateNode(propertySchemaMap, fieldValue, path+"."+fieldName); err != nil {
					return err
				}
			}

		case "string":
			if _, ok := instance.(string); !ok {
				return fmt.Errorf("%s: expected string", path)
			}

		case "boolean":
			if _, ok := instance.(bool); !ok {
				return fmt.Errorf("%s: expected boolean", path)
			}

		case "integer":
			if _, ok := instance.(float64); !ok {
				return fmt.Errorf("%s: expected integer", path)
			}
		}
	}

	if enumValues, ok := schema["enum"].([]any); ok {
		match := false
		for _, enumValue := range enumValues {
			if reflect.DeepEqual(enumValue, instance) {
				match = true
				break
			}
		}
		if !match {
			return fmt.Errorf("%s: value %v not part of enum", path, instance)
		}
	}

	return nil
}

func toStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}

	result := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if ok {
			result = append(result, text)
		}
	}

	return result
}

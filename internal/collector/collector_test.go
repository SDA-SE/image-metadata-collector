package collector

import (
	// "sort"
	// "strings"
	"reflect"
	"testing"

	"github.com/SDA-SE/sdase-image-collector/internal/pkg/kubeclient"
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
			expectedResult: true,
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isSkipImage(&tc.targetImage)

			if result != tc.expectedResult {
				t.Fatalf("Expected %v, got %v, with Namespace=%s, Skip=%v, NamespaceFilter=%v, NamespaceFilterNegated=%v",
					tc.expectedResult,
					result,
					tc.targetImage.Namespace,
					tc.targetImage.Skip,
					tc.targetImage.NamespaceFilter,
					tc.targetImage.NamespaceFilterNegated)
			}
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
			expectedChanged: true,
			expectedResult:  true,
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initialSkip := tc.targetImage.Skip
			cleanCollectorImage(&tc.targetImage)

			if tc.expectedChanged && tc.targetImage.Skip == initialSkip {
				t.Fatalf("Expected Skip to change but it did not change")
			} else if !tc.expectedChanged && tc.targetImage.Skip != initialSkip {
				t.Fatalf("Expected Skip not to change but it did change")
			}

			if tc.targetImage.Skip != tc.expectedResult {
				t.Fatalf("Expected %v, got %v, with Namespace=%s, Skip=%v, NamespaceFilter=%v, NamespaceFilterNegated=%v",
					tc.expectedResult,
					tc.targetImage.Skip,
					tc.targetImage.Namespace,
					tc.targetImage.Skip,
					tc.targetImage.NamespaceFilter,
					tc.targetImage.NamespaceFilterNegated)
			}
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initialImage := tc.targetImage.Image
			initialImageId := tc.targetImage.ImageId

			cleanCollectorImage(&tc.targetImage)

			if tc.expectedImgChanged && tc.targetImage.Image == initialImage {
				t.Fatalf("Expected Image to change but it did not change")
			} else if !tc.expectedImgChanged && tc.targetImage.Image != initialImage {
				t.Fatalf("Expected Image not to change but it did change")
			}

			if tc.expectedImgIdChanged && tc.targetImage.ImageId == initialImageId {
				t.Fatalf("Expected ImageId to change but it did not change")
			} else if !tc.expectedImgIdChanged && tc.targetImage.ImageId != initialImageId {
				t.Fatalf("Expected ImageId not to change but it did change")
			}

			if tc.targetImage.Image != tc.expectedImage {
				t.Fatalf("Expected %v, got %v,", tc.expectedImage, tc.targetImage.Image)
			}

			if tc.targetImage.ImageId != tc.expectedImageId {
				t.Fatalf("Expected %v, got %v,", tc.expectedImageId, tc.targetImage.ImageId)
			}
		})
	}
}

func TestConvert(t *testing.T) {
	defaults := CollectorImage{
		Environment: "myEnv",
		// Destination: "Lorem Ipsum Dolor Sit Amet",
		ContainerType:  "myContainerType",
		Team:           "myTeam",
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

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           defaults.Team,
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

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           defaults.Team,
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

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           defaults.Team,
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

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           "some-none-default-team",
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

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           "some-none-default-team",
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
				Labels:        map[string]string{"contact.sda.se/team": "team-from-label"},
				Annotations:   map[string]string{"contact.sda.se/team": "team-from-annotations"},
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           "team-from-annotations",
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

				Environment:    defaults.Environment,
				ContainerType:  defaults.ContainerType,
				Team:           defaults.Team,
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
				NamespaceName: "myNamespace",
				Annotations:   map[string]string{"sda.se/description": "Lorem Ipsum Dolor Sit Amet"},
			}},
			expectedCollectorImage: &[]CollectorImage{{
				Namespace: "myNamespace",
				Image:     "quay.io/name:tag",

				Environment:    defaults.Environment,
				Description:    "Lorem Ipsum Dolor Sit Amet",
				ContainerType:  defaults.ContainerType,
				Team:           defaults.Team,
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
			results, err := ConvertImages(tc.targetK8Image, tc.defaults, tc.annotationNames)

			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if len(*results) != len(*tc.expectedCollectorImage) {
				t.Fatalf("Lengths does not match. Expected %v, got %v,", len(*tc.expectedCollectorImage), len(*results))
			}

			if !reflect.DeepEqual(results, tc.expectedCollectorImage) {
				t.Fatalf("Expected %v, got %v,", *tc.expectedCollectorImage, *results)
			}

		})
	}
}

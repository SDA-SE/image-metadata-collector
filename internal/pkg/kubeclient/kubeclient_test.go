package kubeclient

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	testclient "k8s.io/client-go/kubernetes/fake"
	"sort"
	"strings"
	"testing"
)

func TestGetNamespaces(t *testing.T) {
	var client Client

	testCases := []struct {
		name               string
		namespaces         []runtime.Object
		expectedNamespaces map[string]Namespace
		expectSuccess      bool
	}{
		{
			name:               "NoNamespaces",
			namespaces:         []runtime.Object{},
			expectedNamespaces: map[string]Namespace{},
			expectSuccess:      true,
		},
		{
			name: "ExistingNamespaceWOLablesOrAnnotations",
			namespaces: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_ns_1",
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_ns_2",
					},
				},
			},
			expectedNamespaces: map[string]Namespace{"test_ns_1": Namespace{Name: "test_ns_1"}, "test_ns_2": Namespace{Name: "test_ns_2"}},
			expectSuccess:      true,
		},
		{
			name: "ExistingNamespaceWithSingleLable",
			namespaces: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "test_ns_1",
						Labels: map[string]string{"label_a": "val_a"},
					},
				},
			},
			expectedNamespaces: map[string]Namespace{"test_ns_1": Namespace{Name: "test_ns_1", Labels: map[string]string{"label_a": "val_a"}}},
			expectSuccess:      true,
		},
		{
			name: "ExistingNamespaceWithSingleAnnotation",
			namespaces: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_1",
						Annotations: map[string]string{"ann_a": "val_a"},
					},
				},
			},
			expectedNamespaces: map[string]Namespace{"test_ns_1": Namespace{Name: "test_ns_1", Annotations: map[string]string{"ann_a": "val_a"}}},
			expectSuccess:      true,
		},
		{
			name: "ExistingNamespaceWithSingleAnnotationAndLabel",
			namespaces: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_1",
						Labels:      map[string]string{"label_a": "val_a"},
						Annotations: map[string]string{"ann_a": "val_a"},
					},
				},
			},
			expectedNamespaces: map[string]Namespace{
				"test_ns_1": Namespace{
					Name:        "test_ns_1",
					Labels:      map[string]string{"label_a": "val_a"},
					Annotations: map[string]string{"ann_a": "val_a"},
				}},
			expectSuccess: true,
		},
		{
			name: "ExistingNamespacesWithMultipleAnnotationAndLabel",
			namespaces: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_1",
						Labels:      map[string]string{"label_a": "val_a", "label_b": "val_b"},
						Annotations: map[string]string{"ann_a": "val_a", "ann_b": "val_b"},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_2",
						Labels:      map[string]string{"label_c": "val_c", "label_d": "val_d"},
						Annotations: map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
					},
				},
			},
			expectedNamespaces: map[string]Namespace{
				"test_ns_1": Namespace{
					Name:        "test_ns_1",
					Labels:      map[string]string{"label_a": "val_a", "label_b": "val_b"},
					Annotations: map[string]string{"ann_a": "val_a", "ann_b": "val_b"},
				},
				"test_ns_2": Namespace{
					Name:        "test_ns_2",
					Labels:      map[string]string{"label_c": "val_c", "label_d": "val_d"},
					Annotations: map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
				},
			},
			expectSuccess: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client.Clientset = testclient.NewSimpleClientset(tc.namespaces...)
			namespaces, err := client.GetNamespaces()

			if tc.expectSuccess && err != nil {
				t.Fatalf("Got an error=%v\n", err)
			} else if !tc.expectSuccess && err == nil {
				t.Fatalf("Expected an error but got none\n")
			} else if len(*namespaces) != len(tc.expectedNamespaces) {
				t.Fatalf("Expected %d namespaces but got %d\n", len(tc.expectedNamespaces), len(*namespaces))
			}

			for _, ns := range *namespaces {
				expectedNs, ok := tc.expectedNamespaces[ns.Name]

				if !ok {
					t.Fatalf("Expected namespace %s but got none\n", ns.Name)
				}

				for label, value := range expectedNs.Labels {
					labelValue, ok := ns.Labels[label]
					if !ok {
						t.Fatalf("Expected label %s but got none\n", label)
					} else if labelValue != value {
						t.Fatalf("Expected label %s with value %s but got %s\n", label, value, labelValue)
					}
				}

				for annotation, value := range expectedNs.Annotations {
					annotationValue, ok := ns.Annotations[annotation]
					if !ok {
						t.Fatalf("Expected annotation %s but got none\n", annotation)
					} else if annotationValue != value {
						t.Fatalf("Expected annotation %s with value %s but got %s\n", annotation, value, annotationValue)
					}
				}
			}

		})
	}
}

func TestGetImages(t *testing.T) {
	var client Client

	testCases := []struct {
		name             string
		pods             []runtime.Object
		targetNamespaces []Namespace
		expectedImages   []Image
		expectSuccess    bool
	}{
		{
			name:             "NoNamespacesNoPods",
			pods:             []runtime.Object{},
			targetNamespaces: []Namespace{},
			expectedImages:   []Image{},
			expectSuccess:    true,
		},
		{
			name: "ExistingNamespaceAndPodsNoImage",
			pods: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_1",
						Labels:      map[string]string{"label_a": "val_a"},
						Annotations: map[string]string{"ann_a": "val_a"},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_2",
						Labels:      map[string]string{"label_c": "val_c", "label_d": "val_d"},
						Annotations: map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "test_ns_1",
						Labels: map[string]string{
							"label1": "value1",
						},
					},
				},
			},
			targetNamespaces: []Namespace{
				Namespace{
					Name:        "test_ns_1",
					Labels:      map[string]string{"label_a": "val_a", "label_b": "val_b"},
					Annotations: map[string]string{"ann_a": "val_a", "ann_b": "val_b"},
				},
				Namespace{
					Name:        "test_ns_2",
					Labels:      map[string]string{"label_c": "val_c", "label_d": "val_d"},
					Annotations: map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
				},
			},
			expectedImages: []Image{},
			expectSuccess:  true,
		},
		{
			name: "ExistingNamespaceAndPodsSingleImage",
			pods: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_1",
						Labels:      map[string]string{"label_a": "val_a"},
						Annotations: map[string]string{"ann_a": "val_a"},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_2",
						Labels:      map[string]string{"label_c": "val_c", "label_d": "val_d"},
						Annotations: map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "test_ns_1",
						Labels: map[string]string{
							"pod_label_1": "value_1",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "container1",
								Image: "quay.io/test/test:latest",
							},
						},
					},
				},
			},
			targetNamespaces: []Namespace{
				Namespace{
					Name:        "test_ns_1",
					Labels:      map[string]string{"label_a": "val_a", "label_b": "val_b"},
					Annotations: map[string]string{"ann_a": "val_a", "ann_b": "val_b"},
				},
				Namespace{
					Name:        "test_ns_2",
					Labels:      map[string]string{"label_c": "val_c", "label_d": "val_d"},
					Annotations: map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
				},
			},
			expectedImages: []Image{
				Image{
					Image:         "quay.io/test/test:latest",
					ImageId:       "",
					NamespaceName: "test_ns_1",
					Labels:        map[string]string{"pod_label_1": "value_1", "label_a": "val_a"},
					Annotations:   map[string]string{"ann_a": "val_a"},
				},
			},
			expectSuccess: true,
		},
		{
			name: "TargetLessNamespacesThanImages",
			pods: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_1",
						Labels:      map[string]string{"label_a": "val_a"},
						Annotations: map[string]string{"ann_a": "val_a"},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_2",
						Labels:      map[string]string{"label_c": "val_c", "label_d": "val_d"},
						Annotations: map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_ns_3",
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "test_ns_1",
						Labels: map[string]string{
							"pod_label_1": "value_1",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "container1",
								Image: "quay.io/test/test:latest",
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "test_ns_2",
						Labels: map[string]string{
							"pod_label_1": "value_1",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "container2",
								Image: "quay.io/test/test:v2",
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod3",
						Namespace: "test_ns_2",
						Labels: map[string]string{
							"pod_label_1": "value_1",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "container3",
								Image: "quay.io/test/test:3",
							},
						},
					},
				},
			},
			targetNamespaces: []Namespace{
				Namespace{
					Name:        "test_ns_1",
					Labels:      map[string]string{"label_a": "val_a", "label_b": "val_b"},
					Annotations: map[string]string{"ann_a": "val_a", "ann_b": "val_b"},
				},
			},
			expectedImages: []Image{
				Image{
					Image:         "quay.io/test/test:latest",
					ImageId:       "",
					NamespaceName: "test_ns_1",
					Labels:        map[string]string{"pod_label_1": "value_1", "label_a": "val_a"},
					Annotations:   map[string]string{"ann_a": "val_a"},
				},
			},
			expectSuccess: true,
		},
		{
			name: "TargetMultipleImagesInSingleNamespace",
			pods: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_1",
						Labels:      map[string]string{"label_a": "val_a"},
						Annotations: map[string]string{"ann_a": "val_a"},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_2",
						Labels:      map[string]string{"label_c": "val_c", "label_d": "val_d"},
						Annotations: map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_ns_3",
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "test_ns_1",
						Labels: map[string]string{
							"pod_label_1": "value_1",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "container1",
								Image: "quay.io/test/test:latest",
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "test_ns_2",
						Labels: map[string]string{
							"pod_label_1": "value_1",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "container2",
								Image: "quay.io/test/test:v2",
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod3",
						Namespace: "test_ns_2",
						Labels: map[string]string{
							"pod_label_1": "value_1",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "container3",
								Image: "quay.io/test/test:v3",
							},
						},
					},
				},
			},
			targetNamespaces: []Namespace{
				Namespace{
					Name:        "test_ns_2",
					Labels:      map[string]string{"label_c": "val_c", "label_d": "val_d"},
					Annotations: map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
				},
			},
			expectedImages: []Image{
				Image{
					Image:         "quay.io/test/test:v2",
					ImageId:       "",
					NamespaceName: "test_ns_2",
					Labels:        map[string]string{"pod_label_1": "value_1", "label_c": "val_c", "label_d": "val_d"},
					Annotations:   map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
				},
				Image{
					Image:         "quay.io/test/test:v3",
					ImageId:       "",
					NamespaceName: "test_ns_2",
					Labels:        map[string]string{"pod_label_1": "value_1", "label_c": "val_c", "label_d": "val_d"},
					Annotations:   map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
				},
			},
			expectSuccess: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client.Clientset = testclient.NewSimpleClientset(tc.pods...)
			images, err := client.GetImages(&tc.targetNamespaces)

			sort.Slice(*images, func(i, j int) bool {
				return strings.ToLower((*images)[i].Image) < strings.ToLower((*images)[j].Image)
			})

			sort.Slice(tc.expectedImages, func(i, j int) bool {
				return strings.ToLower(tc.expectedImages[i].Image) < strings.ToLower(tc.expectedImages[j].Image)
			})

			if tc.expectSuccess && err != nil {
				t.Fatalf("Got an error=%v\n", err)
			} else if !tc.expectSuccess && err == nil {
				t.Fatalf("Expected an error but got none\n")
			} else if len(*images) != len(tc.expectedImages) {
				t.Fatalf("Expected %d images but got %d, (images=%v)\n", len(tc.expectedImages), len(*images), *images)
			}

			for idx, img := range *images {
				expectedImg := tc.expectedImages[idx]

				if expectedImg.Image != img.Image {
					t.Fatalf("Expected image %s but got %s\n", expectedImg.Image, img.Image)
				}

				if expectedImg.ImageId != img.ImageId {
					t.Fatalf("Expected imageId %s but got %s\n", expectedImg.ImageId, img.ImageId)
				}

				if expectedImg.NamespaceName != img.NamespaceName {
					t.Fatalf("Expected namespace %s but got %s\n", expectedImg.NamespaceName, img.NamespaceName)
				}

				for label, value := range expectedImg.Labels {
					labelValue, ok := img.Labels[label]
					if !ok {
						t.Fatalf("Expected label %s but got none\n", label)
					} else if labelValue != value {
						t.Fatalf("Expected label %s with value %s but got %s\n", label, value, labelValue)
					}
				}

				for annotation, value := range expectedImg.Annotations {
					annotationValue, ok := img.Annotations[annotation]
					if !ok {
						t.Fatalf("Expected annotation %s but got none\n", annotation)
					} else if annotationValue != value {
						t.Fatalf("Expected annotation %s with value %s but got %s\n", annotation, value, annotationValue)
					}
				}
			}

		})
	}
}

func TestGetAllImages(t *testing.T) {
	var client Client

	testCases := []struct {
		name           string
		pods           []runtime.Object
		expectedImages []Image
		expectSuccess  bool
	}{
		{
			name:           "NoNamespacesNoPods",
			pods:           []runtime.Object{},
			expectedImages: []Image{},
			expectSuccess:  true,
		},
		{
			name: "ExistingNamespaceAndPodsNoImage",
			pods: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_1",
						Labels:      map[string]string{"label_a": "val_a"},
						Annotations: map[string]string{"ann_a": "val_a"},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_2",
						Labels:      map[string]string{"label_c": "val_c", "label_d": "val_d"},
						Annotations: map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "test_ns_1",
						Labels: map[string]string{
							"label1": "value1",
						},
					},
				},
			},
			expectedImages: []Image{},
			expectSuccess:  true,
		},
		{
			name: "ExistingNamespaceAndPodsSingleImage",
			pods: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_1",
						Labels:      map[string]string{"label_a": "val_a"},
						Annotations: map[string]string{"ann_a": "val_a"},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_2",
						Labels:      map[string]string{"label_c": "val_c", "label_d": "val_d"},
						Annotations: map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "test_ns_1",
						Labels: map[string]string{
							"pod_label_1": "value_1",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "container1",
								Image: "quay.io/test/test:latest",
							},
						},
					},
				},
			},
			expectedImages: []Image{
				Image{
					Image:         "quay.io/test/test:latest",
					ImageId:       "",
					NamespaceName: "test_ns_1",
					Labels:        map[string]string{"pod_label_1": "value_1", "label_a": "val_a"},
					Annotations:   map[string]string{"ann_a": "val_a"},
				},
			},
			expectSuccess: true,
		},
		{
			name: "TargetLessNamespacesThanImages",
			pods: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_1",
						Labels:      map[string]string{"label_a": "val_a"},
						Annotations: map[string]string{"ann_a": "val_a"},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_2",
						Labels:      map[string]string{"label_c": "val_c", "label_d": "val_d"},
						Annotations: map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_ns_3",
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "test_ns_1",
						Labels: map[string]string{
							"pod_label_1": "value_1",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "container1",
								Image: "quay.io/test/test:latest",
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "test_ns_2",
						Labels: map[string]string{
							"pod_label_1": "value_1",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "container2",
								Image: "quay.io/test/test:v2",
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod3",
						Namespace: "test_ns_2",
						Labels: map[string]string{
							"pod_label_1": "value_1",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "container3",
								Image: "quay.io/test/test:3",
							},
						},
					},
				},
			},
			expectedImages: []Image{
				Image{
					Image:         "quay.io/test/test:latest",
					ImageId:       "",
					NamespaceName: "test_ns_1",
					Labels:        map[string]string{"pod_label_1": "value_1", "label_a": "val_a"},
					Annotations:   map[string]string{"ann_a": "val_a"},
				},
				Image{
					Image:         "quay.io/test/test:v2",
					ImageId:       "",
					NamespaceName: "test_ns_2",
					Labels:        map[string]string{"pod_label_1": "value_1", "label_c": "val_c", "label_d": "val_d"},
					Annotations:   map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
				},
				Image{
					Image:         "quay.io/test/test:3",
					ImageId:       "",
					NamespaceName: "test_ns_2",
					Labels:        map[string]string{"pod_label_1": "value_1", "label_c": "val_c", "label_d": "val_d"},
					Annotations:   map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
				}},
			expectSuccess: true,
		},
		{
			name: "TargetMultipleImagesInSingleNamespace",
			pods: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_1",
						Labels:      map[string]string{"label_a": "val_a"},
						Annotations: map[string]string{"ann_a": "val_a"},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test_ns_2",
						Labels:      map[string]string{"label_c": "val_c", "label_d": "val_d"},
						Annotations: map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_ns_3",
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "test_ns_1",
						Labels: map[string]string{
							"pod_label_1": "value_1",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "container1",
								Image: "quay.io/test/test:latest",
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "test_ns_2",
						Labels: map[string]string{
							"pod_label_1": "value_1",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "container2",
								Image: "quay.io/test/test:v2",
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod3",
						Namespace: "test_ns_2",
						Labels: map[string]string{
							"pod_label_1": "value_1",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "container3",
								Image: "quay.io/test/test:v3",
							},
						},
					},
				},
			},
			expectedImages: []Image{
				Image{
					Image:         "quay.io/test/test:latest",
					ImageId:       "",
					NamespaceName: "test_ns_1",
					Labels:        map[string]string{"label_a": "val_a", "pod_label_1": "value_1"},
					Annotations:   map[string]string{"ann_a": "val_a"},
				},
				Image{
					Image:         "quay.io/test/test:v2",
					ImageId:       "",
					NamespaceName: "test_ns_2",
					Labels:        map[string]string{"pod_label_1": "value_1", "label_c": "val_c", "label_d": "val_d"},
					Annotations:   map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
				},
				Image{
					Image:         "quay.io/test/test:v3",
					ImageId:       "",
					NamespaceName: "test_ns_2",
					Labels:        map[string]string{"pod_label_1": "value_1", "label_c": "val_c", "label_d": "val_d"},
					Annotations:   map[string]string{"ann_c": "val_c", "ann_d": "val_d"},
				},
			},
			expectSuccess: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client.Clientset = testclient.NewSimpleClientset(tc.pods...)
			images, err := client.GetAllImages()

			sort.Slice(*images, func(i, j int) bool {
				return strings.ToLower((*images)[i].Image) < strings.ToLower((*images)[j].Image)
			})

			sort.Slice(tc.expectedImages, func(i, j int) bool {
				return strings.ToLower(tc.expectedImages[i].Image) < strings.ToLower(tc.expectedImages[j].Image)
			})

			if tc.expectSuccess && err != nil {
				t.Fatalf("Got an error=%v\n", err)
			} else if !tc.expectSuccess && err == nil {
				t.Fatalf("Expected an error but got none\n")
			} else if len(*images) != len(tc.expectedImages) {
				t.Fatalf("Expected %d images but got %d, (images=%v)\n", len(tc.expectedImages), len(*images), *images)
			}

			for idx, img := range *images {
				expectedImg := tc.expectedImages[idx]

				if expectedImg.Image != img.Image {
					t.Fatalf("Expected image %s but got %s\n", expectedImg.Image, img.Image)
				}

				if expectedImg.ImageId != img.ImageId {
					t.Fatalf("Expected imageId %s but got %s\n", expectedImg.ImageId, img.ImageId)
				}

				if expectedImg.NamespaceName != img.NamespaceName {
					t.Fatalf("Expected namespace %s but got %s\n", expectedImg.NamespaceName, img.NamespaceName)
				}

				for label, value := range expectedImg.Labels {
					labelValue, ok := img.Labels[label]
					if !ok {
						t.Fatalf("Expected label %s but got none\n", label)
					} else if labelValue != value {
						t.Fatalf("Expected label %s with value %s but got %s\n", label, value, labelValue)
					}
				}

				for annotation, value := range expectedImg.Annotations {
					annotationValue, ok := img.Annotations[annotation]
					if !ok {
						t.Fatalf("Expected annotation %s but got none\n", annotation)
					} else if annotationValue != value {
						t.Fatalf("Expected annotation %s with value %s but got %s\n", annotation, value, annotationValue)
					}
				}
			}

		})
	}
}

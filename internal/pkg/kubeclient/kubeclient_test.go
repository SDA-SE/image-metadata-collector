package kubeclient

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestGetNamespaces(t *testing.T) {
	t.Parallel()

	client := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/namespaces" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]any{
			"items": []map[string]any{
				{
					"metadata": map[string]any{
						"name":        "test-ns-1",
						"labels":      map[string]string{"label-a": "value-a"},
						"annotations": map[string]string{"ann-a": "value-a"},
					},
				},
				{
					"metadata": map[string]any{
						"name": "test-ns-2",
					},
				},
			},
		})
	})

	namespaces, err := client.GetNamespaces()
	if err != nil {
		t.Fatalf("GetNamespaces returned error: %v", err)
	}

	expected := []Namespace{
		{
			Name:        "test-ns-1",
			Labels:      map[string]string{"label-a": "value-a"},
			Annotations: map[string]string{"ann-a": "value-a"},
		},
		{Name: "test-ns-2"},
	}

	if !reflect.DeepEqual(expected, *namespaces) {
		t.Fatalf("unexpected namespaces: %#v", *namespaces)
	}
}

func TestGetNamespacesReturnsAPIError(t *testing.T) {
	t.Parallel()

	client := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusForbidden)
	})

	_, err := client.GetNamespaces()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "status 403") {
		t.Fatalf("expected status in error, got %v", err)
	}
}

func TestGetImages(t *testing.T) {
	t.Parallel()

	client := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/namespaces/team-a/pods":
			writeJSON(t, w, map[string]any{
				"items": []map[string]any{
					{
						"metadata": map[string]any{
							"name":        "pod-a",
							"labels":      map[string]string{"workload": "pod"},
							"annotations": map[string]string{"pod-ann": "1"},
						},
						"spec": map[string]any{
							"containers": []map[string]string{
								{"name": "main", "image": "registry/app:v1"},
								{"name": "sidecar", "image": "registry/sidecar:v1"},
							},
							"initContainers": []map[string]string{
								{"name": "init-a", "image": "registry/init:v1"},
							},
						},
						"status": map[string]any{
							"containerStatuses": []map[string]string{
								{"name": "main", "image": "registry/app@sha256:abc", "imageID": "docker-pullable://registry/app@sha256:abc"},
							},
							"initContainerStatuses": []map[string]string{
								{"name": "init-a", "image": "registry/init@sha256:def", "imageID": "docker-pullable://registry/init@sha256:def"},
							},
						},
					},
				},
			})
		case "/apis/batch/v1/namespaces/team-a/jobs":
			writeJSON(t, w, map[string]any{
				"items": []map[string]any{
					{
						"metadata": map[string]any{
							"name":   "job-a",
							"labels": map[string]string{"job-label": "job"},
						},
						"spec": map[string]any{
							"template": map[string]any{
								"spec": map[string]any{
									"containers": []map[string]string{
										{"name": "job-main", "image": "registry/job:v1"},
									},
									"initContainers": []map[string]string{
										{"name": "job-init", "image": "registry/job-init:v1"},
									},
								},
							},
						},
					},
				},
			})
		case "/apis/batch/v1/namespaces/team-a/cronjobs":
			writeJSON(t, w, map[string]any{
				"items": []map[string]any{
					{
						"metadata": map[string]any{
							"name":        "cron-a",
							"annotations": map[string]string{"cron-ann": "cron"},
						},
						"spec": map[string]any{
							"jobTemplate": map[string]any{
								"spec": map[string]any{
									"template": map[string]any{
										"spec": map[string]any{
											"containers": []map[string]string{
												{"name": "cron-main", "image": "registry/cron:v1"},
											},
											"initContainers": []map[string]string{
												{"name": "cron-init", "image": "registry/cron-init:v1"},
											},
										},
									},
								},
							},
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	})

	namespaces := []Namespace{
		{
			Name:        "team-a",
			Labels:      map[string]string{"ns-label": "ns", "workload": "namespace"},
			Annotations: map[string]string{"ns-ann": "ns"},
		},
	}

	images, err := client.GetImages(&namespaces)
	if err != nil {
		t.Fatalf("GetImages returned error: %v", err)
	}

	sortImages(images)

	expected := []Image{
		{
			Image:         "registry/app@sha256:abc",
			ImageId:       "docker-pullable://registry/app@sha256:abc",
			NamespaceName: "team-a",
			Labels:        map[string]string{"ns-label": "ns", "workload": "namespace"},
			Annotations:   map[string]string{"ns-ann": "ns", "pod-ann": "1"},
			ImageType:     ImageTypeOther,
		},
		{
			Image:         "registry/cron-init:v1",
			NamespaceName: "team-a",
			Labels:        map[string]string{"ns-label": "ns", "workload": "namespace"},
			Annotations:   map[string]string{"cron-ann": "cron", "ns-ann": "ns"},
			ImageType:     ImageTypeInitContainer,
		},
		{
			Image:         "registry/cron:v1",
			NamespaceName: "team-a",
			Labels:        map[string]string{"ns-label": "ns", "workload": "namespace"},
			Annotations:   map[string]string{"cron-ann": "cron", "ns-ann": "ns"},
			ImageType:     ImageTypeCronJob,
		},
		{
			Image:         "registry/init@sha256:def",
			ImageId:       "docker-pullable://registry/init@sha256:def",
			NamespaceName: "team-a",
			Labels:        map[string]string{"ns-label": "ns", "workload": "namespace"},
			Annotations:   map[string]string{"ns-ann": "ns", "pod-ann": "1"},
			ImageType:     ImageTypeInitContainer,
		},
		{
			Image:         "registry/job-init:v1",
			NamespaceName: "team-a",
			Labels:        map[string]string{"job-label": "job", "ns-label": "ns", "workload": "namespace"},
			Annotations:   map[string]string{"ns-ann": "ns"},
			ImageType:     ImageTypeInitContainer,
		},
		{
			Image:         "registry/job:v1",
			NamespaceName: "team-a",
			Labels:        map[string]string{"job-label": "job", "ns-label": "ns", "workload": "namespace"},
			Annotations:   map[string]string{"ns-ann": "ns"},
			ImageType:     ImageTypeJob,
		},
		{
			Image:         "registry/sidecar:v1",
			NamespaceName: "team-a",
			Labels:        map[string]string{"ns-label": "ns", "workload": "namespace"},
			Annotations:   map[string]string{"ns-ann": "ns", "pod-ann": "1"},
			ImageType:     ImageTypeOther,
		},
	}

	sortImages(&expected)

	if !reflect.DeepEqual(expected, *images) {
		t.Fatalf("unexpected images: %#v", *images)
	}
}

func TestNewClientFromKubeconfigUsesDefaultPathAndContextOverride(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer expected-token" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		writeJSON(t, w, map[string]any{"items": []any{}})
	}))
	defer server.Close()

	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "config")
	if err := os.WriteFile(kubeconfigPath, []byte(fmt.Sprintf(`
current-context: current
clusters:
  - name: target
    cluster:
      server: %s
      insecure-skip-tls-verify: true
contexts:
  - name: current
    context:
      cluster: target
      user: ignored
  - name: override
    context:
      cluster: target
      user: preferred
users:
  - name: ignored
    user:
      token: ignored-token
  - name: preferred
    user:
      token: expected-token
`, server.URL)), 0o600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}

	originalDefaultPathFunc := defaultKubeconfigPathFunc
	defaultKubeconfigPathFunc = func() string { return kubeconfigPath }
	defer func() {
		defaultKubeconfigPathFunc = originalDefaultPathFunc
	}()

	client, err := newClientFromConfig(&KubeConfig{Context: "override"})
	if err != nil {
		t.Fatalf("newClientFromConfig returned error: %v", err)
	}

	if _, err := client.GetNamespaces(); err != nil {
		t.Fatalf("GetNamespaces returned error: %v", err)
	}
}

func TestNewClientFromKubeconfigUsesMasterURLAndTokenFileOverInlineToken(t *testing.T) {
	t.Parallel()

	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("from-token-file\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer from-token-file" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		writeJSON(t, w, map[string]any{"items": []any{}})
	}))
	defer server.Close()

	kubeconfigPath := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(kubeconfigPath, []byte(fmt.Sprintf(`
current-context: current
clusters:
  - name: target
    cluster:
      server: https://127.0.0.1:1
      insecure-skip-tls-verify: true
contexts:
  - name: current
    context:
      cluster: target
      user: preferred
users:
  - name: preferred
    user:
      token: stale-inline-token
      tokenFile: %s
`, tokenFile)), 0o600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}

	client, err := newClientFromConfig(&KubeConfig{ConfigFile: kubeconfigPath, MasterUrl: server.URL})
	if err != nil {
		t.Fatalf("newClientFromConfig returned error: %v", err)
	}

	if _, err := client.GetNamespaces(); err != nil {
		t.Fatalf("GetNamespaces returned error: %v", err)
	}
}

func TestNewClientFromKubeconfigUsesCAFileDataAndClientCertificate(t *testing.T) {
	t.Parallel()

	caPEM, serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM := generateTestCertificates(t)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.TLS.PeerCertificates) == 0 {
			t.Fatal("expected peer certificate")
		}
		writeJSON(t, w, map[string]any{"items": []any{}})
	}))
	server.TLS = &tls.Config{
		MinVersion: tls.VersionTLS12,
		ClientAuth: tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{
			mustTLSCertificate(t, serverCertPEM, serverKeyPEM),
		},
		ClientCAs: mustCertPool(t, caPEM),
	}
	server.StartTLS()
	defer server.Close()

	tempDir := t.TempDir()
	caFile := filepath.Join(tempDir, "ca.pem")
	if err := os.WriteFile(caFile, caPEM, 0o600); err != nil {
		t.Fatalf("write ca file: %v", err)
	}

	kubeconfigPath := filepath.Join(tempDir, "config")
	if err := os.WriteFile(kubeconfigPath, []byte(fmt.Sprintf(`
current-context: current
clusters:
  - name: file-cluster
    cluster:
      server: %s
      certificate-authority: %s
  - name: inline-cluster
    cluster:
      server: %s
      certificate-authority-data: %s
contexts:
  - name: current
    context:
      cluster: inline-cluster
      user: mtls-user
users:
  - name: mtls-user
    user:
      client-certificate-data: %s
      client-key-data: %s
`, server.URL, caFile, server.URL, base64.StdEncoding.EncodeToString(caPEM), base64.StdEncoding.EncodeToString(clientCertPEM), base64.StdEncoding.EncodeToString(clientKeyPEM))), 0o600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}

	client, err := newClientFromConfig(&KubeConfig{ConfigFile: kubeconfigPath})
	if err != nil {
		t.Fatalf("newClientFromConfig returned error: %v", err)
	}

	if _, err := client.GetNamespaces(); err != nil {
		t.Fatalf("GetNamespaces returned error: %v", err)
	}
}

func TestNewClientFromKubeconfigUsesTLSServerName(t *testing.T) {
	t.Parallel()

	caPEM, serverCertPEM, serverKeyPEM := generateHostnameOnlyServerCertificate(t, "kubernetes.default.svc")

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{"items": []any{}})
	}))
	server.TLS = &tls.Config{
		MinVersion: tls.VersionTLS12,
		Certificates: []tls.Certificate{
			mustTLSCertificate(t, serverCertPEM, serverKeyPEM),
		},
	}
	server.StartTLS()
	defer server.Close()

	tempDir := t.TempDir()
	caFile := filepath.Join(tempDir, "ca.pem")
	if err := os.WriteFile(caFile, caPEM, 0o600); err != nil {
		t.Fatalf("write ca file: %v", err)
	}

	kubeconfigPath := filepath.Join(tempDir, "config")
	if err := os.WriteFile(kubeconfigPath, []byte(fmt.Sprintf(`
current-context: current
clusters:
  - name: target
    cluster:
      server: https://127.0.0.1:1
      certificate-authority: %s
      tls-server-name: kubernetes.default.svc
contexts:
  - name: current
    context:
      cluster: target
      user: preferred
users:
  - name: preferred
    user:
      token: expected-token
`, caFile)), 0o600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}

	client, err := newClientFromConfig(&KubeConfig{ConfigFile: kubeconfigPath, MasterUrl: server.URL})
	if err != nil {
		t.Fatalf("newClientFromConfig returned error: %v", err)
	}

	if _, err := client.GetNamespaces(); err != nil {
		t.Fatalf("GetNamespaces returned error: %v", err)
	}
}

func TestNewClientFromKubeconfigRejectsExecAuth(t *testing.T) {
	t.Parallel()

	kubeconfigPath := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(kubeconfigPath, []byte(`
current-context: current
clusters:
  - name: target
    cluster:
      server: https://example.invalid
      insecure-skip-tls-verify: true
contexts:
  - name: current
    context:
      cluster: target
      user: exec-user
users:
  - name: exec-user
    user:
      exec:
        apiVersion: client.authentication.k8s.io/v1
        command: example
`), 0o600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}

	_, err := newClientFromConfig(&KubeConfig{ConfigFile: kubeconfigPath})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "exec plugins") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewClientUsesInClusterConfig(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer in-cluster-token" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		writeJSON(t, w, map[string]any{"items": []any{}})
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	tempDir := t.TempDir()
	tokenPath := filepath.Join(tempDir, "token")
	caPath := filepath.Join(tempDir, "ca.crt")

	cert := server.Certificate()
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})

	if err := os.WriteFile(tokenPath, []byte("in-cluster-token"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	if err := os.WriteFile(caPath, certPEM, 0o600); err != nil {
		t.Fatalf("write ca: %v", err)
	}

	originalTokenPath := serviceAccountTokenPath
	originalCAPath := serviceAccountCAPath
	originalDefaultPathFunc := defaultKubeconfigPathFunc
	defaultKubeconfigPathFunc = func() string { return "" }
	serviceAccountTokenPath = tokenPath
	serviceAccountCAPath = caPath

	t.Setenv(inClusterHostEnv, serverURL.Hostname())
	t.Setenv(inClusterPortEnv, serverURL.Port())
	defer func() {
		serviceAccountTokenPath = originalTokenPath
		serviceAccountCAPath = originalCAPath
		defaultKubeconfigPathFunc = originalDefaultPathFunc
	}()

	client, err := newClientFromConfig(&KubeConfig{})
	if err != nil {
		t.Fatalf("newClientFromConfig returned error: %v", err)
	}

	if _, err := client.GetNamespaces(); err != nil {
		t.Fatalf("GetNamespaces returned error: %v", err)
	}
}

func newTestAPIClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &Client{
		baseURL:    server.URL,
		httpClient: server.Client(),
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode JSON: %v", err)
	}
}

func sortImages(images *[]Image) {
	sort.Slice(*images, func(i, j int) bool {
		if (*images)[i].Image == (*images)[j].Image {
			return (*images)[i].ImageType < (*images)[j].ImageType
		}
		return (*images)[i].Image < (*images)[j].Image
	})
}

func generateTestCertificates(t *testing.T) ([]byte, []byte, []byte, []byte, []byte) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}

	now := time.Now()
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test-ca",
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create CA cert: %v", err)
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	serverCertPEM, serverKeyPEM := generateSignedCertificate(t, caTemplate, caKey, now, true, []string{"127.0.0.1", "localhost"})
	clientCertPEM, clientKeyPEM := generateSignedCertificate(t, caTemplate, caKey, now, false, nil)

	return caPEM, serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM
}

func generateSignedCertificate(t *testing.T, issuer *x509.Certificate, issuerKey *rsa.PrivateKey, now time.Time, isServer bool, hosts []string) ([]byte, []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		t.Fatalf("generate serial number: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "test-cert",
		},
		NotBefore: now.Add(-time.Hour),
		NotAfter:  now.Add(24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}

	if isServer {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		template.DNSNames = hosts
		template.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	} else {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}

	der, err := x509.CreateCertificate(rand.Reader, template, issuer, &key.PublicKey, issuerKey)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	return certPEM, keyPEM
}

func generateHostnameOnlyServerCertificate(t *testing.T, host string) ([]byte, []byte, []byte) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}

	now := time.Now()
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test-ca",
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create CA cert: %v", err)
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate server key: %v", err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "test-server",
		},
		NotBefore:   now.Add(-time.Hour),
		NotAfter:    now.Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{host},
	}

	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create server cert: %v", err)
	}

	serverCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER})
	serverKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})

	return caPEM, serverCertPEM, serverKeyPEM
}

func mustTLSCertificate(t *testing.T, certPEM, keyPEM []byte) tls.Certificate {
	t.Helper()

	certificate, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("load TLS certificate: %v", err)
	}

	return certificate
}

func mustCertPool(t *testing.T, certPEM []byte) *x509.CertPool {
	t.Helper()

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(certPEM) {
		t.Fatal("append cert to pool")
	}

	return pool
}

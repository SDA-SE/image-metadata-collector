package main

import (
	"flag"
	"net/http"

	"github.com/SDA-SE/sdase-image-collector/internal/cmd"
	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/collector"
	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/model"
	"github.com/SDA-SE/sdase-image-collector/internal/pkg/kubeclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.With().Caller().Logger()

	err := newCommand().Execute()
	cmd.CheckError(err)
}

var kubeconfig, kubecontext, masterURL, defaultTeamValue, defaultProductValue, product, environmentName string
var scanIntervalInSecondsVersionCollector int64
var isVersionCollector, isImageCollector bool
var s3ParameterEntry = model.S3parameterEntry{}
var gitParameterEntry = model.GitParameterEntry{}
var imageCollectorDefaults = model.ImageCollectorDefaults{}

func newCommand() *cobra.Command {

	c := &cobra.Command{
		Use:   "collector",
		Short: "Collect images, apps, and their versions.",
		Long: `collector is a tool that will scan 'Deployment's 'StatefulSet's and 'DaemonSet's 'Namespace's, and 'Pod's for version and image information and push these as metrics to Prometheus.
			Environment variables for image-collector:
				DEFAULT_SCAN_BASEIMAGE_LIFETIME
				DEFAULT_SCAN_DEPENDENCY_CHECK
				DEFAULT_SCAN_DEPENDENCY_TRACK
				DEFAULT_SCAN_DISTROLESS
				DEFAULT_SCAN_LIFETIME
				DEFAULT_SCAN_MALWARE
				DEFAULT_SCAN_NEW_VERSION
				DEFAULT_SCAN_RUN_AS_ROOT
				DEFAULT_SCAN_RUN_AS_PRIVILEGED
				DEFAULT_TEAM_NAME
				DEFAULT_CONTAINER_TYPE
				AWS_WEB_IDENTITY_TOKEN_FILE
				AWS_ROLE_ARN
				DEFAULT_ENGAGEMENT_TAGS
				ANNOTATION_NAME_ENGAGEMENT_TAG
				ANNOTATION_NAME_PRODUCT
				ANNOTATION_NAME_SLACK
				ANNOTATION_NAME_TEAM
				ANNOTATION_NAME_ROCKETCHAT
				ANNOTATION_NAME_CONTAINER_TYPE
				ANNOTATION_NAME_NAMESPACE_FILTER
				ANNOTATION_NAME_NAMESPACE_FILTER_NEGATED
`,
		Run: func(cmd *cobra.Command, args []string) {
			run()
		},
	}
	kubeclient.CreateClientOrDie(kubeconfig, kubecontext, masterURL)
	c.PersistentFlags().StringVar(&environmentName, "cluster-name", "", "Name of the team used for extracting data.")

	c.PersistentFlags().StringVar(&defaultTeamValue, "default-team-value", "unknown", "If no team/owner name can be extracted from a k8s resource, use this value.")
	c.PersistentFlags().StringVar(&defaultProductValue, "default-product-value", "unknown", "If no product name can be extracted from a k8s resource, use this value.")
	c.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	c.PersistentFlags().StringVar(&kubecontext, "kubecontext", "", "The context to use to talk to the Kubernetes apiserver. If unset defaults to whatever your current-context is (kubectl config current-context)")
	c.PersistentFlags().StringVar(&masterURL, "master", "", "URL of the API server")
	cmd.CheckError(c.MarkPersistentFlagRequired("cluster-name"))

	c.PersistentFlags().BoolVar(&isImageCollector, "is-imagecollector", true, "Enable the image collector")
	c.PersistentFlags().Int64Var(&imageCollectorDefaults.ScanIntervalInSeconds, "image-collector-scan-interval", 3600, "Rescan intervalInSeconds in seconds for image collector")
	c.PersistentFlags().StringVar(&imageCollectorDefaults.ConfigBasePath, "image-collector-config-basepath", "/config", "Configuration folder for the image collector")
	c.PersistentFlags().BoolVar(&imageCollectorDefaults.Skip, "image-collector-default-skip", false, "Images in namespaces are skipped without annotations/labels")
	c.PersistentFlags().BoolVar(&imageCollectorDefaults.IsSaveFiles, "image-collector-save-files", false, "In addition to uploading the files to S3, store the files on the disk")

	c.PersistentFlags().BoolVar(&s3ParameterEntry.Disabled, "image-collector-s3-disabled", false, "Disable S3")
	c.PersistentFlags().StringVar(&s3ParameterEntry.S3bucket, "image-collector-s3-bucket", "cluster-image-scanner-collector", "S3 Bucket to store image collector results")
	c.PersistentFlags().StringVar(&s3ParameterEntry.S3accessKey, "image-collector-s3-access-key", "", "S3 Access Key")
	c.PersistentFlags().StringVar(&s3ParameterEntry.S3secretKey, "image-collector-s3-secret-key", "", "S3 Secret Key")
	c.PersistentFlags().StringVar(&s3ParameterEntry.S3endpoint, "image-collector-s3-endpoint", "", "S3 Endpoint (e.g. minio)")
	c.PersistentFlags().StringVar(&s3ParameterEntry.S3region, "image-collector-s3-region", "eu-west-1", "S3 region")
	c.PersistentFlags().BoolVar(&s3ParameterEntry.S3insecure, "image-collector-s3-insecure", false, "Insecure bucket connection")
	c.PersistentFlags().BoolVar(&s3ParameterEntry.S3ForcePathStyle, "image-collector-s3-force-path-style", false, "Enforce S3 Force Path Style (should be true for minio)")

	c.PersistentFlags().StringVar(&gitParameterEntry.Password, "image-collector-git-password", "", "Git Password to connect")
	c.PersistentFlags().StringVar(&gitParameterEntry.Url, "image-collector-git-url", "", "Git URL to connect, use ")
	c.PersistentFlags().StringVar(&gitParameterEntry.PrivateKeyFile, "image-collector-git-private-key-file-path", "/home/nonroot/.ssh/id_rsa", "Path to the private ssh/github key file")
	c.PersistentFlags().StringVar(&gitParameterEntry.Directory, "image-collector-git-directory", "/tmp/git", "Directory to clone to")
	c.PersistentFlags().Int64Var(&gitParameterEntry.GithubAppId, "image-collector-github-app-id", 0, "Github AppId")
	c.PersistentFlags().Int64Var(&gitParameterEntry.GithubInstallationId, "image-collector-github-installation-id", 0, "Github InstallationId")

	var isDebug = false
	c.PersistentFlags().BoolVar(&isDebug, "debug", false, "Set logging level to debug, default logging level is info")
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if isDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	return c
}

func run() {
	client := kubeclient.CreateClientOrDie(kubeconfig, kubecontext, masterURL)
	imageCollectorDefaults.Client = client
	imageCollectorDefaults.Environment = environmentName
	go collector.Run(isImageCollector, imageCollectorDefaults, s3ParameterEntry, gitParameterEntry)

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	))
	err := http.ListenAndServe(":9402", nil)
	log.Fatal().Stack().Err(err).Msg("Could not start listener for version collector")

}

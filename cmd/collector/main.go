package main

import (
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/SDA-SE/sdase-image-collector/internal/cmd"
	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/collector"
	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/model"
	"github.com/SDA-SE/sdase-image-collector/internal/cmd/imagecollector/storage"
	"github.com/SDA-SE/sdase-image-collector/internal/pkg/kubeclient"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const AppName = "collector"
const replaceHyphenWithCamelCase = false

var kubeconfig, kubecontext, masterURL, defaultTeamValue, defaultProductValue, product, environmentName string
var scanIntervalInSecondsVersionCollector int64
var imageCollectorDefaults = model.ImageCollectorDefaults{}
var storageCfg = storage.StorageConfig{}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.With().Caller().Logger()

	err := newCommand().Execute()
	cmd.CheckError(err)
}

func newCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   AppName,
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
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeConfig(cmd)
		},
		Run: func(cmd *cobra.Command, args []string) {
			run()
		},
	}
	c.PersistentFlags().StringVar(&environmentName, "cluster-name", "", "Name of the team used for extracting data.")

	c.PersistentFlags().StringVar(&defaultTeamValue, "default-team-value", "unknown", "If no team/owner name can be extracted from a k8s resource, use this value.")
	c.PersistentFlags().StringVar(&defaultProductValue, "default-product-value", "unknown", "If no product name can be extracted from a k8s resource, use this value.")
	c.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	c.PersistentFlags().StringVar(&kubecontext, "kubecontext", "", "The context to use to talk to the Kubernetes apiserver. If unset defaults to whatever your current-context is (kubectl config current-context)")
	c.PersistentFlags().StringVar(&masterURL, "master", "", "URL of the API server")
	cmd.CheckError(c.MarkPersistentFlagRequired("cluster-name"))

	c.PersistentFlags().Int64Var(&imageCollectorDefaults.ScanIntervalInSeconds, "scan-interval", 3600, "Rescan intervalInSeconds in seconds for image collector")
	c.PersistentFlags().BoolVar(&imageCollectorDefaults.Skip, "default-skip", false, "Images in namespaces are skipped without annotations/labels")
	c.PersistentFlags().BoolVar(&imageCollectorDefaults.IsSaveFiles, "save-files", false, "In addition to uploading the files to S3, store the files on the disk")

	c.PersistentFlags().StringVar(&storageCfg.StorageFlag, "storage", "s3", "Write output to storage location [s3, git, local fs]")

	c.PersistentFlags().StringVar(&storageCfg.S3bucketName, "s3-bucket", "", "S3 Bucket to store image collector results")
	c.PersistentFlags().StringVar(&storageCfg.S3endpoint, "s3-endpoint", "", "S3 Endpoint (e.g. minio)")
	c.PersistentFlags().StringVar(&storageCfg.S3region, "s3-region", storageCfg.S3region, "S3 region")
	c.PersistentFlags().BoolVar(&storageCfg.S3insecure, "s3-insecure", false, "Insecure bucket connection")

	c.PersistentFlags().StringVar(&storageCfg.FsBaseDir, "fs-base-dir", "", "Directory to write the output to, if empty use stdout")

	c.PersistentFlags().StringVar(&storageCfg.GitPassword, "git-password", "", "Git Password to connect")
	c.PersistentFlags().StringVar(&storageCfg.GitUrl, "git-url", storageCfg.GitUrl, "Git URL to connect, use ")
	c.PersistentFlags().StringVar(&storageCfg.GitPrivateKeyFile, "git-private-key-file-path", storageCfg.GitPrivateKeyFile, "Path to the private ssh/github key file")
	c.PersistentFlags().StringVar(&storageCfg.GitDirectory, "git-directory", storageCfg.GitDirectory, "Directory to clone to")
	c.PersistentFlags().Int64Var(&storageCfg.GithubAppId, "github-app-id", storageCfg.GithubAppId, "Github AppId")
	c.PersistentFlags().Int64Var(&storageCfg.GithubInstallationId, "github-installation-id", storageCfg.GithubInstallationId, "Github InstallationId")

	var isDebug = false
	c.PersistentFlags().BoolVar(&isDebug, "debug", false, "Set logging level to debug, default logging level is info")
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if isDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	return c
}

// initializeConfig reads in ENV variables if set.
func initializeConfig(cmd *cobra.Command) error {
	v := viper.New()

	v.SetEnvPrefix(AppName)

	// Environment variables can't have dashes in them, so bind them to their equivalent
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	v.AutomaticEnv()
	bindFlags(cmd, v)

	return nil
}

// bindFlags binds each cobra flag to its associated viper configuration
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		configName := f.Name

		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}

func run() {
	client := kubeclient.CreateClientOrDie(kubeconfig, kubecontext, masterURL)
	imageCollectorDefaults.Client = client
	imageCollectorDefaults.Environment = environmentName

	s, err := storage.NewStorage(&storageCfg)

	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Could not create storage for: " + storageCfg.StorageFlag)
	}

	go collector.Run(imageCollectorDefaults, s)

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	))

	err = http.ListenAndServe(":9402", nil)
	log.Fatal().Stack().Err(err).Msg("Could not start listener for version collector")
}

package main

import (
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/SDA-SE/sdase-image-collector/internal/collector"
	"github.com/SDA-SE/sdase-image-collector/internal/config"
	"github.com/SDA-SE/sdase-image-collector/internal/pkg/kubeclient"
	"github.com/SDA-SE/sdase-image-collector/internal/pkg/storage"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const AppName = "collector"

const ShortDescription = "Collect images, apps, and their versions."
const LongDescription = `Collector is a tool that will scan
	'Deployment's
	'StatefulSet's
	and 'DaemonSet's
	'Namespace's,
	and 'Pod's
	for version and image information and push these as metrics to Prometheus.

	All Parameters can also be set as Environment Variables, following the pattern:
	--environment-name -> COLLECTOR_ENVIRONMENT_NAME
	--team -> COLLECTOR_TEAM
	--scan-interval -> COLLECTOR_SCAN_INTERVAL
	`

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.With().Caller().Logger()

	err := newCommand().Execute()
	if err != nil {
		log.Fatal().Err(err).Msg("Error running collector")
	}
}

func newCommand() *cobra.Command {
	cfg := &config.Config{}

	c := &cobra.Command{
		Use:   AppName,
		Short: ShortDescription,
		Long:  LongDescription,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeConfig(cmd)
		},
		Run: func(cmd *cobra.Command, args []string) {
			run(cfg)
		},
	}

	// Run Configuration
	c.PersistentFlags().DurationVar(&cfg.ScanInterval, "scan-interval", time.Duration(3600)*time.Second, "Rescan intervalInSeconds in seconds for image collector")
	c.PersistentFlags().BoolVar(&cfg.Debug, "debug", false, "Set logging level to debug, default logging level is info")
	c.PersistentFlags().BoolVar(&cfg.ExposeMetrics, "expose-metrics", true, "Expose metrics endpoint")

	// Kubernetes Config
	c.PersistentFlags().StringVar(&cfg.KubeConfig.ConfigFile, "kube-config", "", "absolute path to the kubeconfig file")
	c.PersistentFlags().StringVar(&cfg.KubeConfig.Context, "kube-context", "", "The context to use to talk to the Kubernetes apiserver. If unset defaults to whatever your current-context is (kubectl config current-context)")
	c.PersistentFlags().StringVar(&cfg.KubeConfig.MasterUrl, "master-url", "", "URL of the API server")

	// Output/Storage Config
	c.PersistentFlags().StringVar(&cfg.StorageConfig.StorageFlag, "storage", "s3", "Write output to storage location [s3, git, local fs]")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.S3bucketName, "s3-bucket", "", "S3 Bucket to store image collector results")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.S3endpoint, "s3-endpoint", "", "S3 Endpoint (e.g. minio)")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.S3region, "s3-region", "", "S3 region")
	c.PersistentFlags().BoolVar(&cfg.StorageConfig.S3insecure, "s3-insecure", false, "Insecure bucket connection")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.FsBaseDir, "fs-base-dir", "", "Directory to write the output to, if empty use stdout")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.GitPassword, "git-password", "", "Git Password to connect")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.GitUrl, "git-url", "", "Git URL to connect, use ")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.GitPrivateKeyFile, "git-private-key-file", "", "Path to the private ssh/github key file")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.GitDirectory, "git-directory", "", "Directory to clone to")
	c.PersistentFlags().Int64Var(&cfg.StorageConfig.GithubAppId, "github-app-id", 0, "Github AppId")
	c.PersistentFlags().Int64Var(&cfg.StorageConfig.GithubInstallationId, "github-installation-id", 0, "Github InstallationId")

	// Annotation Key/Name Config
	c.PersistentFlags().StringVar(&cfg.AnnotationNames.Base, "annotation-name-base", "sdase.org/", "Annotation name for general annotations")
	c.PersistentFlags().StringVar(&cfg.AnnotationNames.Scans, "annotation-name-scans", "clusterscanner.sdase.org/", "Annotation name for scan related annotations")
	c.PersistentFlags().StringVar(&cfg.AnnotationNames.Contact, "annotation-name-contact", "contact.sdase.org/", "Annotation name for contact related annotations")
	c.PersistentFlags().StringVar(&cfg.AnnotationNames.DefectDojo, "annotation-name-defect-dojo", "defectdojo.sdase.org/", "Annotation name for defectdojo related annotations")

	// Deployment wide Defaults
	c.PersistentFlags().StringVar(&cfg.CollectorImage.Environment, "environment-name", "", "Name of the environment")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanDependencyCheck, "is-scan-dependency-check", true, "Default enable/disable DependencyCheck scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanDependencyTrack, "is-scan-dependency-track", false, "Default enable/disable DependencyTrack scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanLifetime, "is-scan-lifetime", true, "Default enable/disable Lifetime scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanBaseimageLifetime, "is-scan-baseimage-lifetime", true, "Default enable/disable Baseimage Lifetime scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanDistroless, "is-scan-distroless", true, "Default enable/disable Distroless scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanMalware, "is-scan-malware", true, "Default enable/disable Malware scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanNewVersion, "is-scan-new-version", true, "Default enable/disable New Version scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanRunAsRoot, "is-scan-runasroot", true, "Default enable/disable RunAsRoot scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanRunAsPrivileged, "is-scan-run-as-privilaged", true, "Default enable/disable RunAsPrivileged scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsPotentiallyRunningAsRoot, "is-scan-potentially-running-as-root", true, "Default enable/disable PotentiallyRunningAsRoot scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsPotentiallyRunningAsPrivileged, "is-scan-potentially-running-as-privileged", true, "Default enable/disable PotentiallyRunningAsPrivileged scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.Skip, "skip", false, "Default behaviour for skipping scans for images")
	c.PersistentFlags().StringSliceVar(&cfg.CollectorImage.EngagementTags, "engagement-tags", []string{}, "Default engagement tags to use")
	c.PersistentFlags().StringVar(&cfg.CollectorImage.ContainerType, "container-type", "application", "Default container-type to use")
	c.PersistentFlags().StringVar(&cfg.CollectorImage.Team, "team", "", "Default team to use")
	c.PersistentFlags().StringVar(&cfg.CollectorImage.Product, "product", "", "Default product to use")
	c.PersistentFlags().StringVar(&cfg.CollectorImage.Slack, "slack", "", "Default slack channel to use")
	c.PersistentFlags().StringVar(&cfg.CollectorImage.Email, "email", "", "Default email to use")
	c.PersistentFlags().StringVar(&cfg.CollectorImage.Rocketchat, "rocketchat", "", "Default rocketchat channel to use")
	c.PersistentFlags().StringVar(&cfg.CollectorImage.NamespaceFilter, "namespace-filter", "", "Default namespace filter to use")
	c.PersistentFlags().StringVar(&cfg.CollectorImage.NamespaceFilterNegated, "negated-namespace-filter", "", "Default negated namespace filter to use")

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if cfg.Debug {
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
			err := cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			if err != nil {
				log.Fatal().Stack().Err(err).Msg("Could not set flag " + f.Name)
			}

		}
	})
}

// run starts the collector and metrics endpoint
func run(cfg *config.Config) {
	k8client := kubeclient.NewClient(&cfg.KubeConfig)

	storage, err := storage.NewStorage(&cfg.StorageConfig)

	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Could not create storage for: " + cfg.StorageConfig.StorageFlag)
	}

	collectorDefaults := &cfg.CollectorImage
	annotationNames := &cfg.AnnotationNames

	go func() {
		for {
			// Collect images from K8
			k8Images, err := k8client.GetAllImages()
			if err != nil {
				log.Fatal().Stack().Err(err).Msg("Could not retrieve images from K8")
			}

			// Convert & Clean k8 images to collector images
			images, err := collector.ConvertImages(k8Images, collectorDefaults, annotationNames)
			if err != nil {
				log.Fatal().Stack().Err(err).Msg("Could not collect images")
			}

			// Store images
			err = collector.Store(images, collectorDefaults.Environment, storage)
			if err != nil {
				log.Fatal().Stack().Err(err).Msg("Could not store collected images")
			}

			// Wait for next scan
			time.Sleep(cfg.ScanInterval)
		}
	}()

	if cfg.ExposeMetrics {
		// Expose the registered metrics via HTTP.
		http.Handle("/metrics", promhttp.HandlerFor(
			prometheus.DefaultGatherer,
			promhttp.HandlerOpts{
				// Opt into OpenMetrics to support exemplars.
				EnableOpenMetrics: true,
			},
		))

		err = http.ListenAndServe(":9402", nil)
		log.Fatal().Stack().Err(err).Msg("Could not start listener for image collector")
	}
}

package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/SDA-SE/image-metadata-collector/internal/collector"
	"github.com/SDA-SE/image-metadata-collector/internal/config"
	"github.com/SDA-SE/image-metadata-collector/internal/pkg/kubeclient"
	"github.com/SDA-SE/image-metadata-collector/internal/pkg/storage"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const AppName = "collector"

const ShortDescription = "Collect images"
const LongDescription = `Image Metadata Collector is a tool that will scan
	'Namespace's,
	and 'Pod's
	for image and team information.
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
			// Initialize the configuration
			if err := initializeConfig(cmd); err != nil {
				return err
			}

			// Set the logging level based on the debug flag
			if cfg.Debug {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			run(cfg)
		},
	}

	// Run Configuration
	c.PersistentFlags().BoolVar(&cfg.Debug, "debug", false, "Set logging level to debug, default logging level is info")
	c.Flags().StringSliceVarP(&cfg.RunConfig.ImageFilter, "image-filter", "s", []string{}, "Images to set the skip flag to true. Images as regex comma seperated without spaces. e.g. 'mock-service,mongo,openpolicyagent/opa,/istio/")

	// Kubernetes Config
	c.PersistentFlags().StringVar(&cfg.KubeConfig.ConfigFile, "kube-config", "", "absolute path to the kubeconfig file")
	c.PersistentFlags().StringVar(&cfg.KubeConfig.Context, "kube-context", "", "The context to use to talk to the Kubernetes apiserver. If unset defaults to whatever your current-context is (kubectl config current-context)")
	c.PersistentFlags().StringVar(&cfg.KubeConfig.MasterUrl, "master-url", "", "URL of the API server")

	// Output/Storage Config
	c.PersistentFlags().StringVar(&cfg.StorageConfig.StorageFlag, "storage", "api", "Write output to storage location [api, s3, git, local fs]")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.FileName, "filename", "", "Output filename, defaults to '<environment>-output.json'")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.S3BucketName, "s3-bucket", "", "S3 Bucket to store image collector results")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.S3Endpoint, "s3-endpoint", "", "S3 Endpoint (e.g. minio)")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.S3Region, "s3-region", "", "S3 region")
	c.PersistentFlags().BoolVar(&cfg.StorageConfig.S3Insecure, "s3-insecure", false, "Insecure bucket connection")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.GitPassword, "git-password", "", "Git Password to connect")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.GitUrl, "git-url", "", "Git URL to connect, use ")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.GitPrivateKeyFile, "git-private-key-file", "", "Path to the private ssh/github key file")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.GitDirectory, "git-directory", "", "Directory to clone to")
	c.PersistentFlags().Int64Var(&cfg.StorageConfig.GithubAppId, "github-app-id", 0, "Github AppId")
	c.PersistentFlags().Int64Var(&cfg.StorageConfig.GithubInstallationId, "github-installation-id", 0, "Github InstallationId")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.ApiKey, "api-key", "", "API Key")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.ApiSignature, "api-signature", "", "API Signature")
	c.PersistentFlags().StringVar(&cfg.StorageConfig.ApiEndpoint, "api-endpoint", "", "API Endpoint, e.g. https://example.io/v1/account/$ACCOUNT/cluster/$CLUSTER/image-collector-report/images")
	// HTTP Headers
	// use like: --http-header "Authorization:Bearer token" --http-header "Content-Type:application/json"
	pflag.StringArrayVar(&cfg.StorageConfig.HTTPHeaders, "http-header", []string{}, "List of HTTP headers in 'key:value' format. Repeat flag for multiple headers.")

	// Annotation Key/Name Config
	c.PersistentFlags().StringVar(&cfg.AnnotationNames.Base, "annotation-name-base", "sdase.org/", "Annotation name for general annotations")
	c.PersistentFlags().StringVar(&cfg.AnnotationNames.Scans, "annotation-name-scans", "clusterscanner.sdase.org/", "Annotation name for scan related annotations")
	c.PersistentFlags().StringVar(&cfg.AnnotationNames.Contact, "annotation-name-contact", "contact.sdase.org/", "Annotation name for contact related annotations")
	c.PersistentFlags().StringVar(&cfg.AnnotationNames.DefectDojo, "annotation-name-defect-dojo", "defectdojo.sdase.org/", "Annotation name for defectdojo related annotations")

	// Deployment wide Defaults
	c.PersistentFlags().StringVar(&cfg.CollectorImage.Environment, "environment-name", "", "Name of the environment")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanDependencyCheck, "is-scan-dependency-check", false, "Default enable/disable DependencyCheck scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanDependencyTrack, "is-scan-dependency-track", true, "Default enable/disable DependencyTrack scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanLifetime, "is-scan-lifetime", true, "Default enable/disable Lifetime scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanBaseimageLifetime, "is-scan-baseimage-lifetime", true, "Default enable/disable Baseimage Lifetime scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanDistroless, "is-scan-distroless", true, "Default enable/disable Distroless scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanMalware, "is-scan-malware", true, "Default enable/disable Malware scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanNewVersion, "is-scan-new-version", true, "Default enable/disable New Version scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanRunAsRoot, "is-scan-runasroot", true, "Default enable/disable RunAsRoot scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsScanRunAsPrivileged, "is-scan-run-as-privileged", true, "Default enable/disable RunAsPrivileged scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsPotentiallyRunningAsRoot, "is-scan-potentially-running-as-root", true, "Default enable/disable PotentiallyRunningAsRoot scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.IsPotentiallyRunningAsPrivileged, "is-scan-potentially-running-as-privileged", true, "Default enable/disable PotentiallyRunningAsPrivileged scan")
	c.PersistentFlags().Int64Var(&cfg.CollectorImage.ScanLifetimeMaxDays, "ScanLifetimeMaxDays", 120, "Default max days for (base) image lifetime scan")
	c.PersistentFlags().BoolVar(&cfg.CollectorImage.Skip, "skip", false, "Default behaviour for skipping scans for images")
	c.PersistentFlags().StringSliceVar(&cfg.CollectorImage.EngagementTags, "engagement-tags", []string{}, "Default engagement tags to use")
	c.PersistentFlags().StringVar(&cfg.CollectorImage.ContainerType, "container-type", "application", "Default container-type to use")
	c.PersistentFlags().StringVar(&cfg.CollectorImage.Team, "team", "", "Default team to use")
	c.PersistentFlags().StringVar(&cfg.CollectorImage.Product, "product", "", "Default product to use")
	c.PersistentFlags().StringVar(&cfg.CollectorImage.Slack, "slack", "", "Default slack channel to use")
	c.PersistentFlags().StringVar(&cfg.CollectorImage.Email, "email", "", "Default email to use")
	c.PersistentFlags().StringVar(&cfg.CollectorImage.NamespaceFilter, "namespace-filter", "", "Default namespace filter to use")
	c.PersistentFlags().StringVar(&cfg.CollectorImage.NamespaceFilterNegated, "negated_namespace_filter", "", "Default negated namespace filter to use")

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

	storage, err := storage.NewStorage(&cfg.StorageConfig, cfg.Environment)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Could not create storage for: " + cfg.StorageConfig.StorageFlag)
	}

	collectorDefaults := &cfg.CollectorImage
	annotationNames := &cfg.AnnotationNames
	runConfig := &cfg.RunConfig

	// Collect images from K8
	k8Images, err := k8client.GetAllImagesForAllNamespaces()
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Could not retrieve images from K8")
	}
	// debug log all retrieved images
	log.Debug().Interface("images", k8Images).Msg("")
	log.Info().Msg("Images retrieved from K8")

	// Convert & Clean k8 images to collector images
	images, err := collector.ConvertImages(k8Images, collectorDefaults, annotationNames, runConfig)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Could not collect images")
	}
	log.Debug().Interface("images", images).Msg("")
	log.Info().Msg("Images collected & converted")

	// Store images
	err = collector.Store(images, storage, collector.JsonIndentMarshal)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Could not store collected images")
	}
	log.Info().Msg("Images collected and stored")
	log.Debug().Interface("storage", storage).Msg("using storage")
}

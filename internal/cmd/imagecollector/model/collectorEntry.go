package model

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

type CollectorEntry struct {
	Environment                      string   `validate:"required" json:"environment"`
	Namespace                        string   `validate:"required" json:"namespace"`
	Image                            string   `validate:"required" json:"image"`
	ImageId                          string   `validate:"required" json:"image_id"`
	Team                             string   `validate:"ascii" json:"team" copier:"must"`
	Slack                            string   `json:"slack" copier:"must"`
	Rocketchat                       string   `json:"rocketchat"`
	Email                            string   `validate:"omitempty,email" json:"email"`
	AppKubernetesName                string   `validate:"ascii" json:"app_kubernetes_io_name"`
	AppKubernetesVersion             string   `validate:"ascii" json:"app_kubernetes_io_version"`
	ContainerType                    string   `validate:"oneof=application third-party,required" json:"container_type"`
	IsScanBaseimageLifetime          bool     `json:"is_scan_baseimage_lifetime"`
	IsScanDependencyCheck            bool     `json:"is_scan_dependency_check"`
	IsScanDependencyTrack            bool     `json:"is_scan_dependency_track"`
	IScanDistroless                  bool     `json:"is_scan_distroless"`
	IScanLifetime                    bool     `json:"is_scan_lifetime"`
	IScanMalware                     bool     `json:"is_scan_malware"`
	IsScanNewVersion                 bool     `json:"is_scan_new_version"`
	IsScanRunAsRoot                  bool     `json:"is_scan_runasroot"`
	IsPotentiallyRunningAsRoot       bool     `json:"is_potentially_running_as_root"`
	IsScanRunAsPrivileged            bool     `json:"is_scan_run_as_privileged"`
	IsPotentiallyRunningAsPrivileged bool     `json:"is_potentially_running_as_privileged"`
	ScanMaxDaysLifetime              int      `validate:"numeric" json:"scan_lifetime_max_days"`
	Skip                             bool     `json:"skip"`
	ScmSourceUrl                     bool     `json:"scm_source_url"`
	EngagementTags                   []string `json:"engagement_tags"`
}

func ValidateCollectorEntry(sl validator.StructLevel) {
	channelRegex := `^#[\w.\-]+$`
	entry := sl.Current().Interface().(CollectorEntry)
	validChannel := regexp.MustCompile(channelRegex)
	if entry.Slack != "" && !validChannel.MatchString(entry.Slack) {
		sl.ReportError(entry.Slack, "Slack", "Slack", "", channelRegex)
	}
	if entry.Rocketchat != "" && !validChannel.MatchString(entry.Rocketchat) {
		sl.ReportError(entry.Rocketchat, "Rocketchat", "Rocketchat", "", channelRegex)
	}
}

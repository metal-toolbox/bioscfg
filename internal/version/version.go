package version

import (
	"encoding/json"
	"runtime"
	rdebug "runtime/debug"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	GitCommit         string
	GitBranch         string
	GitSummary        string
	BuildDate         string
	AppVersion        string
	FleetdbAPIVersion = fleetdbAPIVersion()
	GoVersion         = runtime.Version()
)

type Version struct {
	GitCommit         string `json:"git_commit"`
	GitBranch         string `json:"git_branch"`
	GitSummary        string `json:"git_summary"`
	BuildDate         string `json:"build_date"`
	AppVersion        string `json:"app_version"`
	GoVersion         string `json:"go_version"`
	FleetdbAPIVersion string `json:"fleetdbapi_version"`
}

func (v *Version) AsLogFields() []any {
	return []any{
		"version", v.AppVersion,
		"commit", v.GitCommit,
		"branch", v.GitBranch,
	}
}

func Current() *Version {
	return &Version{
		GitBranch:         GitBranch,
		GitCommit:         GitCommit,
		GitSummary:        GitSummary,
		BuildDate:         BuildDate,
		AppVersion:        AppVersion,
		GoVersion:         GoVersion,
		FleetdbAPIVersion: FleetdbAPIVersion,
	}
}

func (v *Version) AsMap() (map[string]any, error) {
	var asMap map[string]interface{}

	bytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytes, &asMap)
	if err != nil {
		return nil, err
	}

	return asMap, nil
}

func ExportBuildInfoMetric() {
	buildInfo := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bioscfg_build_info",
			Help: "A metric with a constant '1' value, labeled by branch, commit, summary, builddate, version, Go version from which bioscfg was built.",
		},
		[]string{"branch", "commit", "summary", "builddate", "version", "goversion", "fleetdbapiVersion"},
	)

	buildInfo.WithLabelValues(GitBranch, GitCommit, GitSummary, BuildDate, AppVersion, GoVersion, FleetdbAPIVersion).Set(1)
}

func fleetdbAPIVersion() string {
	buildInfo, ok := rdebug.ReadBuildInfo()
	if !ok {
		return ""
	}

	for _, d := range buildInfo.Deps {
		if strings.Contains(d.Path, "fleetdb") {
			return d.Version
		}
	}

	return ""
}

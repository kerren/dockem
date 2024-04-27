package utils

type BuildDockerImageParams struct {
	Directory        string
	DockerBuildFlags []string
	DockerPassword   string
	DockerUsername   string
	DockerfilePath   string
	ImageName        string
	Latest           bool
	MainVersion      bool
	Registry         string
	Tag              []string
	VersionFile      string
	WatchDirectory   []string
	WatchFile        []string
}

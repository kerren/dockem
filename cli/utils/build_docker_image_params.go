package utils

type BuildDockerImageParams struct {
	Directory            string
	DockerPassword       string
	DockerUsername       string
	DockerfilePath       string
	IgnoreBuildDirectory bool
	ImageName            string
	Latest               bool
	MainVersion          bool
	Registry             string
	StrictRegistry       bool
	Tag                  []string
	VersionFile          string
	WatchDirectory       []string
	WatchFile            []string
}

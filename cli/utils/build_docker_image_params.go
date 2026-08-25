package utils

type BuildDockerImageParams struct {
	Directory            string
	DockerPassword       string
	DockerUsername       string
	DockerfilePath       string
	Exclude              []string
	IgnoreBuildDirectory bool
	IgnoreFile           string
	ImageName            string
	Latest               bool
	MainVersion          bool
	OutputFile           string
	OutputFormat         string
	Registry             string
	RespectDockerignore  bool
	StrictRegistry       bool
	Tag                  []string
	VersionFile          string
	WatchDirectory       []string
	WatchFile            []string
}

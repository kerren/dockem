package utils

type BuildDockerImageParams struct {
	WatchFile        []string
	WatchDirectory   []string
	Directory        string
	DockerBuildFlags []string
	DockerfilePath   string
	ImageName        string
	Latest           bool
	VersionFile      string
	Registry         string
	Tag              []string
	DockerUsername   string
	DockerPassword   string
}

func BuildDockerImage(params BuildDockerImageParams) {

}

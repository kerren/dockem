package utils

import (
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
)

func CreateRegclientClient(registry string, username string, password string, buildLog *BuildLog) *regclient.RegClient {
	host := config.Host{}
	customHost := false
	if registry != "" {
		host.Hostname = registry
		host.Name = registry
		customHost = true
		buildLog.dockerRegistry = registry
	}
	if username != "" {
		host.User = username
		host.RepoAuth = true
		customHost = true
		buildLog.dockerUsername = username
	}
	if password != "" {
		host.Pass = password
		host.RepoAuth = true
		customHost = true
		buildLog.dockerPassword = password
	}
	if customHost && registry == "" {
		host.Name = config.DockerRegistry
		host.Hostname = config.DockerRegistryDNS
		buildLog.dockerRegistry = config.DockerRegistry
	}
	buildLog.customHost = customHost

	var regclientOpts []regclient.Opt

	if customHost && password != "" {
		regclientOpts = []regclient.Opt{
			regclient.WithConfigHost(host),
		}
	} else if customHost && password == "" {
		regclientOpts = []regclient.Opt{
			regclient.WithConfigHost(host),
			regclient.WithDockerCreds(),
		}
	} else {
		regclientOpts = []regclient.Opt{
			regclient.WithDockerCreds(),
		}
	}

	return regclient.New(regclientOpts...)
}

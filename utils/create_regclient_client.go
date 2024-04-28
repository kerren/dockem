package utils

import (
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
)

func CreateRegclientClient(registry string, username string, password string) *regclient.RegClient {
	host := config.Host{}
	customHost := false
	if registry != "" {
		host.Hostname = registry
		host.Name = registry
		customHost = true
	}
	if username != "" {
		host.User = username
		host.RepoAuth = true
		customHost = true
	}
	if password != "" {
		host.Pass = password
		host.RepoAuth = true
		customHost = true
	}
	if customHost && registry == "" {
		host.Name = config.DockerRegistry
		host.Hostname = config.DockerRegistryDNS
	}

	var regclientOpts []regclient.Opt

	if customHost {
		regclientOpts = []regclient.Opt{
			regclient.WithConfigHost(host),
			regclient.WithDockerCreds(),
		}

	} else {
		regclientOpts = []regclient.Opt{}
	}

	return regclient.New(regclientOpts...)
}

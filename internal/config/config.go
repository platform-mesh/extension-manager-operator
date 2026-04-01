package config

import (
	"github.com/spf13/pflag"
)

type ServerConfig struct {
	ServerPort string
}

func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		ServerPort: "8088",
	}
}

func (c *ServerConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.ServerPort, "server-port", c.ServerPort, "Set the server port")
}

type OperatorConfig struct {
	KCPAPIExportEndpointSliceName          string
	SubroutinesContentConfigurationEnabled bool
}

func NewOperatorConfig() *OperatorConfig {
	return &OperatorConfig{
		KCPAPIExportEndpointSliceName:          "core.platform-mesh.io",
		SubroutinesContentConfigurationEnabled: true,
	}
}

func (c *OperatorConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.KCPAPIExportEndpointSliceName, "kcp-api-export-endpoint-slice-name", c.KCPAPIExportEndpointSliceName, "API export endpoint slice name for multicluster discovery")
	fs.BoolVar(&c.SubroutinesContentConfigurationEnabled, "subroutines-content-configuration-enabled", c.SubroutinesContentConfigurationEnabled, "Enable the content configuration subroutine")
}

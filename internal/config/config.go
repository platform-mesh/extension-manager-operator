package config

import (
	"github.com/spf13/pflag"
)

type ServerConfig struct {
	ServerPort                  string
	EntityTypeValidationEnabled bool
}

func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		ServerPort:                  "8088",
		EntityTypeValidationEnabled: false,
	}
}

func (c *ServerConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.ServerPort, "server-port", c.ServerPort, "Set the server port")
	fs.BoolVar(&c.EntityTypeValidationEnabled, "entity-type-validation-enabled", c.EntityTypeValidationEnabled, "Enable entityType validation for ContentConfigurations")
}

type OperatorConfig struct {
	KCPEnabled                             bool
	KCPAPIExportEndpointSliceName          string
	SubroutinesContentConfigurationEnabled bool
	EntityTypeValidationEnabled            bool
}

func NewOperatorConfig() *OperatorConfig {
	return &OperatorConfig{
		KCPEnabled:                             false,
		KCPAPIExportEndpointSliceName:          "core.platform-mesh.io",
		SubroutinesContentConfigurationEnabled: true,
		EntityTypeValidationEnabled:            false,
	}
}

func (c *OperatorConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.KCPEnabled, "kcp-enabled", c.KCPEnabled, "Enable KCP mode")
	fs.StringVar(&c.KCPAPIExportEndpointSliceName, "kcp-api-export-endpoint-slice-name", c.KCPAPIExportEndpointSliceName, "Set the KCP API export endpoint slice name")
	fs.BoolVar(&c.SubroutinesContentConfigurationEnabled, "subroutines-content-configuration-enabled", c.SubroutinesContentConfigurationEnabled, "Enable the content configuration subroutine")
	fs.BoolVar(&c.EntityTypeValidationEnabled, "entity-type-validation-enabled", c.EntityTypeValidationEnabled, "Enable entityType validation for ContentConfigurations")
}

package config

type ServerConfig struct {
	IsLocal    bool   `mapstructure:"is-local"`
	ServerPort string `mapstructure:"server-port"`
}

type OperatorConfig struct {
	KCP struct {
		Enabled                    bool   `mapstructure:"kcp-enabled" default:"false"`
		APIExportEndpointSliceName string `mapstructure:"kcp-api-export-endpoint-slice-name" default:"core.platform-mesh.io"`
	} `mapstructure:",squash"`
	Subroutines struct {
		ContentConfiguration struct {
			Enabled bool `mapstructure:"subroutines-content-configuration-enabled" default:"true"`
		} `mapstructure:",squash"`
	} `mapstructure:",squash"`
}

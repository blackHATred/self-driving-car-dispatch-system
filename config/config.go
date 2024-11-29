package config

type ServerConfig struct {
	DatabaseUrl    string `mapstructure:"database_url"`
	DatabaseNumber int    `mapstructure:"database_number"`
	VehicleHost    string `mapstructure:"vehicle_host"`
	VehiclePort    int    `mapstructure:"vehicle_port"`
	DispatcherHost string `mapstructure:"dispatcher_host"`
	DispatcherPort int    `mapstructure:"dispatcher_port"`
	SecretKey      string
}

type AdminConfig struct {
	DatabaseUrl    string `mapstructure:"database_url"`
	DatabaseNumber int    `mapstructure:"database_number"`
	Addr           string `mapstructure:"address"`
	SecretKey      string
}

type ClientConfig struct {
	addr string `mapstructure:"address"`
}

func (c *ClientConfig) GetAddress() string {
	return c.addr
}

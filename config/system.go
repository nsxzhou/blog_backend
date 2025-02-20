package config

type System struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Env       string `mapstructure:"env"`
	StartTime string `mapstructure:"start_time"`
	MachineID int64  `mapstructure:"machine_id"`
}

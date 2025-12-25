package config

var GlobalConfig Config

/*
type Config struct {
	ConfigFile string `koanf:"config.file"`
	EnvPrefix  string `koanf:"env.prefix"`
	ESHost     string `koanf:"es.host"`
	ESPort     int    `koanf:"es.port"`
	KibanaHost string `koanf:"kibana.host"`
	KibanaPort int    `koanf:"kibana.port"`
}
*/

type Config struct {
	ConfigFile Conf   `koanf:"config" json:"config" yaml:"config"`
	EnvPrefix  Env    `koanf:"env" json:"env" yaml:"env"`
	Http       Http   `koanf:"http" json:"http" yaml:"http"`
	ES         ES     `koanf:"es" json:"es" yaml:"es"`
	Kibana     Kibana `koanf:"kibana" json:"kibana" yaml:"kibana"`
	DB         DB     `koanf:"db" json:"db" yaml:"db"`
	Redis      Redis  `koanf:"redis" json:"redis" yaml:"redis"`
	Cron       Cron   `koanf:"cron" json:"cron" yaml:"cron"`
}

type Conf struct {
	File string `koanf:"file" json:"file" yaml:"file"`
}

type Env struct {
	Prefix string `koanf:"prefix" json:"prefix" yaml:"prefix"`
}

type Http struct {
	Host        string `koanf:"host" yaml:"host" json:"host"`
	Port        int    `koanf:"port" yaml:"port" json:"port"`
	ReleaseMode bool   `koanf:"releaseMode" yaml:"releaseMode" json:"releaseMode"`
}

type ES struct {
	Host     string `koanf:"host" json:"host" yaml:"host"`
	Port     int    `koanf:"port" json:"port" yaml:"port"`
	Protocol string `koanf:"protocol" json:"protocol" yaml:"protocol"`
	Username string `koanf:"username" yaml:"username" json:"username"`
	Password string `koanf:"password" yaml:"password" json:"password"`
}

type Kibana struct {
	Host string `koanf:"host" json:"host" yaml:"host"`
	Port int    `koanf:"port" json:"port" yaml:"port"`
}

type DB struct {
	Host     string `koanf:"host" yaml:"host" json:"host"`
	Port     int    `koanf:"port" yaml:"port" json:"port"`
	Username string `koanf:"username" yaml:"username" json:"username"`
	Password string `koanf:"password" yaml:"password" json:"password"`
	Name     string `koanf:"name" yaml:"name" json:"name"`
}

type Redis struct {
	Host     string `koanf:"host" yaml:"host" json:"host"`
	Port     int    `koanf:"port" yaml:"port" json:"port"`
	Password string `koanf:"password" yaml:"password" json:"password"`
	DB       int    `koanf:"db" yaml:"db" json:"db"`
}

type Cron struct {
	Schedule string `koanf:"schedule" yaml:"schedule" json:"schedule"`
}

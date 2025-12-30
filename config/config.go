package config

import v1 "k8s.io/api/core/v1"

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
	Kube       Kube   `koanf:"kube" json:"kube" yaml:"kube"`
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
	Host           string            `koanf:"host" json:"host" yaml:"host"`
	Port           int               `koanf:"port" json:"port" yaml:"port"`
	Protocol       string            `koanf:"protocol" json:"protocol" yaml:"protocol"`
	Username       string            `koanf:"username" yaml:"username" json:"username"`
	Password       string            `koanf:"password" yaml:"password" json:"password"`
	RestoreKey     string            `koanf:"restorekey" yaml:"restore_key" json:"restore_key"`
	RestoreCount   int32             `koanf:"restorecount" yaml:"restore_count" json:"restore_count"`
	Name           string            `koanf:"name" yaml:"name" json:"name"`
	Namespace      string            `koanf:"namespace" yaml:"namespace" json:"namespace"`
	ServiceAccount string            `koanf:"serviceaccount" yaml:"service_account" json:"service_account"`
	Plugins        []string          `koanf:"plugins" yaml:"plugins" json:"plugins"`
	LimitCPU       string            `koanf:"limitcpu" yaml:"limit_cpu" json:"limit_cpu"`
	LimitMem       string            `koanf:"limitmem" yaml:"limit_mem" json:"limit_mem"`
	RequestCPU     string            `koanf:"requestcpu" yaml:"request_cpu" json:"request_cpu"`
	RequestMem     string            `koanf:"requestmem" yaml:"request_mem" json:"request_mem"`
	StorageClass   string            `koanf:"storageclass" yaml:"storage_class" json:"storage_class"`
	Labels         map[string]string `koanf:"labels" yaml:"labels" json:"labels"`
	Annotations    map[string]string `koanf:"annotations" yaml:"annotations" json:"annotations"`
	NodeAffinity   v1.NodeAffinity   `koanf:"nodeaffinity" yaml:"nodeAffinity" json:"nodeAffinity"`
	Tolerations    map[string]string `koanf:"tolerations" yaml:"tolerations" json:"tolerations"`
	ContainerName  string            `koanf:"containername" yaml:"container_name" json:"container_name"`
	TopologyKey    string            `koanf:"topologykey" yaml:"topology_key" json:"topology_key"`
	DiskMinSize    float64           `koanf:"diskminsize" yaml:"disk_min_size" json:"disk_min_size"`
	RandomLen      int               `koanf:"randomlen" yaml:"random_len" json:"random_len"`
	Concurrency    int               `koanf:"concurrency" yaml:"concurrency" json:"concurrency"`
	MaxTasks       int               `koanf:"maxtasks" yaml:"max_tasks" json:"max_tasks"`
	Timeout        int               `koanf:"timeout" yaml:"timeout" json:"timeout"`
	Interval       int               `koanf:"interval" yaml:"interval" json:"interval"`
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

type Kube struct {
	Config string `koanf:"config" yaml:"config" json:"config"`
}

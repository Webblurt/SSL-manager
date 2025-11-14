package models

type Config struct {
	CertDir     string   `yaml:"cert_dir"`
	ReloadNginx bool     `yaml:"reload_nginx"`
	ReloadCmd   string   `yaml:"reload_cmd"`
	Domains     []string `yaml:"domains"`
	Email       string   `yaml:"email"`
	Auth        struct {
		AccessSecKey  string `yaml:"access_sec_key"`
		RefreshSecKey string `yaml:"refresh_sec_key"`
	} `yaml:"auth"`
}

package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func readConfigFromFile(configPath string) (*Config, error) {
	mkdir()
	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	// Unmarshal config file
	var config Config
	err = yaml.Unmarshal(data, &config)
	// Return config
	return &config, err
}

func mkdir() {
	os.MkdirAll("service", 0755)
	os.MkdirAll("log", 0755)
	os.MkdirAll("binary", 0755)
}

type BasicAuth struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

type Config struct {
	Accounts []*BasicAuth `yaml:"accounts" json:"accounts"`
	Port     int          `yaml:"port" json:"port"`
	Services []*Service   `yaml:"services" json:"services"`
	Mode     Mode         `yaml:"mode" json:"mode"`
}

type Service struct {
	Name string `yaml:"name" json:"name"`
	Tag  string `yaml:"tag" json:"tag"`
	Exec string `yaml:"exec" json:"exec"`

	Env  []string `yaml:"env" json:"env"`
	Args []string `yaml:"args" json:"args"`

	Running bool `yaml:"-" json:"running"`

	runner *Runner `yaml:"-" json:"-"`
}

func (s *Service) MarshalJSON() ([]byte, error) {
	s.runner.checkStat()
	pid := 0
	if s.runner.process != nil {
		pid = int(s.runner.process.Pid)
	}
	swp := ServiceWithProcess{
		Name:        s.Name,
		Tag:         s.Tag,
		Exec:        s.Exec,
		Running:     s.Running,
		Mem:         s.runner.mem,
		CPU:         s.runner.cpu,
		Connections: s.runner.connections,
		PID:         pid,
	}
	return json.Marshal(swp)
}

type ServiceWithProcess struct {
	PID         int      `json:"pid"`
	Tag         string   `json:"tag"`
	Name        string   `json:"name"`
	Exec        string   `json:"exec"`
	Running     bool     `json:"running"`
	Mem         int      `json:"mem"`
	CPU         float64  `json:"cpu"`
	Connections []string `json:"connections"`
}

func (s *Service) ExecPath() string {
	return filepath.Join("service", s.Exec)
}

func GenerateDefault() *Config {
	s := Service{
		Name: "test",
		Exec: "echo",
		Tag:  "default",
		Env:  []string{"TEST=1", "TEST=2"},
		Args: []string{"-port=1", "-conf=local.config.yaml"},
	}

	return &Config{
		Accounts: []*BasicAuth{{Username: "admin", Password: "admin"}},
		Port:     7798,
		Services: []*Service{&s},
		Mode:     Staging,
	}
}

func (c Config) AccountMap() map[string]string {
	m := make(map[string]string)
	for _, a := range c.Accounts {
		m[a.Username] = a.Password
	}
	return m
}

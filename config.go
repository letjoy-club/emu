package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/samber/lo"
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
	if err != nil {
		return nil, err
	}

	if config.MetaVars == nil {
		config.MetaVars = map[string]string{}
	}

	re := regexp.MustCompile(`@(\S+)`)
	for _, s := range config.Services {
		s.Env = lo.Map(s.Env, func(env string, i int) string {
			for key, value := range config.MetaVars {
				env = strings.ReplaceAll(env, key, value)
			}
			return env
		})

		if s.Args != nil {
			s.Args = lo.Map(s.Args, func(arg string, i int) string {
				results := re.FindAllString(arg, -1)
				if len(results) >= 1 {
					for _, result := range results {
						err := ReplaceMetaFile(result, config.MetaVars)
						if err != nil {
							continue
						}
					}
				}
				return arg
			})
		}
	}

	return &config, err
}

func ReplaceMetaFile(file string, metaVars map[string]string) error {
	filePath := strings.TrimPrefix(file, "@")
	data, err := os.ReadFile(path.Join("service", filePath))
	if err != nil {
		return err
	}
	for key, value := range metaVars {
		data = bytes.ReplaceAll(data, []byte(key), []byte(value))
	}
	os.WriteFile(path.Join("service", "@"+filePath), data, 0644)
	return nil
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
	Name     string       `yaml:"name" json:"name"`
	Accounts []*BasicAuth `yaml:"accounts" json:"accounts"`
	Port     int          `yaml:"port" json:"port"`
	Services []*Service   `yaml:"services" json:"services"`
	Mode     Mode         `yaml:"mode" json:"mode"`

	MetaVars map[string]string `yaml:"meta-variables" json:"metaVars"`
}

type Service struct {
	Name    string `yaml:"name" json:"name"`
	Tag     string `yaml:"tag" json:"tag"`
	Exec    string `yaml:"exec" json:"exec"`
	Running bool   `yaml:"-" json:"running"`
	Folder  string `yaml:"folder" json:"folder"`

	Env  []string `yaml:"env" json:"env"`
	Args []string `yaml:"args" json:"args"`

	runner *Runner `yaml:"-" json:"-"`
}

func (s *Service) MarshalJSON() ([]byte, error) {
	s.runner.checkStat()
	pid := 0
	if s.runner.process != nil {
		pid = int(s.runner.process.Pid)
	}
	swp := ServiceWithProcess{
		Name:    s.Name,
		Tag:     s.Tag,
		Exec:    s.Exec,
		Running: s.Running,
		Mem:     s.runner.mem,
		CPU:     s.runner.cpu,
		FDNum:   s.runner.fdNum,
		PID:     pid,

		Connections: s.runner.connections,
		Paths:       s.runner.paths,
	}
	return json.Marshal(swp)
}

type ServiceWithProcess struct {
	PID     int     `json:"pid"`
	Tag     string  `json:"tag"`
	Name    string  `json:"name"`
	Exec    string  `json:"exec"`
	Running bool    `json:"running"`
	Mem     int     `json:"mem"`
	CPU     float64 `json:"cpu"`
	FDNum   int     `json:"fdNum"`

	Connections []string `json:"connections"`
	Paths       []string `json:"paths"`
}

func (s *Service) Packed() bool {
	return s.Folder != ""
}

func (s *Service) ServiceFolder() string {
	return filepath.Join("service", s.Folder)
}

func (s *Service) ExecPath() string {
	if s.Packed() {
		return filepath.Join("service", s.Folder, s.Exec)
	}
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
		Name:     "deploy",
		MetaVars: map[string]string{},
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

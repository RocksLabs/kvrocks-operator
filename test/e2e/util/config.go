package util

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

const (
	DefaultClusterName   = "e2e-test"
	DefaultNamespace     = "kvrocks"
	DefaultKruiseVersion = "1.4.0"
	DefaultMinikubeShell = "start_minikube_cluster.sh"
)

type Config struct {
	kruiseVersion string `yaml:"kruiseVersion"`
	clusterName   string `yaml:"clusterName"`
	namespace     string `yaml:"namespace"`
	kubeConfig    string `yaml:"kubeConfig"`
}

func NewConfig(configFilePath string) (*Config, error) {
	config := &Config{}

	configData, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(configData, config)
	if err != nil {
		return nil, err
	}

	if config.clusterName == "" {
		config.clusterName = DefaultClusterName
	}
	if config.kruiseVersion == "" {
		config.kruiseVersion = DefaultKruiseVersion
	}
	if config.namespace == "" {
		config.namespace = DefaultNamespace
	}
	return config, nil
}

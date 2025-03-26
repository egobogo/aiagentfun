package filesys

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"

	"github.com/egobogo/aiagents/internal/config"
)

// FilesysConfigProvider is a concrete implementation of ConfigProvider that reads YAML config files.
type FilesysConfigProvider struct {
	cfg *config.Config
}

// NewFilesysConfigProvider creates a new FilesysConfigProvider and loads the configuration from the given path.
func NewFilesysConfigProvider(path string) (*FilesysConfigProvider, error) {
	prov := &FilesysConfigProvider{}
	// Load config during initialization.
	cfg, err := prov.LoadConfig(path)
	if err != nil {
		return nil, err
	}
	prov.cfg = cfg
	return prov, nil
}

// LoadConfig reads and unmarshals the YAML configuration file into a Config struct.
func (f *FilesysConfigProvider) LoadConfig(path string) (*config.Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML config: %w", err)
	}
	return &cfg, nil
}

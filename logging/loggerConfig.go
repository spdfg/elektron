package logging

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type LoggerConfig struct {
	SchedTraceConfig struct {
		Enabled           bool   `yaml:"enabled"`
		FilenameExtension string `yaml:"filenameExtension"`
		AllowOnConsole    bool   `yaml:"allowOnConsole"`
	} `yaml:"schedTrace"`

	PCPConfig struct {
		Enabled           bool   `yaml:"enabled"`
		FilenameExtension string `yaml:"filenameExtension"`
		AllowOnConsole    bool   `yaml:"allowOnConsole"`
	} `yaml:"pcp"`

	ConsoleConfig struct {
		Enabled           bool   `yaml:"enabled"`
		FilenameExtension string `yaml:"filenameExtension"`
		MinLogLevel       string `yaml:"minLogLevel"`
		AllowOnConsole    bool   `yaml:"allowOnConsole"`
	} `yaml:"console"`

	SPSConfig struct {
		Enabled           bool   `yaml:"enabled"`
		FilenameExtension string `yaml:"filenameExtension"`
		AllowOnConsole    bool   `yaml:"allowOnConsole"`
	} `yaml:"sps"`

	TaskDistrConfig struct {
		Enabled           bool   `yaml:"enabled"`
		FilenameExtension string `yaml:"filenameExtension"`
		AllowOnConsole    bool   `yaml:"allowOnConsole"`
	} `yaml:"clsfnTaskDistrOverhead"`

	SchedWindowConfig struct {
		Enabled           bool   `yaml:"enabled"`
		FilenameExtension string `yaml:"filenameExtension"`
		AllowOnConsole    bool   `yaml:"allowOnConsole"`
	} `yaml:"schedWindow"`

	Format []string `yaml:"format"`
}

func GetConfig(logConfigFilename string) (*LoggerConfig, error) {

	yamlFile, err := ioutil.ReadFile(logConfigFilename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read log config file")
	}

	c := &LoggerConfig{}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Error in unmarshalling yaml: %v", err)
	}

	return c, nil
}

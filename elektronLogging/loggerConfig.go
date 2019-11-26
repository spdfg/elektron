package elektronLogging

import (
	log "github.com/sirupsen/logrus"
	elekEnv "github.com/spdfg/elektron/environment"
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

func (c *LoggerConfig) GetConfig() *LoggerConfig {

	yamlFile, err := ioutil.ReadFile(elekEnv.LogConfigYaml)
	if err != nil {
		log.Printf("Error in reading yaml file   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Error in unmarshalling yaml: %v", err)
	}

	return c
}

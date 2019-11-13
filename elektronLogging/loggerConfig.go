package elektronLogging
import (
    "gopkg.in/yaml.v2"
    "io/ioutil"
    log "github.com/sirupsen/logrus"
    elekEnv "github.com/spdfg/elektron/environment"
)

type LoggerConfig struct {
	
	SchedTraceConfig struct {
		Enabled bool `yaml:"enabled"`
		FilenameExtension string `yaml:"filenameExtension"`
		EnableColumnHeaders bool `yaml:"enableColumnHeaders"`
        AllowOnConsole bool `yaml:"allowOnConsole"`
	} `yaml:"schedTrace"`

	PCPConfig struct {
		Enabled bool `yaml:"enabled"`
		FilenameExtension string `yaml:"filenameExtension"`
		EnableColumnHeaders bool `yaml:"enableColumnHeaders"`
        AllowOnConsole bool `yaml:"allowOnConsole"`
	} `yaml:"pcp"`

	ConsoleConfig struct {
		Enabled bool `yaml:"enabled"`
		FilenameExtension string `yaml:"filenameExtension"`
		EnableColumnHeaders bool `yaml:"enableColumnHeaders"`
		MinLogLevel string `yaml:"minLogLevel"`
	} `yaml:"console"`

	SPSConfig struct {
		Enabled bool `yaml:"enabled"`
		FilenameExtension string `yaml:"filenameExtension"`
		EnableColumnHeaders bool `yaml:"enableColumnHeaders"`
        AllowOnConsole bool `yaml:"allowOnConsole"`
	} `yaml:"sps"`

	TaskDistConfig struct {
		Enabled bool `yaml:"enabled"`
		FilenameExtension string `yaml:"filenameExtension"`
		EnableColumnHeaders bool `yaml:"enableColumnHeaders"`
        AllowOnConsole bool `yaml:"allowOnConsole"`
	} `yaml:"clsfnTaskDistOverhead"`
	
	SchedWindowConfig struct {
		Enabled bool `yaml:"enabled"`
		FilenameExtension string `yaml:"filenameExtension"`
		EnableColumnHeaders bool `yaml:"enableColumnHeaders"`
        AllowOnConsole bool `yaml:"allowOnConsole"`
	} `yaml:"schedWindow"`
	
	Format[] string `yaml:"format"`
}

func (c *LoggerConfig) GetConfig() *LoggerConfig {

    yamlFile, err := ioutil.ReadFile(elekEnv.LogConfigYaml)
    if err != nil {
        log.Printf("yamlFile.Get err   #%v ", err)
    }
    err = yaml.Unmarshal(yamlFile, c)
    if err != nil {
        log.Fatalf("Unmarshal: %v", err)
    }

    return c
}

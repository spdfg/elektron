// Copyright (C) 2018 spdfg
//
// This file is part of Elektron.
//
// Elektron is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Elektron is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Elektron.  If not, see <http://www.gnu.org/licenses/>.
//

package logging

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type loggerConfig struct {
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

func getConfig(logConfigFilename string) (*loggerConfig, error) {

	yamlFile, err := ioutil.ReadFile(logConfigFilename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read log config file")
	}

	c := &loggerConfig{}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Error in unmarshalling yaml: %v", err)
	}

	return c, nil
}

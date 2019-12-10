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
	"bytes"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

type elektronFormatter struct {
	TimestampFormat string
}

func (f elektronFormatter) Format(entry *log.Entry) ([]byte, error) {
	var b *bytes.Buffer

	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	level := fmt.Sprintf("[%s]:", strings.ToUpper(entry.Level.String()))
	message := strings.Join([]string{level, entry.Time.Format(f.TimestampFormat), entry.Message, " "}, " ")

	var formattedFields []string
	for key, value := range entry.Data {
		formattedFields = append(formattedFields, strings.Join([]string{key, value.(string)}, "="))
	}

	b.WriteString(message)
	b.WriteString(strings.Join(formattedFields, ", "))
	b.WriteByte('\n')
	return b.Bytes(), nil
}

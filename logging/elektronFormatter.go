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

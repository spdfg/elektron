package elektronLogging

import (
	"bytes"
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"strings"
)

type ElektronFormatter struct {
	TimestampFormat string
}

func (f ElektronFormatter) getColor(entry *log.Entry) *color.Color {
	switch entry.Level {
	case log.InfoLevel:
		return color.New(color.FgGreen, color.Bold)
	case log.WarnLevel:
		return color.New(color.FgYellow, color.Bold)
	case log.ErrorLevel:
		return color.New(color.FgRed, color.Bold)
	case log.FatalLevel:
		return color.New(color.FgRed, color.Bold)
	default:
		return color.New(color.FgWhite, color.Bold)
	}
}
func (f ElektronFormatter) Format(entry *log.Entry) ([]byte, error) {
	var b *bytes.Buffer

	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	levelColor := f.getColor(entry)
	level := levelColor.Sprintf("[%s]:", strings.ToUpper(entry.Level.String()))
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

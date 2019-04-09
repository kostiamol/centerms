package cfg

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

// NewLog initializes and returns a new instance of log.
func NewLog(appID, logLevel string) (*logrus.Logger, error) {
	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return nil, fmt.Errorf("ParseLevel(): ")
	}

	//hostname, err := os.Hostname()
	//if err != nil {
	//	return nil, fmt.Errorf("hostname(): %s")
	//}

	log := &logrus.Logger{
		Level: lvl,
		Out:   os.Stdout,
		Formatter: &formatter{
			Formatter: &logrus.TextFormatter{},
			//defaultFields: logrus.Fields{
			//	"host":    hostname,
			//	"service": appID,
			//},
		},
	}
	return log, nil
}

type formatter struct {
	logrus.Formatter
	defaultFields logrus.Fields
}

func (f *formatter) Format(entry *logrus.Entry) ([]byte, error) {
	for k, v := range f.defaultFields {
		entry.Data[k] = v
	}
	return f.Formatter.Format(entry)
}

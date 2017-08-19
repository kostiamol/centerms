package sys

import (
	log "github.com/Sirupsen/logrus"
)

func CheckError(desc string, err error) error {
	if err != nil {
		log.Errorln(desc, err)

		return err
	}
	return nil
}



package entities

import (
	log "github.com/Sirupsen/logrus"
	"errors"
)

func ValidateMAC(mac interface{}) bool {
	switch v := mac.(type) {
	case string:
		switch len(v) {
		case 17:
			return true
		default:
			log.Error("MAC should contain 17 symbols")
			return false
		}
	default:
		log.Error("MAC should be in a string format")
		return false
	}
}

func ValidateSendFreq(sendFreq interface{}) bool {
	switch v := sendFreq.(type) {
	case int64:
		switch {
		case v > 150:
			return true
		default:
			log.Error("Send Frequency should be more than 150!")
			return false
		}
	default:
		log.Error("Send Frequency should be in int64 format")
		return false
	}
}

func ValidateCollectFreq(collectFreq interface{}) bool {
	switch v := collectFreq.(type) {
	case int64:
		switch {
		case v > 150:

			return true
		default:
			log.Error("Collect Frequency should be more than 150!")
			return false
		}
	default:
		log.Error("Collect Frequency should be in int64 format")
		return false
	}
}

func ValidateTurnedOn(turnedOn interface{}) bool {
	switch turnedOn.(type) {
	case bool:
		return true
	default:
		log.Error("TurnedOn should be in bool format!")
		return false
	}
}

func ValidateStreamOn(streamOn interface{}) bool {
	switch streamOn.(type) {
	case bool:
		return true
	default:
		log.Error("StreamOn should be in bool format!")
		return false
	}
}

func ValidateDevMeta(meta DevMeta) (bool, error) {
	if !ValidateMAC(meta.MAC) {
		log.Error("Invalid MAC")
		return false, errors.New("Invalid MAC. ")
	}
	return true, nil
}

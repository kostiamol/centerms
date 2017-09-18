package entities

import (
	"errors"
)

func ValidateType(t string) bool {
	switch t {
	case "fridge":
		return true;
	case "washer":
		return true;
	default:
		return false;
	}
}

func ValidateMAC(mac string) error {
	if len(mac) != 17 {
		return errors.New("mac address should contain 17 symbols")
	}
	return nil
}

func ValidateSendFreq(sf int64) error {
	if sf > 150 {
		return errors.New("send frequency should be more than 150")
	}
	return nil
}

func ValidateCollectFreq(cf int64) error {
	if cf > 150 {
		return errors.New("collect frequency should be more than 150")

	}
	return nil
}

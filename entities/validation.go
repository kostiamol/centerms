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

func ValidateMAC(mac string) bool {
	if len(mac) != 17 {
		errors.New("mac address should contain 17 symbols")
		return false
	}
	return true
}

func ValidateSendFreq(sf int64) bool {
	if sf > 150 {
		errors.New("send frequency should be more than 150")
		return false
	}
	return true
}

func ValidateCollectFreq(cf int64) bool {
	if cf > 150 {
		errors.New("collect frequency should be more than 150")
		return false
	}
	return true
}
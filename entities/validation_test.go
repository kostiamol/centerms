package entities

import (
	"github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestValidateMAC(t *testing.T) {
	convey.Convey("ValidateMAC. MAC = 00-00-00-00-00-00", t, func() {
		actual := ValidateMAC("00-00-00-00-00-00")
		convey.So(actual, convey.ShouldBeTrue)
	})
	convey.Convey("ValidateMAC. MAC = 00-00-00-00-00-00", t, func() {
		actual := ValidateMAC("00-00-00-00-00-")
		convey.So(actual, convey.ShouldBeFalse)
	})
	convey.Convey("ValidateMAC. MAC = 12345678912345678", t, func() {
		actual := ValidateMAC("12345678912345678")
		convey.So(actual, convey.ShouldBeFalse)
	})

	convey.Convey("ValidateMAC. MAC = ", t, func() {
		actual := ValidateMAC("")
		convey.So(actual, convey.ShouldBeFalse)
	})
	convey.Convey("ValidateMAC. MAC = -1-2-3-4-5-6-7-8-9-10-11-12-13", t, func() {
		actual := ValidateMAC("-1-2-3-4-5-6-7-8-9-10-11-12-13")
		convey.So(actual, convey.ShouldBeFalse)
	})
}

func TestValidateSendFreq(t *testing.T) {
	convey.Convey("ValidateSendFreq. value = 9223372036854775807", t, func() {
		var test = int64(9223372036854775807)
		actual := ValidateSendFreq(test)
		convey.So(actual, convey.ShouldBeTrue)
	})
	convey.Convey("ValidateSendFreq. value = -9223372036854775808", t, func() {
		var test = int64(-9223372036854775808)
		actual := ValidateSendFreq(test)
		convey.So(actual, convey.ShouldBeFalse)
	})
	convey.Convey("ValidateSendFreq. value = 150", t, func() {
		var test = int64(150)
		actual := ValidateSendFreq(test)
		convey.So(actual, convey.ShouldBeFalse)
	})
}

func TestValidateCollectFreq(t *testing.T) {

	convey.Convey("ValidateCollectFreq. value = 9223372036854775807", t, func() {
		var test = int64(9223372036854775807)
		actual := ValidateCollectFreq(test)
		convey.So(actual, convey.ShouldBeTrue)
	})
	convey.Convey("ValidateCollectFreq. value = -9223372036854775808", t, func() {
		var test = int64(-9223372036854775808)
		actual := ValidateCollectFreq(test)
		convey.So(actual, convey.ShouldBeFalse)
	})
	convey.Convey("ValidateCollectFreq. value = 150", t, func() {
		var test = int64(150)
		actual := ValidateCollectFreq(test)
		convey.So(actual, convey.ShouldBeFalse)
	})
}

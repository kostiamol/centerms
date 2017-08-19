package entities

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)

func TestValidateMAC(t *testing.T) {

	Convey("ValidateMAC. MAC = 00-00-00-00-00-00", t, func() {
		actual := ValidateMAC("00-00-00-00-00-00")
		So(actual,ShouldBeTrue)
	})
	Convey("ValidateMAC. MAC = 00-00-00-00-00-00", t, func() {
		actual := ValidateMAC("00-00-00-00-00-")
		So(actual,ShouldBeFalse)
	})
	Convey("ValidateMAC. MAC = 12345678912345678", t, func() {
		actual := ValidateMAC("12345678912345678")
		So(actual,ShouldBeFalse)
	})

	Convey("ValidateMAC. MAC = ", t, func() {
		actual := ValidateMAC("")
		So(actual,ShouldBeFalse)
	})
	Convey("ValidateMAC. MAC = -1-2-3-4-5-6-7-8-9-10-11-12-13", t, func() {
		actual := ValidateMAC("-1-2-3-4-5-6-7-8-9-10-11-12-13")
		So(actual,ShouldBeFalse)
	})
}

func TestValidateSendFreq(t *testing.T) {

	Convey("ValidateSendFreq. value = 9223372036854775807", t, func() {
		var test interface{} = int64(9223372036854775807)
		actual := ValidateSendFreq(test)
		So(actual,ShouldBeTrue)
	})
	Convey("ValidateSendFreq. value = -9223372036854775808", t, func() {
		var test interface{} = int64(-9223372036854775808)
		actual := ValidateSendFreq(test)
		So(actual,ShouldBeFalse)
	})
	Convey("ValidateSendFreq. value = 150", t, func() {
		var test interface{} = int64(150)
		actual := ValidateSendFreq(test)
		So(actual,ShouldBeFalse)
	})
	Convey("ValidateSendFreq. value = abc", t, func() {
		var test interface{} = "abc"
		actual := ValidateSendFreq(test)
		So(actual,ShouldBeFalse)
	})
	Convey("ValidateSendFreq. value = interface{}", t, func() {
		var test interface{}
		actual := ValidateSendFreq(test)
		So(actual,ShouldBeFalse)
	})
}

func TestValidateCollectFreq(t *testing.T) {

	Convey("ValidateCollectFreq. value = 9223372036854775807", t, func() {
		var test interface{} = int64(9223372036854775807)
		actual := ValidateCollectFreq(test)
		So(actual,ShouldBeTrue)
	})
	Convey("ValidateCollectFreq. value = -9223372036854775808", t, func() {
		var test interface{} = int64(-9223372036854775808)
		actual := ValidateCollectFreq(test)
		So(actual,ShouldBeFalse)
	})
	Convey("ValidateCollectFreq. value = 150", t, func() {
		var test interface{} = int64(150)
		actual := ValidateCollectFreq(test)
		So(actual,ShouldBeFalse)
	})
	Convey("ValidateCollectFreq. value = abc", t, func() {
		var test interface{} = "abc"
		actual := ValidateCollectFreq(test)
		So(actual,ShouldBeFalse)
	})
	Convey("ValidateCollectFreq. value = interface{}", t, func() {
		var test interface{}
		actual := ValidateCollectFreq(test)
		So(actual,ShouldBeFalse)
	})
}

func TestValidateTurnedOn(t *testing.T) {

	Convey("ValidateTurnedOn. value = true", t, func() {
		var test interface{} = true
		actual := ValidateTurnedOn(test)
		So(actual,ShouldBeTrue)
	})
	Convey("ValidateTurnedOn. value = false", t, func() {
		var test interface{} = false
		actual := ValidateTurnedOn(test)
		So(actual,ShouldBeTrue)
	})
	Convey("ValidateTurnedOn. value = abc", t, func() {
		var test interface{} = "abc"
		actual := ValidateTurnedOn(test)
		So(actual,ShouldBeFalse)
	})
	Convey("ValidateTurnedOn. value = interface{}", t, func() {
		var test interface{}
		actual := ValidateTurnedOn(test)
		So(actual,ShouldBeFalse)
	})
}

func TestValidateStreamOn(t *testing.T) {

	Convey("ValidateStreamOn. value = true", t, func() {
		var test interface{} = true
		actual := ValidateStreamOn(test)
		So(actual,ShouldBeTrue)
	})
	Convey("ValidateStreamOn. value = false", t, func() {
		var test interface{} = false
		actual := ValidateStreamOn(test)
		So(actual,ShouldBeTrue)
	})
	Convey("ValidateStreamOn. value = abc", t, func() {
		var test interface{} = "abc"
		actual := ValidateStreamOn(test)
		So(actual,ShouldBeFalse)
	})
	Convey("ValidateStreamOn. value = interface{}", t, func() {
		var test interface{}
		actual := ValidateStreamOn(test)
		So(actual,ShouldBeFalse)
	})
}

func TestValidateDevMeta(t *testing.T) {

	Convey("ValidateMAC. MAC = 00-00-00-00-00-00", t, func() {
		meta := DevMeta{MAC:"00-00-00-00-00-00"}
		actual, _ := ValidateDevMeta(meta)
		So(actual,ShouldBeTrue)
	})
	Convey("ValidateMAC. MAC = 00-00-00-00-00-00", t, func() {
		meta := DevMeta{MAC:"00-00-00-00-00-"}
		actual, _ := ValidateDevMeta(meta)
		So(actual,ShouldBeFalse)
	})
	Convey("ValidateMAC. MAC = 12345678912345678", t, func() {
		meta := DevMeta{MAC:"12345678912345678"}
		actual, _ := ValidateDevMeta(meta)
		So(actual,ShouldBeFalse)
	})
	Convey("ValidateMAC. MAC = ", t, func() {
		meta := DevMeta{MAC:""}
		actual, _ := ValidateDevMeta(meta)
		So(actual,ShouldBeFalse)
	})
	Convey("ValidateMAC. MAC = -1-2-3-4-5-6-7-8-9-10-11-12-13", t, func() {
		meta := DevMeta{MAC:"-1-2-3-4-5-6-7-8-9-10-11-12-13"}
		actual, _ := ValidateDevMeta(meta)
		So(actual,ShouldBeFalse)
	})
}

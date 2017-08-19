package devices

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFloat32ToString(t *testing.T) {

	Convey("Float64ToString. Value = 23.3", t, func() {
		expected := "23.2"
		actual := Float64ToString(23.2)
		So(actual,ShouldEqual, expected)
	})
	Convey("Float64ToString. Value = -9 223 372 036 854 775 808 .0 ", t, func() {
		expected := "-9223372036854775808.0 "
		actual := Float64ToString(-9223372036854775808.0 )
		So(actual,ShouldEqual, expected)
	})
	Convey("Float64ToString. Value =  9 223 372 036 854 775 807.0", t, func() {
		expected := "9223372036854775807.0"
		actual := Float64ToString( 9223372036854775807.0)
		So(actual,ShouldEqual, expected)
	})
	Convey("Float64ToString. Value = 0", t, func() {
		expected := "0"
		actual := Float64ToString(0)
		So(actual,ShouldEqual, expected)
	})
}

func TestInt64ToString(t *testing.T) {

	Convey("Int64ToString. Value = 23", t, func() {
		expected := "23"
		actual := Int64ToString(23)
		So(actual,ShouldEqual, expected)
	})
	Convey("Int64ToString. Value = 9223372036854775807", t, func() {
		expected := "9223372036854775807"
		actual := Int64ToString(9223372036854775807)
		So(actual,ShouldEqual, expected)
	})
	Convey("Int64ToString. Value = -9223372036854775808", t, func() {
		expected := "-9223372036854775808"
		actual := Int64ToString(-9223372036854775808)
		So(actual,ShouldEqual, expected)
	})
	Convey("Int64ToString. Value = 0", t, func() {
		expected := "0"
		actual := Int64ToString(0)
		So(actual,ShouldEqual, expected)
	})
}
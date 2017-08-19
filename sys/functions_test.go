package sys

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
	"errors"
)

func TestCheckError(t *testing.T) {

	Convey("CheckError err = nil. Should return nil", t, func() {
	actual := CheckError("", nil)
		So(actual,ShouldBeNil)
	})

	Convey("CheckError err = not nil. Should return nil", t, func() {
		err := errors.New("Error")
		actual := CheckError("", err)
		So(actual,ShouldResemble, err)
	})
}


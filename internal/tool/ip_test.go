package tool

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetClientIp(t *testing.T) {
	Convey("Test client ip", t, func() {
		i := 0
		ip, _ := Ip.GetClientIp()
		Convey(`"check ."`, func() {
			i++
			So(
				len(strings.Split(ip, ".")),
				ShouldEqual,
				4,
			)
		})
	})
}

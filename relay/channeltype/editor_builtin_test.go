package channeltype

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBuiltinEditorTypesCoversAllChannelIDs(t *testing.T) {
	Convey("every channel type id below Dummy has builtin editor meta", t, func() {
		seen := map[int]bool{}
		for _, b := range BuiltinEditorTypes() {
			seen[b.TypeValue] = true
		}
		for id := 1; id < Dummy; id++ {
			if id == Custom {
				continue
			}
			So(seen[id], ShouldBeTrue)
		}
		So(len(BuiltinEditorTypes()), ShouldEqual, Dummy-2)
	})
}

// Code generated by "stringer -type GamcroAPI"; DO NOT EDIT.

package internal

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[TypeAPI-1]
	_ = x[TapAPI-2]
	_ = x[ClipPostAPI-4]
	_ = x[ClipGetAPI-8]
	_ = x[GamcroAPI_end-16]
}

const (
	_GamcroAPI_name_0 = "TypeAPITapAPI"
	_GamcroAPI_name_1 = "ClipPostAPI"
	_GamcroAPI_name_2 = "ClipGetAPI"
	_GamcroAPI_name_3 = "GamcroAPI_end"
)

var (
	_GamcroAPI_index_0 = [...]uint8{0, 7, 13}
)

func (i GamcroAPI) String() string {
	switch {
	case 1 <= i && i <= 2:
		i -= 1
		return _GamcroAPI_name_0[_GamcroAPI_index_0[i]:_GamcroAPI_index_0[i+1]]
	case i == 4:
		return _GamcroAPI_name_1
	case i == 8:
		return _GamcroAPI_name_2
	case i == 16:
		return _GamcroAPI_name_3
	default:
		return "GamcroAPI(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}

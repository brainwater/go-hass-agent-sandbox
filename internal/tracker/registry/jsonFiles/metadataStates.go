// Code generated by "stringer -type=state -output metadataStates.go -linecomment"; DO NOT EDIT.

package registry

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[disabledState-1]
	_ = x[registeredState-2]
}

const _state_name = "disabledregistered"

var _state_index = [...]uint8{0, 8, 18}

func (i state) String() string {
	i -= 1
	if i < 0 || i >= state(len(_state_index)-1) {
		return "state(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _state_name[_state_index[i]:_state_index[i+1]]
}

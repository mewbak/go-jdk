// Code generated by "stringer -type=InstKind -trimprefix=Inst"; DO NOT EDIT.

package ir

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[InstInvalid-0]
	_ = x[InstIload-1]
	_ = x[InstLload-2]
	_ = x[InstRet-3]
	_ = x[InstIret-4]
	_ = x[InstLret-5]
	_ = x[InstCallStatic-6]
	_ = x[InstCallGo-7]
	_ = x[InstIcmp-8]
	_ = x[InstLcmp-9]
	_ = x[InstJump-10]
	_ = x[InstJumpEqual-11]
	_ = x[InstJumpNotEqual-12]
	_ = x[InstJumpGtEq-13]
	_ = x[InstJumpGt-14]
	_ = x[InstJumpLt-15]
	_ = x[InstImul-16]
	_ = x[InstIadd-17]
	_ = x[InstLadd-18]
	_ = x[InstFadd-19]
	_ = x[InstIsub-20]
	_ = x[InstIneg-21]
	_ = x[InstLneg-22]
	_ = x[InstDadd-23]
	_ = x[InstConvL2I-24]
	_ = x[InstConvF2I-25]
	_ = x[InstConvD2I-26]
	_ = x[InstConvI2L-27]
}

const _InstKind_name = "InvalidIloadLloadRetIretLretCallStaticCallGoIcmpLcmpJumpJumpEqualJumpNotEqualJumpGtEqJumpGtJumpLtImulIaddLaddFaddIsubInegLnegDaddConvL2IConvF2IConvD2IConvI2L"

var _InstKind_index = [...]uint8{0, 7, 12, 17, 20, 24, 28, 38, 44, 48, 52, 56, 65, 77, 85, 91, 97, 101, 105, 109, 113, 117, 121, 125, 129, 136, 143, 150, 157}

func (i InstKind) String() string {
	if i < 0 || i >= InstKind(len(_InstKind_index)-1) {
		return "InstKind(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _InstKind_name[_InstKind_index[i]:_InstKind_index[i+1]]
}

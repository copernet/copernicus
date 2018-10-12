package errcode

import "fmt"

// RejectCode represents a numeric value by which a remote peer indicates
// why a message was rejected.
type RejectCode uint8

// These constants define the various supported reject codes.
const (
	RejectMalformed       RejectCode = 0x01
	RejectInvalid         RejectCode = 0x10
	RejectObsolete        RejectCode = 0x11
	RejectDuplicate       RejectCode = 0x12
	RejectNonstandard     RejectCode = 0x40
	RejectDust            RejectCode = 0x41
	RejectInsufficientFee RejectCode = 0x42
	RejectCheckpoint      RejectCode = 0x43
)

// Map of reject codes back strings for pretty printing.
var rejectCodeStrings = map[RejectCode]string{
	RejectMalformed:       "REJECT_MALFORMED",
	RejectInvalid:         "REJECT_INVALID",
	RejectObsolete:        "REJECT_OBSOLETE",
	RejectDuplicate:       "REJECT_DUPLICATE",
	RejectNonstandard:     "REJECT_NONSTANDARD",
	RejectDust:            "REJECT_DUST",
	RejectInsufficientFee: "REJECT_INSUFFICIENTFEE",
	RejectCheckpoint:      "REJECT_CHECKPOINT",
}

// String returns the RejectCode in human-readable form.
func (code RejectCode) String() string {
	if s, ok := rejectCodeStrings[code]; ok {
		return s
	}

	return fmt.Sprintf("Unknown RejectCode (%d)", uint8(code))
}

type InternalRejectCode int

const (
	RejectHighFee      InternalRejectCode = 0x100
	RejectAlreadyKnown InternalRejectCode = 0x101
	RejectConflict     InternalRejectCode = 0x102
)

// Map of reject codes back strings for pretty printing.
var internalRejects = map[InternalRejectCode]string{
	RejectHighFee:      "RejectHighFee",
	RejectAlreadyKnown: "RejectAlreadyKnown",
	RejectConflict:     "RejectConflict",
}

// String returns the RejectCode in human-readable form.
func (code InternalRejectCode) String() string {
	if s, ok := internalRejects[code]; ok {
		return s
	}

	return fmt.Sprintf("Unknown InternalRejectCode (%d)", uint8(code))
}

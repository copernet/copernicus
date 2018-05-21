package valistate

import "fmt"

const (
	RejectMalformed       byte = 0x01
	RejectInvalid              = 0x10
	RejectObsolete             = 0x11
	RejectDuplicate            = 0x12
	RejectNonStandard          = 0x40
	RejectDust                 = 0x41
	RejectInsufficientFee      = 0x42
	RejectCheckPoint           = 0x43
)

const (
	ModeValid   = iota // everything ok
	ModeInvalid        // network rule violation (DoS value may be set)
	ModeError          // run-time error
)

type ValidationState struct {
	mode               int
	dos                int
	rejectReason       string
	rejectCode         uint
	corruptionPossible bool
	debugMessage       string
}

func (vs *ValidationState) Dos(lvl int, ret bool, rejectCode uint, rejectReason string, corruption bool, dbgMsg string) bool {
	vs.rejectCode = rejectCode
	vs.rejectReason = rejectReason
	vs.corruptionPossible = corruption
	vs.debugMessage = dbgMsg
	if vs.mode == ModeError {
		return ret
	}
	vs.dos += lvl
	vs.mode = ModeInvalid
	return ret
}

func (vs *ValidationState) Invalid(ret bool, rejectCode uint, rejectReason string, dbgMsg string) bool {
	return vs.Dos(0, ret, rejectCode, rejectReason, false, dbgMsg)
}

func (vs *ValidationState) Error(rejectReason string) bool {
	if vs.mode == ModeValid {
		vs.rejectReason = rejectReason
	}
	vs.mode = ModeError
	return false
}

func (vs *ValidationState) IsValid() bool {
	return vs.mode == ModeValid
}

func (vs *ValidationState) IsInvalid() bool {
	return vs.mode == ModeInvalid
}

func (vs *ValidationState) IsError() bool {
	return vs.mode == ModeError
}

func (vs *ValidationState) IsInvalidDumpDos() (int, bool) {
	if vs.IsInvalid() {
		return vs.dos, true
	}
	return 0, false
}

func (vs *ValidationState) CorruptionPossible() bool {
	return vs.corruptionPossible
}

func (vs *ValidationState) SetCorruptionPossible() {
	vs.corruptionPossible = true
}

func (vs *ValidationState) RejectCode() uint {
	return vs.rejectCode
}

func (vs *ValidationState) RejectReason() string {
	return vs.rejectReason
}

func (vs *ValidationState) DebugMessage() string {
	return vs.debugMessage
}

func (vs *ValidationState) GetRejectCode() uint {
	return vs.rejectCode
}

func (vs *ValidationState) GetRejectReason() string {
	return vs.rejectReason
}

func (vs *ValidationState) GetDebugMessage() string {
	return vs.debugMessage
}

func (vs *ValidationState) FormatStateMessage() string {
	debug := ""
	if len(vs.GetDebugMessage()) != 0 {
		debug = ", " + vs.GetDebugMessage()
	}

	return fmt.Sprintf("%s%s (code %d)", vs.GetRejectReason(),
		debug, vs.GetRejectCode())
}

func NewValidationState() *ValidationState {
	v := new(ValidationState)
	v.mode = ModeValid
	v.dos = 0
	v.rejectCode = 0
	v.corruptionPossible = false
	return v
}

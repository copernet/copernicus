package model

const (
	REJECT_MALFORMED       byte = 0x01
	REJECT_INVALID              = 0x10
	REJECT_OBSOLETE             = 0x11
	REJECT_DUPLICATE            = 0x12
	REJECT_NONSTANDARD          = 0x40
	REJECT_DUST                 = 0x41
	REJECT_INSUFFICIENTFEE      = 0x42
	REJECT_CHECKPOINT           = 0x43
)

const (
	MODE_VALID   = iota // everything ok
	MODE_INVALID        // network rule violation (DoS value may be set)
	MODE_ERROR          // run-time error
)

type ValidationState struct {
	mode               int
	dos                int
	rejectReason       string
	rejectCode         byte
	corruptionPossible bool
	debugMessage       string
}

func (vs *ValidationState) Dos(lvl int, ret bool, rejectCode byte, rejectReason string, corruption bool, dbgMsg string) bool {
	vs.rejectCode = rejectCode
	vs.rejectReason = rejectReason
	vs.corruptionPossible = corruption
	vs.debugMessage = dbgMsg
	if vs.mode == MODE_ERROR {
		return ret
	}
	vs.dos += lvl
	vs.mode = MODE_INVALID
	return ret
}

func (vs *ValidationState) Invalid(ret bool, rejectCode byte, rejectReason string, dbgMsg string) bool {
	return vs.Dos(0, ret, rejectCode, rejectReason, false, dbgMsg)
}

func (vs *ValidationState) Error(rejectReason string) bool {
	if vs.mode == MODE_VALID {
		vs.rejectReason = rejectReason
	}
	vs.mode = MODE_ERROR
	return false
}

func (vs *ValidationState) IsValid() bool {
	return vs.mode == MODE_VALID
}

func (vs *ValidationState) IsInvalid() bool {
	return vs.mode == MODE_INVALID
}

func (vs *ValidationState) IsError() bool {
	return vs.mode == MODE_ERROR
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

func (vs *ValidationState) RejectCode() byte {
	return vs.rejectCode
}

func (vs *ValidationState) RejectReason() string {
	return vs.rejectReason
}

func (vs *ValidationState) DebugMessage() string {
	return vs.debugMessage
}

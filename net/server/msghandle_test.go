package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	pctx    = context.TODO()
	ctxTest context.Context
	clfunc  context.CancelFunc
)

func init() {
	ctxTest, clfunc = context.WithCancel(pctx)
	defer clfunc()
}

func TestSetMsgHandle(t *testing.T) {
	SetMsgHandle(ctxTest, s.MsgChan, s)
}

func TestValueFromAmount(t *testing.T) {
	amounts := valueFromAmount(1000)
	assert.Equal(t, amounts, float64(1e-05))

	amounts = valueFromAmount(100000)
	assert.Equal(t, amounts, float64(0.001))

	amounts = valueFromAmount(-1000)
	assert.Equal(t, amounts, float64(-1e-05))

	amounts = valueFromAmount(0)
	assert.Equal(t, amounts, float64(0))
}

func TestGetNetworkInfo(t *testing.T) {
	ret, err := GetNetworkInfo()
	if err != nil {
		t.Error(err.Error())
	}

	assert.Equal(t, ret.Version, 1000000)
	assert.Equal(t, ret.ProtocolVersion, uint32(70013))
	assert.Equal(t, ret.LocalRelay, true)
	assert.Equal(t, ret.NetworkActive, true)
}

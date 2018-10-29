package server

import (
	"context"
	"testing"
)

var pctx = context.TODO()
var ctxTest context.Context
var clfunc context.CancelFunc

func init() {
	ctxTest, clfunc = context.WithCancel(pctx)
}

func TestSetMsgHandle(t *testing.T) {
	SetMsgHandle(ctxTest, s.MsgChan, s)
}

package service

import (
	"sync"
	"context"
)

type MsgHandle struct {
	mtx sync.Mutex
	sendToPeerMag  <- chan interface{}
	recvChannel  chan interface{}
	errChannel	chan error
}

// NewMsgHandle create a msgHandle for these message from peer And RPC.
// Then begins the core block handler which processes block and inv messages.
func NewMsgHandle(ctx context.Context, cmdCh <- chan interface{}) *MsgHandle {
	msg := &MsgHandle{mtx:sync.Mutex{}, sendToPeerMag:cmdCh}
	ctxChild, _ := context.WithCancel(ctx)
	go msg.start(ctxChild)
	return msg
}

// start begins the core block handler which processes block and inv messages.
// It must be run as a goroutine.
func (msg *MsgHandle)start(ctx context.Context)  {
	out:
	for{
		select{
		case m := <-msg.recvChannel:
			go func(m interface{}) {
				 msg.sendToPeerMag <- m
			}(m)
		case <-ctx.Done():
			break out
		}
	}
}

type addrType struct {
	addr string
	port int
	result chan interface{}
}

// Peer And net caller
func (msg *MsgHandle)ProcessMsg(message interface{}) (ret Imsg, err error) {
	msg.recvChannel <- message

	m := message.(addrType)
	switch message.(type) {
	case addrType:
		select{
		case r := <- m.result:
			_ = r
			return r, nil
		case err := <- msg.errChannel:
			return nil, err
		}
	}
}

type PingStruct struct {
	Nonce int
	ret chan interface{}
}

type PingRspStruct1 struct {
	addr string
}

func RpcCall()  {
	msg := NewMsgHandle(make(chan interface{}))
	msg.ProcessForRpc(PingStruct{ret:make(chan interface{})})

}

// Rpc process things .
func (msg *MsgHandle)ProcessForRpc(message interface{}) (rsp Imsg, err error) {

	ret , err := msg.ProcessMsg(message)
	if err != nil{
		return nil, err
	}

	switch ret.(type) {
	case PingRspStruct1:
		msg.sendToPeerMag <- ret
	case SigalPeer:
		msg.sendToPeerMag <- ret
	default:
	}

	return ret, err
}
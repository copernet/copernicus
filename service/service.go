package service

import (
	"sync"
)

type MsgHandle struct {
	mtx sync.Mutex
	sendChannel  <- chan interface{}
	recvChannel  chan interface{}
	errChannel	chan error
}

func NewMsgHandle(s <- chan interface{}) *MsgHandle {
	return &MsgHandle{mtx:sync.Mutex{}, sendChannel:s}
}

func (msg *MsgHandle)Start()  {
	go msg.processmsg()
}

func (msg *MsgHandle)processmsg()  {
	for{
		select{
		case m := <-msg.recvChannel:
			go func(m interface{}) {
				 msg.sendChannel <- m
			}(m)
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
		msg.sendChannel <- ret
	case SigalPeer:
		msg.sendChannel <- ret
	default:
	}

	return ret, err
}
package sugar

import (
	"reflect"
	"sync"
)

type MsgType int

const (
	MsgTypeMsg MsgType = iota // this type of message should have message head
	MsgTypeCmd                // this type of message's message head can be empty, use \n as separator
)

type NetType int

const (
	NetTypeTCP NetType = iota //TCP
	NetTypeUDP                //UDP
)

type ConnType int

const (
	ConnTypeListen ConnType = iota //listen type
	ConnTypeConn                   //connected type
	ConnTypeAccept                 //accepted type
)

type IMsgQue interface {
	ID() uint32
	GetMsgType() MsgType
	GetConnType() ConnType
	GetNetType() NetType

	LocalAddr() string
	RemoteAddr() string

	Stop()
	IsStop() bool
	Available() bool

	Send(m *Message) (re bool)
	SendString(str string) (re bool)
	SendStringLn(str string) (re bool)
	SendByteStr(str []byte) (re bool)
	SendByteStrLn(str []byte) (re bool)
	SendCallback(m *Message, c chan *Message) (re bool)
	SetTimeout(t int)
	GetTimeout() int
	Reconnect(t int) //reconnect interval, unit: s, this function only can be invoked when connection was closed

	GetHandler() IMsgHandler

	SetUser(user interface{})
	GetUser() interface{}
	SetExtData(extData interface{})
	GetExtData() interface{}

	tryCallback(msg *Message) (re bool)
}

type msgQue struct {
	id uint32 //uniquely identify

	writeCh chan *Message //write channel
	stop    int32         //stop token
	msgTyp  MsgType       //message type
	connTyp ConnType

	handler       IMsgHandler
	parser        IParser
	parserFactory *Parser
	timeout       int

	init         bool
	available    bool
	callback     map[int]chan *Message
	user         interface{}
	extData      interface{}
	callbackLock sync.Mutex
}

func (r *msgQue) SetUser(user interface{}) {
	r.user = user
}

func (r *msgQue) Available() bool {
	return r.available
}

func (r *msgQue) GetUser() interface{} {
	return r.user
}

func (r *msgQue) SetExtData(extData interface{}) {
	r.extData = extData
}

func (r *msgQue) GetExtData() interface{} {
	return r.extData
}

func (r *msgQue) GetHandler() IMsgHandler {
	return r.handler
}

func (r *msgQue) GetMsgType() MsgType {
	return r.msgTyp
}

func (r *msgQue) GetConnType() ConnType {
	return r.connTyp
}

func (r *msgQue) ID() uint32 {
	return r.id
}

func (r *msgQue) SetTimeout(t int) {
	r.timeout = t
}

func (r *msgQue) GetTimeout() int {
	return r.timeout
}
func (r *msgQue) Reconnect(t int) {

}

func (r *msgQue) Send(m *Message) (re bool) {
	if m == nil || !r.available {
		return
	}
	defer func() {
		if err := recover(); err != nil {
			re = false
		}
	}()

	r.writeCh <- m
	return true
}

func (r *msgQue) SendCallback(m *Message, c chan *Message) (re bool) {
	if c == nil || cap(c) < 1 {
		Errorf("try send callback but chan is null or no buffer")
		return
	}
	if r.Send(m) {
		r.setCallback(m.Tag(), c)
	} else {
		c <- nil
		return
	}
	return true
}

func (r *msgQue) SendString(str string) (re bool) {
	return r.Send(&Message{Data: []byte(str)})
}

func (r *msgQue) SendStringLn(str string) (re bool) {
	return r.SendString(str + "\n")
}

func (r *msgQue) SendByteStr(str []byte) (re bool) {
	return r.SendString(string(str))
}

func (r *msgQue) SendByteStrLn(str []byte) (re bool) {
	return r.SendString(string(str) + "\n")
}

func (r *msgQue) tryCallback(msg *Message) (re bool) {
	if r.callback == nil {
		return false
	}
	defer func() {
		if err := recover(); err != nil {

		}
		r.callbackLock.Unlock()
	}()
	r.callbackLock.Lock()
	if r.callback != nil {
		tag := msg.Tag()
		if c, ok := r.callback[tag]; ok {
			delete(r.callback, tag)
			c <- msg
			re = true
		}
	}
	return
}

func (r *msgQue) setCallback(tag int, c chan *Message) {
	defer func() {
		if err := recover(); err != nil {

		}
		r.callback[tag] = c
		r.callbackLock.Unlock()
	}()

	r.callbackLock.Lock()
	if r.callback == nil {
		r.callback = make(map[int]chan *Message)
	}
	oc, ok := r.callback[tag]
	if ok { // channel might be closed already
		oc <- nil
	}
}

func (r *msgQue) BaseStop() {
	if r.writeCh != nil {
		close(r.writeCh)
	}

	for k, v := range r.callback {
		v <- nil
		delete(r.callback, k)
	}
	msgQueMapSync.Lock()
	delete(msgQueMap, r.id)
	msgQueMapSync.Unlock()
	Infof("msgQue close id:%d", r.id)
}

func (r *msgQue) processMsg(msgQue IMsgQue, msg *Message) bool {
	if r.parser != nil && msg.Data != nil {
		mp, err := r.parser.ParseC2S(msg)
		if err == nil {
			msg.IMsgParser = mp
		} else {
			if r.parser.GetErrType() == ParseErrTypeSendRemind {
				if msg.Head != nil {
					r.Send(r.parser.GetRemindMsg(err, r.msgTyp).CopyTag(msg))
				} else {
					r.Send(r.parser.GetRemindMsg(err, r.msgTyp))
				}
				return true
			} else if r.parser.GetErrType() == ParseErrTypeClose {
				return false
			} else if r.parser.GetErrType() == ParseErrTypeContinue {
				return true
			}
		}
	}
	f := r.handler.GetHandlerFunc(msgQue, msg)
	if f == nil {
		f = r.handler.OnProcessMsg
	}
	return f(msgQue, msg)
}

type HandlerFunc func(msgQue IMsgQue, msg *Message) bool

type IMsgHandler interface {
	OnNewMsgQue(msgQue IMsgQue) bool
	OnDelMsgQue(msgQue IMsgQue)
	OnProcessMsg(msgQue IMsgQue, msg *Message) bool //default message handler
	OnConnectComplete(msgQue IMsgQue, ok bool) bool
	GetHandlerFunc(msgQue IMsgQue, msg *Message) HandlerFunc
}

type IMsgRegister interface {
	Register(cmd, act uint8, fun HandlerFunc)
	RegisterMsg(v interface{}, fun HandlerFunc)
}

type DefMsgHandler struct {
	msgMap  map[int]HandlerFunc
	typeMap map[reflect.Type]HandlerFunc
}

func (r *DefMsgHandler) OnNewMsgQue(msgQue IMsgQue) bool                { return true }
func (r *DefMsgHandler) OnDelMsgQue(msgQue IMsgQue)                     {}
func (r *DefMsgHandler) OnProcessMsg(msgQue IMsgQue, msg *Message) bool { return true }
func (r *DefMsgHandler) OnConnectComplete(msgQue IMsgQue, ok bool) bool { return true }
func (r *DefMsgHandler) GetHandlerFunc(msgQue IMsgQue, msg *Message) HandlerFunc {
	if msgQue.tryCallback(msg) {
		return r.OnProcessMsg
	}

	if msg.CmdAct() == 0 {
		if r.typeMap != nil {
			if f, ok := r.typeMap[reflect.TypeOf(msg.C2S())]; ok {
				return f
			}
		}
	} else if r.msgMap != nil {
		if f, ok := r.msgMap[msg.CmdAct()]; ok {
			return f
		}
	}

	return nil
}

func (r *DefMsgHandler) RegisterMsg(v interface{}, fun HandlerFunc) {
	msgType := reflect.TypeOf(v)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		Fatal("message pointer required")
		return
	}
	if r.typeMap == nil {
		r.typeMap = map[reflect.Type]HandlerFunc{}
	}
	r.typeMap[msgType] = fun
}

func (r *DefMsgHandler) Register(cmd, act uint8, fun HandlerFunc) {
	if r.msgMap == nil {
		r.msgMap = map[int]HandlerFunc{}
	}
	r.msgMap[CmdAct(cmd, act)] = fun
}

type EchoMsgHandler struct {
	DefMsgHandler
}

func (r *EchoMsgHandler) OnProcessMsg(msgQue IMsgQue, msg *Message) bool {
	msgQue.Send(msg)
	return true
}

var msgQueID uint32 //message queue ID
var msgQueMapSync sync.Mutex
var msgQueMap = map[uint32]IMsgQue{}
var DefMsgQueTimeout = 180

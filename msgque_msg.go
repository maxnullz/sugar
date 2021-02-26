package sugar

import (
	"fmt"
	"unsafe"
)

const (
	MsgHeadSize = 12
)

const (
	FlagEncrypt  = 1 << 0 //encrypted data
	FlagCompress = 1 << 1 //compressed data
	FlagContinue = 1 << 2 //still have remain data need to be received
	FlagNeedAck  = 1 << 3 //message should be acknowledged
	FlagAck      = 1 << 4 //acknowledgement message
	FlagReSend   = 1 << 5 //message been re-sent
	FlagClient   = 1 << 6 //use this to tell the message source, internal server or published client
)

var MaxMsgDataSize uint32 = 1024 * 1024

type MessageHead struct {
	Len   uint32
	Error uint16
	Cmd   uint8
	Act   uint8
	Index uint16
	Flags uint16

	forever bool
	data    []byte
}

func (r *MessageHead) Bytes() []byte {
	if r.forever {
		return r.data
	}
	data := make([]byte, MsgHeadSize)
	phead := (*MessageHead)(unsafe.Pointer(&data[0]))
	phead.Len = r.Len
	phead.Error = r.Error
	phead.Cmd = r.Cmd
	phead.Act = r.Act
	phead.Index = r.Index
	phead.Flags = r.Flags
	return data
}

func (r *MessageHead) BytesWithData(wdata []byte) []byte {
	r.Len = uint32(len(wdata))
	data := make([]byte, MsgHeadSize+r.Len)
	phead := (*MessageHead)(unsafe.Pointer(&data[0]))
	phead.Len = r.Len
	phead.Error = r.Error
	phead.Cmd = r.Cmd
	phead.Act = r.Act
	phead.Index = r.Index
	phead.Flags = r.Flags
	if wdata != nil {
		copy(data[MsgHeadSize:], wdata)
	}
	return data
}

func (r *MessageHead) FromBytes(data []byte) error {
	if len(data) < MsgHeadSize {
		return ErrMsgLenTooShort
	}
	phead := (*MessageHead)(unsafe.Pointer(&data[0]))
	r.Len = phead.Len
	r.Error = phead.Error
	r.Cmd = phead.Cmd
	r.Act = phead.Act
	r.Index = phead.Index
	r.Flags = phead.Flags
	if r.Len > MaxMsgDataSize {
		return ErrMsgLenTooLong
	}
	return nil
}

func CmdAct(cmd, act uint8) int {
	return int(cmd)<<8 + int(act)
}

func Tag(cmd, act uint8, index uint16) int {
	return int(cmd)<<16 + int(act)<<8 + int(index)
}

func (r *MessageHead) CmdAct() int {
	return CmdAct(r.Cmd, r.Act)
}

func (r *MessageHead) Tag() int {
	return Tag(r.Cmd, r.Act, r.Index)
}

func (r *MessageHead) String() string {
	return fmt.Sprintf("Len:%v Error:%v Cmd:%v Act:%v Index:%v Flags:%v", r.Len, r.Error, r.Cmd, r.Act, r.Index, r.Flags)
}

func NewMessageHead(data []byte) *MessageHead {
	head := &MessageHead{}
	if err := head.FromBytes(data); err != nil {
		return nil
	}
	return head
}

func MessageHeadFromByte(data []byte) *MessageHead {
	if len(data) < MsgHeadSize {
		return nil
	}
	phead := new(*MessageHead)
	*phead = (*MessageHead)(unsafe.Pointer(&data[0]))
	if (*phead).Len > MaxMsgDataSize {
		return nil
	}
	return *phead
}

type Message struct {
	Head       *MessageHead //message head, can be empty
	Data       []byte       //message data
	IMsgParser              //message parser
	User       interface{}  //user self defined data
}

func (r *Message) CmdAct() int {
	if r.Head != nil {
		return CmdAct(r.Head.Cmd, r.Head.Act)
	}
	return 0
}

func (r *Message) Cmd() uint8 {
	if r.Head != nil {
		return r.Head.Cmd
	}
	return 0
}

func (r *Message) Act() uint8 {
	if r.Head != nil {
		return r.Head.Act
	}
	return 0
}

func (r *Message) Tag() int {
	if r.Head != nil {
		return Tag(r.Head.Cmd, r.Head.Act, r.Head.Index)
	}
	return 0
}

func (r *Message) Bytes() []byte {
	if r.Head != nil {
		if r.Data != nil {
			return r.Head.BytesWithData(r.Data)
		}
		return r.Head.Bytes()
	}
	return r.Data
}

func (r *Message) CopyTag(old *Message) *Message {
	r.Head.Cmd = old.Head.Cmd
	r.Head.Act = old.Head.Act
	r.Head.Index = old.Head.Index
	return r
}

func NewErrMsg(err error) *Message {
	errcode, ok := ErrIDMap[err]
	if !ok {
		errcode = ErrIDMap[ErrErrIDNotFound]
	}
	return &Message{
		Head: &MessageHead{
			Error: errcode,
		},
	}
}

func NewStrMsg(str string) *Message {
	return &Message{
		Data: []byte(str),
	}
}

func NewDataMsg(data []byte) *Message {
	return &Message{
		Head: &MessageHead{
			Len: uint32(len(data)),
		},
		Data: data,
	}
}

func NewMsg(cmd, act uint8, index, err uint16, data []byte) *Message {
	return &Message{
		Head: &MessageHead{
			Len:   uint32(len(data)),
			Error: err,
			Cmd:   cmd,
			Act:   act,
			Index: index,
		},
		Data: data,
	}
}

func NewForverMsg(cmd, act uint8, index, err uint16, data []byte) *Message {
	msg := &Message{
		Head: &MessageHead{
			Len:     uint32(len(data)),
			Error:   err,
			Cmd:     cmd,
			Act:     act,
			Index:   index,
			forever: true,
		},
		Data: data,
	}
	msg.Head.data = msg.Bytes()
	return msg
}

func NewTagMsg(cmd, act uint8, index uint16) *Message {
	return &Message{
		Head: &MessageHead{
			Cmd:   cmd,
			Act:   act,
			Index: index,
		},
	}
}

package sugar

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type udpMsgQue struct {
	msgQue
	conn     *net.UDPConn
	readCh   chan []byte // channel
	addr     *net.UDPAddr
	lastTick int64
	sync.Mutex
}

func (r *udpMsgQue) GetNetType() NetType {
	return NetTypeUDP
}

func (r *udpMsgQue) Stop() {
	if atomic.CompareAndSwapInt32(&r.stop, 0, 1) {
		Go(func() {
			if r.init {
				r.handler.OnDelMsgQue(r)
			}
			r.available = false
			if r.readCh != nil {
				close(r.readCh)
			}

			udpMapLock.Lock()
			delete(udpMap, r.addr.String())
			udpMapLock.Unlock()

			if IsStop() && len(udpMap) == 0 && r.conn != nil {
				r.conn.Close()
			}
			r.BaseStop()
		})
	}
}

func (r *udpMsgQue) IsStop() bool {
	if r.stop == 0 {
		if IsStop() {
			r.Stop()
		}
	}
	return r.stop == 1
}

func (r *udpMsgQue) LocalAddr() string {
	if r.conn != nil {
		return r.conn.LocalAddr().String()
	}
	return ""
}

func (r *udpMsgQue) RemoteAddr() string {
	if r.addr != nil {
		return r.addr.String()
	}
	return ""
}

func (r *udpMsgQue) read() {
	defer func() {
		if err := recover(); err != nil {
			Errorf("msgQue read panic id:%v err:%v", r.id, err.(error))
			LogStack()
		}
		r.Stop()
	}()
	var data []byte
	for !r.IsStop() {
		select {
		case data = <-r.readCh:
		}
		if data == nil {
			break
		}
		var msg *Message
		if r.msgTyp == MsgTypeCmd {
			msg = &Message{Data: data}
		} else {
			head := MessageHeadFromByte(data)
			if head == nil {
				break
			}
			if head.Len > 0 {
				msg = &Message{Head: head, Data: data[MsgHeadSize:]}
			} else {
				msg = &Message{Head: head}
			}
		}
		r.lastTick = Timestamp
		if !r.init {
			if !r.handler.OnNewMsgQue(r) {
				break
			}
			r.init = true
		}

		if !r.processMsg(r, msg) {
			break
		}
	}
}

func (r *udpMsgQue) write() {
	defer func() {
		if err := recover(); err != nil {
			Errorf("msgQue write panic id:%v err:%v", r.id, err.(error))
			LogStack()
		}
		r.Stop()
	}()

	timeoutCheck := false
	tick := time.NewTimer(time.Second * time.Duration(r.timeout))
	for !r.IsStop() {
		var m *Message
		select {
		case m = <-r.writeCh:
		case <-tick.C:
			left := int(Timestamp - r.lastTick)
			if left < r.timeout {
				timeoutCheck = true
				tick = time.NewTimer(time.Second * time.Duration(r.timeout-left))
			}
		}
		if timeoutCheck {
			timeoutCheck = false
			continue
		}
		if m == nil {
			break
		}

		if r.msgTyp == MsgTypeCmd {
			if m.Data != nil {
				r.conn.WriteToUDP(m.Data, r.addr)
			}
		} else {
			if m.Head != nil || m.Data != nil {
				r.conn.WriteToUDP(m.Bytes(), r.addr)
			}
		}

		r.lastTick = Timestamp
	}
}

func (r *udpMsgQue) sendRead(data []byte, n int) (re bool) {
	defer func() {
		if err := recover(); err != nil {
			re = false
		}
	}()

	re = true
	if len(r.readCh) < cap(r.readCh) {
		pData := make([]byte, n)
		copy(pData, data)
		r.readCh <- pData
	}
	return
}

var udpMap = map[string]*udpMsgQue{}
var udpMapLock sync.Mutex

func (r *udpMsgQue) listenTrue() {
	data := make([]byte, 1<<16)
	for !r.IsStop() {
		r.Lock()
		n, addr, err := r.conn.ReadFromUDP(data)
		r.Unlock()
		if err != nil {
			if err.(net.Error).Timeout() {
				continue
			}
			break
		}

		if n <= 0 {
			continue
		}

		udpMapLock.Lock()
		msgQue, ok := udpMap[addr.String()]
		if !ok {
			msgQue = newUDPAccept(r.conn, r.msgTyp, r.handler, r.parserFactory, addr)
			udpMap[addr.String()] = msgQue
		}
		udpMapLock.Unlock()

		if !msgQue.sendRead(data, n) {
			Errorf("drop msg because msgQue full msgqueid:%v", msgQue.id)
		}
	}
}

func (r *udpMsgQue) listen() {
	for i := 0; i < UDPServerGoCnt; i++ {
		Go(func() {
			r.listenTrue()
		})
	}
	r.listenTrue()
	r.Stop()
}

func newUDPAccept(conn *net.UDPConn, msgtyp MsgType, handler IMsgHandler, parser *Parser, addr *net.UDPAddr) *udpMsgQue {
	msgQue := udpMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgQueID, 1),
			writeCh:       make(chan *Message, 64),
			msgTyp:        msgtyp,
			handler:       handler,
			available:     true,
			timeout:       DefMsgQueTimeout,
			connTyp:       ConnTypeAccept,
			parserFactory: parser,
		},
		conn:     conn,
		readCh:   make(chan []byte, 64),
		addr:     addr,
		lastTick: Timestamp,
	}
	if parser != nil {
		msgQue.parser = parser.Get()
	}
	msgQueMapSync.Lock()
	msgQueMap[msgQue.id] = &msgQue
	msgQueMapSync.Unlock()

	Go(func() {
		Infof("process read for msgQue:%d", msgQue.id)
		msgQue.read()
		Infof("process read end for msgQue:%d", msgQue.id)
	})
	Go(func() {
		Infof("process write for msgQue:%d", msgQue.id)
		msgQue.write()
		Infof("process write end for msgQue:%d", msgQue.id)
	})

	Infof("new msgQue id:%d from addr:%s", msgQue.id, addr.String())
	return &msgQue
}

func newUDPListen(conn *net.UDPConn, msgtyp MsgType, handler IMsgHandler, parser *Parser, addr string) *udpMsgQue {
	msgQue := udpMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgQueID, 1),
			msgTyp:        msgtyp,
			handler:       handler,
			available:     true,
			parserFactory: parser,
			connTyp:       ConnTypeListen,
		},
		conn: conn,
	}
	conn.SetReadBuffer(1 << 24)
	conn.SetWriteBuffer(1 << 24)
	msgQueMapSync.Lock()
	msgQueMap[msgQue.id] = &msgQue
	msgQueMapSync.Unlock()
	Infof("new udp listen id:%d addr:%s", msgQue.id, addr)
	return &msgQue
}

var UDPServerGoCnt = 32

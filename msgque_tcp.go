package sugar

import (
	"bufio"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type tcpMsgQue struct {
	msgQue
	conn       net.Conn
	listener   net.Listener
	network    string
	address    string
	wait       sync.WaitGroup
	connecting int32
}

func (r *tcpMsgQue) GetNetType() NetType {
	return NetTypeTCP
}
func (r *tcpMsgQue) Stop() {
	if atomic.CompareAndSwapInt32(&r.stop, 0, 1) {
		Go(func() {
			if r.init {
				r.handler.OnDelMsgQue(r)
				if r.connecting == 1 {
					r.available = false
					return
				}
			}
			r.available = false
			if r.listener != nil {
				if tcp, ok := r.listener.(*net.TCPListener); ok {
					tcp.Close()
				}
			}

			r.BaseStop()
		})
	}
}

func (r *tcpMsgQue) IsStop() bool {
	if r.stop == 0 {
		if IsStop() {
			r.Stop()
		}
	}
	return r.stop == 1
}

func (r *tcpMsgQue) LocalAddr() string {
	if r.conn != nil {
		return r.conn.LocalAddr().String()
	} else if r.listener != nil {
		return r.listener.Addr().String()
	}
	return ""
}

func (r *tcpMsgQue) RemoteAddr() string {
	if r.conn != nil {
		return r.conn.RemoteAddr().String()
	}
	return ""
}

func (r *tcpMsgQue) readMsg() {
	headData := make([]byte, MsgHeadSize)
	var data []byte
	var head *MessageHead

	for !r.IsStop() {
		if r.timeout > 0 {
			r.conn.SetReadDeadline(time.Now().Add(time.Duration(r.timeout) * time.Second))
		}
		if head == nil {
			_, err := io.ReadFull(r.conn, headData)
			if err != nil {
				if err != io.EOF {
					Errorf("msgQue:%v recv data err:%v", r.id, err)
				}
				break
			}
			if head = NewMessageHead(headData); head == nil {
				Errorf("msgQue:%v read msg head failed", r.id)
				break
			}
			if head.Len == 0 {
				if !r.processMsg(r, &Message{Head: head}) {
					Errorf("msgQue:%v process msg cmd:%v act:%v", r.id, head.Cmd, head.Act)
					break
				}
				head = nil
			} else {
				data = make([]byte, head.Len)
			}
		} else {
			_, err := io.ReadFull(r.conn, data)
			if err != nil {
				Errorf("msgQue:%v recv data err:%v", r.id, err)
				break
			}

			if !r.processMsg(r, &Message{Head: head, Data: data}) {
				Errorf("msgQue:%v process msg cmd:%v act:%v", r.id, head.Cmd, head.Act)
				break
			}

			head = nil
			data = nil
		}
	}
}

func (r *tcpMsgQue) writeMsg() {
	var m *Message
	var head []byte
	writeCount := 0
	for !r.IsStop() || m != nil {
		if m == nil {
			select {
			case m = <-r.writeCh:
				if m != nil {
					head = m.Head.Bytes()
				}
			}
		}
		if m != nil {
			if r.timeout > 0 {
				r.conn.SetWriteDeadline(time.Now().Add(time.Duration(r.timeout) * time.Second))
			}
			if writeCount < MsgHeadSize {
				n, err := r.conn.Write(head[writeCount:])
				if err != nil {
					Errorf("msgQue write id:%v err:%v", r.id, err)
					r.Stop()
					break
				}
				writeCount += n
			}

			if writeCount >= MsgHeadSize && m.Data != nil {
				n, err := r.conn.Write(m.Data[writeCount-MsgHeadSize : int(m.Head.Len)])
				if err != nil {
					Errorf("msgQue write id:%v err:%v", r.id, err)
					break
				}
				writeCount += n
			}

			if writeCount == int(m.Head.Len)+MsgHeadSize {
				writeCount = 0
				m = nil
			}
		}
	}
}

func (r *tcpMsgQue) readCmd() {
	reader := bufio.NewReader(r.conn)
	for !r.IsStop() {
		if r.timeout > 0 {
			r.conn.SetReadDeadline(time.Now().Add(time.Duration(r.timeout) * time.Second))
		}
		data, err := reader.ReadBytes('\n')
		if err != nil {
			break
		}
		if !r.processMsg(r, &Message{Data: data}) {
			break
		}
	}
}

func (r *tcpMsgQue) writeCmd() {
	var m *Message
	writeCount := 0
	for !r.IsStop() || m != nil {
		if m == nil {
			select {
			case m = <-r.writeCh:
			}
		}
		if m != nil {
			if r.timeout > 0 {
				r.conn.SetWriteDeadline(time.Now().Add(time.Duration(r.timeout) * time.Second))
			}
			n, err := r.conn.Write(m.Data[writeCount:])
			if err != nil {
				Errorf("msgQue write id:%v err:%v", r.id, err)
				break
			}
			writeCount += n
			if writeCount == len(m.Data) {
				writeCount = 0
				m = nil
			}
		}
	}
}

func (r *tcpMsgQue) read() {
	defer func() {
		r.wait.Done()
		if err := recover(); err != nil {
			Errorf("msgQue read panic id:%v err:%v", r.id, err.(error))
			LogStack()
		}
		r.Stop()
	}()

	r.wait.Add(1)
	if r.msgTyp == MsgTypeCmd {
		r.readCmd()
	} else {
		r.readMsg()
	}
}

func (r *tcpMsgQue) write() {
	defer func() {
		r.wait.Done()
		if err := recover(); err != nil {
			Errorf("msgQue write panic id:%v err:%v", r.id, err.(error))
		}
		if r.conn != nil {
			r.conn.Close()
		}
		r.Stop()
	}()
	r.wait.Add(1)
	if r.msgTyp == MsgTypeCmd {
		r.writeCmd()
	} else {
		r.writeMsg()
	}
}

func (r *tcpMsgQue) listen() {
	for !r.IsStop() {
		c, err := r.listener.Accept()
		if err != nil {
			break
		} else {
			Go(func() {
				msgQue := newTCPAccept(c, r.msgTyp, r.handler, r.parserFactory)
				if r.handler.OnNewMsgQue(msgQue) {
					msgQue.init = true
					msgQue.available = true
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
				} else {
					msgQue.Stop()
				}
			})
		}
	}

	r.Stop()
}

func (r *tcpMsgQue) connect() {
	Infof("connect to addr:%s msgQue:%d", r.address, r.id)
	c, err := net.DialTimeout(r.network, r.address, time.Second)
	if err != nil {
		Infof("connect to addr:%s failed msgQue:%d", r.address, r.id)
		r.handler.OnConnectComplete(r, false)
		atomic.CompareAndSwapInt32(&r.connecting, 1, 0)
		r.Stop()
	} else {
		r.conn = c
		r.available = true
		Infof("connect to addr:%s ok msgQue:%d", r.address, r.id)
		if r.handler.OnConnectComplete(r, true) {
			atomic.CompareAndSwapInt32(&r.connecting, 1, 0)
			Go(func() {
				Infof("process read for msgQue:%d", r.id)
				r.read()
				Infof("process read end for msgQue:%d", r.id)
			})
			Go(func() {
				Infof("process write for msgQue:%d", r.id)
				r.write()
				Infof("process write end for msgQue:%d", r.id)
			})
		} else {
			atomic.CompareAndSwapInt32(&r.connecting, 1, 0)
			r.Stop()
		}
	}
}

func (r *tcpMsgQue) Reconnect(t int) {
	if IsStop() {
		return
	}
	if r.conn != nil {
		if r.stop == 0 {
			return
		}
	}

	if !atomic.CompareAndSwapInt32(&r.connecting, 0, 1) {
		return
	}

	if r.init {
		if t < 1 {
			t = 1
		}
	}
	r.init = true
	Go(func() {
		if r.conn != nil {
			r.conn.Close()
			if len(r.writeCh) == 0 {
				r.writeCh <- nil
			}
			r.wait.Wait()
		}
		r.stop = 0
		if t > 0 {
			SetTimeout(t*1000, func(arg ...interface{}) int {
				r.connect()
				return 0
			})
		} else {
			r.connect()
		}

	})
}

func newTCPConn(network, addr string, conn net.Conn, msgTyp MsgType, handler IMsgHandler, parser *Parser, user interface{}) *tcpMsgQue {
	msgQue := tcpMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgQueID, 1),
			writeCh:       make(chan *Message, 64),
			msgTyp:        msgTyp,
			handler:       handler,
			timeout:       DefMsgQueTimeout,
			connTyp:       ConnTypeConn,
			parserFactory: parser,
			user:          user,
		},
		conn:    conn,
		network: network,
		address: addr,
	}
	if parser != nil {
		msgQue.parser = parser.Get()
	}
	msgQueMapSync.Lock()
	msgQueMap[msgQue.id] = &msgQue
	msgQueMapSync.Unlock()
	Infof("new msgQue id:%d connect to addr:%s:%s", msgQue.id, network, addr)
	return &msgQue
}

func newTCPAccept(conn net.Conn, msgtyp MsgType, handler IMsgHandler, parser *Parser) *tcpMsgQue {
	msgQue := tcpMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgQueID, 1),
			writeCh:       make(chan *Message, 64),
			msgTyp:        msgtyp,
			handler:       handler,
			timeout:       DefMsgQueTimeout,
			connTyp:       ConnTypeAccept,
			parserFactory: parser,
		},
		conn: conn,
	}
	if parser != nil {
		msgQue.parser = parser.Get()
	}
	msgQueMapSync.Lock()
	msgQueMap[msgQue.id] = &msgQue
	msgQueMapSync.Unlock()
	Infof("new msgQue id:%d from addr:%s", msgQue.id, conn.RemoteAddr().String())
	return &msgQue
}

func newTCPListen(listener net.Listener, msgtyp MsgType, handler IMsgHandler, parser *Parser, addr string) *tcpMsgQue {
	msgQue := tcpMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgQueID, 1),
			msgTyp:        msgtyp,
			handler:       handler,
			parserFactory: parser,
			connTyp:       ConnTypeListen,
		},
		listener: listener,
	}

	msgQueMapSync.Lock()
	msgQueMap[msgQue.id] = &msgQue
	msgQueMapSync.Unlock()
	Infof("new tcp listen id:%d addr:%s", msgQue.id, addr)
	return &msgQue
}

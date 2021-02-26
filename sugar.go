package sugar

import (
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

func StartServer(addr string, typ MsgType, handler IMsgHandler, parser *Parser) error {
	addrInfo := strings.Split(addr, "://")
	if addrInfo[0] == "tcp" || addrInfo[0] == "all" {
		listen, err := net.Listen("tcp", addrInfo[1])
		if err == nil {
			msgQue := newTCPListen(listen, typ, handler, parser, addr)
			Go(func() {
				cid := AddStopCheck("msgQue listen")
				Debugf("process listen for msgQue:%d", msgQue.id)
				msgQue.listen()
				Debugf("process listen end for msgQue:%d", msgQue.id)
				RemoveStopCheck(cid)
			})
		} else {
			Errorf("listen on %s failed, err:%s", addr, err)
			return err
		}
	}
	if addrInfo[0] == "udp" || addrInfo[0] == "all" {
		udpAddr, err := net.ResolveUDPAddr("udp", addrInfo[1])
		if err != nil {
			Errorf("listen on %s failed, err:%s", addr, err)
			return err
		}
		conn, err := net.ListenUDP("udp", udpAddr)
		if err == nil {
			msgQue := newUDPListen(conn, typ, handler, parser, addr)
			Go(func() {
				Debugf("process listen for msgQue:%d", msgQue.id)
				msgQue.listen()
				Debugf("process listen end for msgQue:%d", msgQue.id)
			})
		} else {
			Errorf("listen on %s failed, err:%s", addr, err)
			return err
		}
	}
	return nil
}

func StartConnect(netType string, addr string, typ MsgType, handler IMsgHandler, parser *Parser, user interface{}) IMsgQue {
	msgQue := newTCPConn(netType, addr, nil, typ, handler, parser, user)
	if handler.OnNewMsgQue(msgQue) {
		msgQue.Reconnect(0)
		return msgQue
	}
	msgQue.Stop()
	return nil
}

func Daemon(skip ...string) {
	if os.Getppid() != 1 {
		filePath, _ := filepath.Abs(os.Args[0])
		var newCmd []string
		for _, v := range os.Args {
			add := true
			for _, s := range skip {
				if strings.Contains(v, s) {
					add = false
					break
				}
			}
			if add {
				newCmd = append(newCmd, v)
			}
		}
		cmd := exec.Command(filePath)
		cmd.Args = newCmd
		cmd.Start()
	}
}

func WaitForSystemExit(atexit ...func()) {
	stat.StartTime = time.Now()
	stopChan = make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, os.Kill, syscall.SIGTERM)
	for stop == 0 {
		select {
		case <-stopChan:
			Stop()
		}
	}
	Stop()
	for _, v := range atexit {
		v()
	}
	for _, v := range redisManagers {
		v.close()
	}
	waitAllForRedis.Wait()
}

func Stop() {
	if !atomic.CompareAndSwapInt32(&stop, 0, 1) {
		return
	}

	for _, v := range msgQueMap {
		v.Stop()
	}

	// trigger routines final clean
	notifyRoutinesClose()

	stopChan <- nil

	// wait all routines closed
	for sc := 0; !waitAll.TryWait(); sc++ {
		Sleep(1)
		if sc >= 3000 {
			stopCheckMap.Lock()
			for _, v := range stopCheckMap.M {
				Errorf("Server Stop Timeout: %v", v)
			}
			stopCheckMap.Unlock()
			sc = 0
		}
	}

	Info("Server Stop")
}

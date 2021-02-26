package sugar

import (
	"sync"
	"sync/atomic"
)

var stopMap = map[uint64]chan struct{}{}
var stopMapLock sync.Mutex

// Try
func Try(fun func(), handler func(interface{})) {
	defer func() {
		if err := recover(); err != nil {
			if handler == nil {
				LogStack()
				Errorf("error catch:%v", err)
			} else {
				handler(err)
			}
			atomic.AddInt32(&stat.PanicCount, 1)
			stat.LastPanic = int(Timestamp)
		}
	}()
	fun()
}

// Go
func Go(fn func()) {
	waitAll.Add(1)
	id := atomic.AddUint64(&goID, 1)
	c := atomic.AddInt64(&goCount, 1)
	DebugRoutineStartStack(id, c)
	go func() {
		Try(fn, nil)
		waitAll.Done()
		c = atomic.AddInt64(&goCount, -1)

		DebugRoutineEndStack(id, c)
	}()
}

// notifyRoutinesClose
func notifyRoutinesClose() {
	stopMapLock.Lock()
	for k, v := range stopMap {
		close(v)
		delete(stopMap, k)
	}
	stopMapLock.Unlock()
}

// Go2
func Go2(fn func(stopCh chan struct{})) bool {
	if IsStop() {
		return false
	}
	waitAll.Add(1)
	id := atomic.AddUint64(&goID, 1)
	c := atomic.AddInt64(&goCount, 1)
	DebugRoutineStartStack(id, c)

	go func() {
		id := atomic.AddUint64(&goID, 1)
		stopCh := make(chan struct{})
		stopMapLock.Lock()
		stopMap[id] = stopCh
		stopMapLock.Unlock()
		Try(func() { fn(stopCh) }, nil)

		stopMapLock.Lock()
		if _, ok := stopMap[id]; ok {
			close(stopCh)
			delete(stopMap, id)
		}
		stopMapLock.Unlock()

		waitAll.Done()
		c = atomic.AddInt64(&goCount, -1)
		DebugRoutineEndStack(id, c)
	}()
	return true
}

// GoArgs
func GoArgs(fn func(...interface{}), args ...interface{}) {
	waitAll.Add(1)
	id := atomic.AddUint64(&goID, 1)
	c := atomic.AddInt64(&goCount, 1)
	DebugRoutineStartStack(id, c)

	go func() {
		Try(func() { fn(args...) }, nil)

		waitAll.Done()
		c = atomic.AddInt64(&goCount, -1)
		DebugRoutineEndStack(id, c)
	}()
}

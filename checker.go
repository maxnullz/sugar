package sugar

import (
	"os"
	"sync"
	"sync/atomic"
)

type StopCheckMap struct {
	sync.Mutex
	M map[uint64]string
}

func IsStop() bool {
	return stop == 1
}

func IsRunning() bool {
	return stop == 0
}

func AddStopCheck(cs string) uint64 {
	id := atomic.AddUint64(&stopCheckIndex, 1)
	if id == 0 {
		id = atomic.AddUint64(&stopCheckIndex, 1)
	}
	stopCheckMap.Lock()
	stopCheckMap.M[id] = cs
	stopCheckMap.Unlock()
	return id
}

func RemoveStopCheck(id uint64) {
	stopCheckMap.Lock()
	delete(stopCheckMap.M, id)
	stopCheckMap.Unlock()
}

var stopCheckIndex uint64
var stopCheckMap = StopCheckMap{M: map[uint64]string{}}
var stop int32 // stop sign
var stopChan chan os.Signal

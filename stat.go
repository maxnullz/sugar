package sugar

import "time"

type Stat struct {
	GoCount     int
	MsgQueCount int
	StartTime   time.Time
	LastPanic   int
	PanicCount  int32
}

func GetStat() *Stat {
	stat.GoCount = int(goCount)
	stat.MsgQueCount = len(msgQueMap)
	return &stat
}

var goID uint64
var goCount int64 //number of goroutines
var stat = Stat{}

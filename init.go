package sugar

import (
	"runtime"

	"go.uber.org/zap"
)

func init() {
	z, _ := zap.NewProduction(zap.AddCallerSkip(1))
	logger = z.Sugar()
	runtime.GOMAXPROCS(runtime.NumCPU())
	timerTick()
}

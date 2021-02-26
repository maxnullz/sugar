package sugar

import (
	"fmt"
	"runtime"
	"strings"
)

type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Panic(args ...interface{})
	Fatal(args ...interface{})

	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Panicf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

var (
	logger Logger
)

func HookLogger(in Logger) {
	logger = in
}

func Debug(args ...interface{}) {
	logger.Debug(args)
}
func Info(args ...interface{}) {
	logger.Info(args)
}
func Warn(args ...interface{}) {
	logger.Warn(args)
}
func Error(args ...interface{}) {
	logger.Error(args)
}
func Panic(args ...interface{}) {
	logger.Panic(args)
}
func Fatal(args ...interface{}) {
	logger.Fatal(args)
}

func Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args)
}
func Infof(format string, args ...interface{}) {
	logger.Infof(format, args)
}
func Warnf(format string, args ...interface{}) {
	logger.Warnf(format, args)
}
func Errorf(format string, args ...interface{}) {
	logger.Errorf(format, args)
}
func Panicf(format string, args ...interface{}) {
	logger.Panicf(format, args)
}
func Fatalf(format string, args ...interface{}) {
	logger.Fatalf(format, args)
}

func LogStack() {
	buf := make([]byte, 1<<12)
	Error(string(buf[:runtime.Stack(buf, false)]))
}

func DebugRoutineStartStack(id uint64, count int64) {
	_, file, line, _ := runtime.Caller(2)
	i := strings.LastIndex(file, "/") + 1
	i = strings.LastIndex((string)(([]byte(file))[:i-1]), "/") + 1
	stack := fmt.Sprintf("%s:%d", (string)(([]byte(file))[i:]), line)
	Debugf("goroutine start id:%d count:%d stack: %s", id, count, stack)
}

func DebugRoutineEndStack(id uint64, count int64) {
	_, file, line, _ := runtime.Caller(2)
	i := strings.LastIndex(file, "/") + 1
	i = strings.LastIndex((string)(([]byte(file))[:i-1]), "/") + 1
	stack := fmt.Sprintf("%s:%d", (string)(([]byte(file))[i:]), line)
	Debugf("goroutine start id:%d count:%d stack: %s", id, count, stack)
}

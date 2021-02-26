package sugar

import "time"

var StartTick int64
var NowTick int64
var Timestamp int64

func SetTimeout(interval int, fn func(...interface{}) int, args ...interface{}) {
	if interval < 0 {
		Errorf("invalid timeout interval:%v", interval)
		return
	}
	Debugf("timeout  interval:%v", interval)

	Go2(func(stopCh chan struct{}) {
		var tick *time.Timer
		for interval > 0 {
			tick = time.NewTimer(time.Millisecond * time.Duration(interval))
			select {
			case <-stopCh:
				interval = 0
			case <-tick.C:
				tick.Stop()
				interval = fn(args...)
			}
		}
		if tick != nil {
			tick.Stop()
		}
	})
}

func timerTick() {
	StartTick = time.Now().UnixNano() / 1000000
	NowTick = StartTick
	Timestamp = NowTick / 1000
	Go(func() {
		for IsRunning() {
			Sleep(1)
			NowTick = time.Now().UnixNano() / 1000000
			Timestamp = NowTick / 1000
		}
	})
}

const layout = "2006-01-02 15:04:05"

func ParseTime(str string) (time.Time, error) {
	return time.Parse(layout, str)
}

func Date() string {
	return time.Now().Format(layout)
}

func UnixTime(sec, nsec int64) time.Time {
	return time.Unix(sec, nsec)
}

func UnixMs() int64 {
	return time.Now().UnixNano() / 1000000
}

func Now() time.Time {
	return time.Now()
}

func Sleep(ms int) {
	time.Sleep(time.Millisecond * time.Duration(ms))
}

func GetNextHourIntervalS(timestamp int64) int {
	return int(3600 - (timestamp % 3600))
}

func GetNextHourIntervalMS(timestamp int64) int {
	return GetNextHourIntervalS(timestamp) * 1000
}

func GetHour24(timestamp int64, timezone int) int {
	hour := int((timestamp%86400)/3600) + timezone
	if hour > 24 {
		return hour - 24
	}
	return hour
}

func GetHour23(timestamp int64, timezone int) int {
	hour := GetHour24(timestamp, timezone)
	if hour == 24 {
		return 0 // 24:00 equal to 00:00
	}
	return hour
}

func GetHour(timestamp int64, timezone int) int {
	return GetHour23(timestamp, timezone)
}

func IsDiffDay(now, old int64, timezone int) int {
	now += int64(timezone * 3600)
	old += int64(timezone * 3600)
	return int((now / 86400) - (old / 86400))
}

func IsDiffHour(now, old int64, hour, timezone int) bool {
	diff := IsDiffDay(now, old, timezone)
	if diff == 1 {
		if GetHour23(old, timezone) > hour {
			return GetHour23(now, timezone) >= hour
		}
	} else if diff >= 2 {
		return true
	}

	return (GetHour23(now, timezone) >= hour) && (GetHour23(old, timezone) < hour)
}

func IsDiffWeek(now, old int64, hour, timezone int) bool {
	diffHour := IsDiffHour(now, old, hour, timezone)
	now += int64(timezone * 3600)
	old += int64(timezone * 3600)
	_, nw := time.Unix(now, 0).UTC().ISOWeek()
	_, ow := time.Unix(old, 0).UTC().ISOWeek()
	return nw != ow && diffHour
}

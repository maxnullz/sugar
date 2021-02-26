package sugar

import (
	"net"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-redis/redis"
)

type RedisConfig struct {
	Addr     string
	Password string
	PoolSize int
}

type Redis struct {
	*redis.Client
	pubSub  *redis.PubSub
	conf    *RedisConfig
	manager *RedisManager
}

func (r *Redis) ScriptStr(cmd int, keys []string, args ...interface{}) (string, error) {
	data, err := r.Script(cmd, keys, args...)
	if err != nil {
		return "", err
	}
	errcode, ok := data.(int64)
	if ok {
		return "", GetError(uint16(errcode))
	}

	str, ok := data.(string)
	if !ok {
		return "", ErrDBDataType
	}

	return str, nil
}

func (r *Redis) ScriptStrArray(cmd int, keys []string, args ...interface{}) ([]string, error) {
	data, err := r.Script(cmd, keys, args...)
	if err != nil {
		return nil, err
	}
	errcode, ok := data.(int64)
	if ok {
		return nil, GetError(uint16(errcode))
	}

	iArray, ok := data.([]interface{})
	if !ok {
		return nil, ErrDBDataType
	}

	var strArray []string
	for _, v := range iArray {
		if str, ok := v.(string); ok {
			strArray = append(strArray, str)
		} else {
			return nil, ErrDBDataType
		}
	}

	return strArray, nil
}

func (r *Redis) ScriptInt64(cmd int, keys []string, args ...interface{}) (int64, error) {
	data, err := r.Script(cmd, keys, args...)
	if err != nil {
		return 0, err
	}
	code, ok := data.(int64)
	if ok {
		return code, nil
	}
	return 0, ErrDBDataType
}

func (r *Redis) Script(cmd int, keys []string, args ...interface{}) (interface{}, error) {
	hash, _ := scriptHashMap[cmd]
	re, err := r.EvalSha(hash, keys, args...).Result()
	if err != nil {
		script, ok := scriptMap[cmd]
		if !ok {
			return nil, err
		}

		if strings.HasPrefix(err.Error(), "NOSCRIPT ") {
			Infof("try reload redis script %v", scriptCommitMap[cmd])
			hash, err = r.ScriptLoad(script).Result()
			if err != nil {
				return nil, err
			}
			scriptHashMap[cmd] = hash
			re, err = r.EvalSha(hash, keys, args...).Result()
			if err == nil {
				return re, nil
			}
		}
		return nil, err
	}

	return re, nil
}

type RedisManager struct {
	dbs      map[int]*Redis
	subMap   map[string]*Redis
	channels []string
	fun      func(channel, data string)
	lock     sync.RWMutex
}

func (r *RedisManager) GetByRid(rid int) *Redis {
	r.lock.RLock()
	db := r.dbs[rid]
	r.lock.RUnlock()
	return db
}

func (r *RedisManager) GetGlobal() *Redis {
	return r.GetByRid(0)
}

func (r *RedisManager) Sub(fun func(channel, data string), channels ...string) {
	r.channels = channels
	r.fun = fun
	for _, v := range r.subMap {
		if v.pubSub != nil {
			v.pubSub.Close()
		}
	}
	for _, v := range r.subMap {
		pubSub := v.Subscribe(channels...)
		v.pubSub = pubSub
		goForRedis(func() {
			for IsRunning() {
				msg, err := pubSub.ReceiveMessage()
				if err == nil {
					Go(func() { fun(msg.Channel, msg.Payload) })
				} else if _, ok := err.(net.Error); !ok {
					break
				}
			}
		})
	}
}

func (r *RedisManager) Exist(id int) bool {
	r.lock.Lock()
	_, ok := r.dbs[id]
	r.lock.Unlock()
	return ok
}

func (r *RedisManager) Add(id int, conf *RedisConfig) {
	r.lock.Lock()
	if _, ok := r.dbs[id]; ok {
		Errorf("redis already have id:%v", id)
		r.lock.Unlock()
		return
	}
	r.lock.Unlock()
	re := &Redis{
		Client: redis.NewClient(&redis.Options{
			Addr:     conf.Addr,
			Password: conf.Password,
			PoolSize: conf.PoolSize,
		}),
		conf:    conf,
		manager: r,
	}

	if _, ok := r.subMap[conf.Addr]; !ok {
		r.subMap[conf.Addr] = re
		if len(r.channels) > 0 {
			pubSub := re.Subscribe(r.channels...)
			re.pubSub = pubSub
			goForRedis(func() {
				for IsRunning() {
					msg, err := pubSub.ReceiveMessage()
					if err == nil {
						Go(func() { r.fun(msg.Channel, msg.Payload) })
					} else if _, ok := err.(net.Error); !ok {
						break
					}
				}
			})
		}
	}

	r.lock.Lock()
	r.dbs[id] = re
	r.lock.Unlock()
	Infof("connect to redis %v", conf.Addr)
}

func (r *RedisManager) close() {
	for _, v := range r.dbs {
		if v.pubSub != nil {
			v.pubSub.Close()
		}
		v.Close()
	}
}

var (
	scriptMap       = map[int]string{}
	scriptCommitMap = map[int]string{}
	scriptHashMap   = map[int]string{}
	scriptIndex     int32
)

func NewRedisScript(commit, str string) int {
	cmd := int(atomic.AddInt32(&scriptIndex, 1))
	scriptMap[cmd] = str
	scriptCommitMap[cmd] = commit
	return cmd
}

func NewRedisManager(conf *RedisConfig) *RedisManager {
	redisManager := &RedisManager{
		subMap: map[string]*Redis{},
		dbs:    map[int]*Redis{},
	}

	redisManager.Add(0, conf)
	redisManagers = append(redisManagers, redisManager)
	return redisManager
}

func RedisError(err error) bool {
	if err == redis.Nil {
		return false
	}
	return err != nil
}

func goForRedis(fn func()) {
	waitAllForRedis.Add(1)
	id := atomic.AddUint64(&goID, 1)
	c := atomic.AddInt64(&goCount, 1)
	DebugRoutineStartStack(id, c)

	go func() {
		Try(fn, nil)

		waitAllForRedis.Done()
		c = atomic.AddInt64(&goCount, -1)
		DebugRoutineEndStack(id, c)
	}()
}

var waitAllForRedis sync.WaitGroup
var redisManagers []*RedisManager

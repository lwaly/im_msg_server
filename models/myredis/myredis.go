package myredis

import (
	"time"

	"im_msg_server/common"

	"github.com/gomodule/redigo/redis"
)

const (
	REDIS_T_HASH    = 1
	REDIS_T_SET     = 2
	REDIS_T_KEYS    = 3
	REDIS_T_STRING  = 4
	REDIS_T_LIST    = 5
	REDIS_T_SORTSET = 6
)

const (
	IM_DATA_USER_MSG_INFO         = 2001 //用户消息结构体
	IM_DATA_USER_MSG_UNREAD_LIST  = 2002 //用户未读消息列表
	IM_DATA_USER_MSG_READ_LIST    = 2003 //用户已读消息列表
	IM_DATA_GROUP_MSG_INFO        = 2004 //群消息结构体
	IM_DATA_GROUP_MSG_UNREAD_LIST = 2005 //用户未读群消息列表
	IM_DATA_GROUP_MSG_READ_LIST   = 2006 //用户已读群消息列表
	IM_DATA_USER_RELATION_LIST    = 2007 //用户关系列表
	IM_DATA_GROUP_MEM_LIST        = 2008 //群组成员列表
)

const (
	MGO_IDLE_COUNT   = 100 //连接池空闲个数
	MGO_ACTIVE_COUNT = 100 //连接池活动个数
	MGO_IDLE_TIMEOUT = 180 //空闲超时时间
)

var RedisClients map[int]*redis.Pool

func init() {
	str := common.Conf.GetKeyList("redis")
	RedisClients = make(map[int]*redis.Pool)
	index := 0
	for _, value := range str {
		if str1, err := common.Conf.GetValue("redis", value); nil == err {
			RedisClients[index] = createPool(MGO_IDLE_COUNT, MGO_ACTIVE_COUNT, MGO_IDLE_TIMEOUT, str1)
			index++
		}
	}

	str = common.Conf.GetKeyList("queue")
	for _, value := range str {
		if str1, err := common.Conf.GetValue("queue", value); nil == err {
			RedisClients[index] = createPool(MGO_IDLE_COUNT, MGO_ACTIVE_COUNT, MGO_IDLE_TIMEOUT, str1)
			index++
		}
	}
}

func createPool(maxIdle, maxActive, idleTimeout int, address string) (obj *redis.Pool) {
	obj = new(redis.Pool)
	obj.MaxIdle = maxIdle
	obj.MaxActive = maxActive
	obj.IdleTimeout = (time.Duration)(idleTimeout) * time.Second
	obj.Wait = true
	obj.Dial = func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", address)
		if err != nil {
			return nil, err
		}
		return c, err
	}
	return
}

func GetConn() (conn redis.Conn) {
	if len(RedisClients) <= 0 {
		return nil
	}

	return RedisClients[0].Get()
}

func GetQueueConn() (conn redis.Conn) {
	if len(RedisClients) <= 0 {
		return nil
	}

	return RedisClients[1].Get()
}

package models

//用户消息表结构定义，表操作文件
import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"im_msg_server/models/mymongo"
	"im_msg_server/models/myredis"

	"im_msg_server/common"

	"github.com/gomodule/redigo/redis"
)

const (
	UNREAD = 0
	READ   = 1
)

type UserMsg struct {
	Id          bson.ObjectId `bson:"_id" json:"Id,omitempty"`
	UId         uint64        `bson:"UId" json:"UId,omitempty"`
	ObjectId    uint64        `bson:"ObjectId" json:"ObjectId,omitempty"`
	MsgType     uint8         `bson:"MsgType" json:"MsgType,omitempty"` //消息类型:通知，聊天消息
	ContentType uint8         `bson:"ContentType" json:"ContentType,omitempty"`
	Content     string        `bson:"Content" json:"Content,omitempty"`
	Status      uint8         `bson:"Status" json:"Status,omitempty"`
	CreateTime  int64         `bson:"CreateTime" json:"CreateTimeId,omitempty"`
	UpdateTime  int64         `bson:"UpdateTime" json:"UpdateTime,omitempty"`
}

type StUserUnreadMsg struct {
	Id          string `bson:"_id" json:"Id,omitempty"`
	UId         uint64 `bson:"UId" json:"UId,omitempty"`
	MsgType     uint8  `bson:"MsgType" json:"MsgType,omitempty"` //消息类型:通知，聊天消息
	ContentType uint8  `bson:"ContentType" json:"ContentType,omitempty"`
	Content     string `bson:"Content" json:"Content,omitempty"`
	CreateTime  int64  `bson:"CreateTime" json:"CreateTime,omitempty"`
}

type StUserMsg struct {
	Id          string `bson:"_id" json:"Id,omitempty"`
	MsgType     uint8  `bson:"MsgType" json:"MsgType,omitempty"` //消息类型:通知，聊天消息
	ContentType uint8  `bson:"ContentType" json:"Type,omitempty"`
	Content     string `bson:"Content" json:"Content,omitempty"`
	CreateTime  int64  `bson:"CreateTime" json:"CreateTime,omitempty"`
}

//消息插入
func (info *UserMsg) Insert() (code int, err error) {
	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(user_msg)
	err = c.Insert(info)

	if err != nil {
		if mgo.IsDup(err) {
			code = ErrDupRows
		} else {
			code = ErrDatabase
		}
		common.Errorf("fail to insert.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
		return
	} else {
		code = 0
	}

	conn := myredis.GetConn()
	if conn.Err() != nil {
		err = conn.Err()
		code = ErrDatabase
		common.Errorf("fail to insert.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, conn.Err().Error())
		return
	}
	defer conn.Close()

	strKey := fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_HASH, myredis.IM_DATA_USER_MSG_INFO, info.ObjectId)
	bytes, _ := json.Marshal(info)
	strMember := fmt.Sprintf("%s", bytes)

	_, err = conn.Do("HSET", strKey, info.Id.Hex(), strMember)

	if err != nil {
		common.Errorf("fail to insert.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
		code = ErrDatabase
		return
	}

	strKey = fmt.Sprintf("%d:%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_USER_MSG_READ_LIST, info.UId, info.ObjectId)
	strMember = fmt.Sprintf("%s:%d", info.Id.Hex(), info.ObjectId)
	_, err = conn.Do("ZADD", strKey, info.CreateTime, strMember)

	if err != nil {
		common.Errorf("fail to insert.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
		code = ErrDatabase
		return
	}

	strKey = fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_USER_MSG_UNREAD_LIST, info.ObjectId)
	_, err = conn.Do("ZADD", strKey, info.CreateTime, strMember)

	if err != nil {
		common.Errorf("fail to insert.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
		code = ErrDatabase
	}
	return
}

//更新读状态
func (info *UserMsg) Update() (code int, err error) {
	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(user_msg)
	err = c.UpdateId(info.Id, bson.M{"$set": bson.M{"Status": info.Status, "UpdateTime": time.Now().UnixNano()}})
	if err != nil {
		common.Errorf("fail to update.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
		if err == mgo.ErrNotFound {
			return ErrNotFound, err
		}

		return ErrDatabase, err
	}

	conn := myredis.GetConn()
	if conn.Err() != nil {
		err = conn.Err()
		code = ErrDatabase
		common.Errorf("fail to update.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, conn.Err().Error())
		return
	}
	defer conn.Close()

	strKey := fmt.Sprintf("%d:%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_USER_MSG_READ_LIST, info.ObjectId, info.UId)
	strMember := fmt.Sprintf("%s:%d", info.Id.Hex(), info.ObjectId)
	_, err = conn.Do("ZADD", strKey, info.CreateTime, strMember)

	if err != nil {
		common.Errorf("fail to update.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
		code = ErrDatabase
		return
	}

	strKey = fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_USER_MSG_UNREAD_LIST, info.ObjectId)
	_, err = conn.Do("ZREM", strKey, strMember)

	if err != nil {
		common.Errorf("fail to update.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
		code = ErrDatabase
	}
	return
}

//批量更新读状态
func (info *UserMsg) Updates(Recodes []StUserUnreadMsg) (code int, err error) {
	mConn := mymongo.Conn()
	defer mConn.Close()

	var temps []string
	var temp string
	var tempTime string
	var tempKey string
	var tempMap map[string][]string
	tempMap = make(map[string][]string)

	c := mConn.DB(db).C(user_msg)
	for _, value := range Recodes {
		err = c.UpdateId(bson.ObjectIdHex(value.Id), bson.M{"$set": bson.M{"Status": info.Status, "UpdateTime": time.Now().UnixNano()}})
		if err != nil {
			common.Errorf("fail to update.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
			if err == mgo.ErrNotFound {
				return ErrNotFound, err
			}

			return ErrDatabase, err
		}
		temp = fmt.Sprintf("%s:%d", value.Id, info.ObjectId)
		temps = append(temps, temp)

		tempKey = fmt.Sprintf("%d:%d", info.ObjectId, value.UId)
		iter, ok := tempMap[tempKey]
		tempTime = fmt.Sprintf("%d", value.CreateTime)
		if ok {
			tempMap[tempKey] = append(iter, tempTime, temp)
		} else {
			tempMap[tempKey] = []string{tempTime, temp}
		}
	}

	conn := myredis.GetConn()
	if conn.Err() != nil {
		err = conn.Err()
		code = ErrDatabase
		common.Errorf("fail to update.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, conn.Err().Error())
		return
	}
	defer conn.Close()

	for index := range tempMap {
		strKey := fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_SORTSET, myredis.IM_DATA_USER_MSG_READ_LIST, index)
		_, err = conn.Do("ZADD", redis.Args{}.Add(strKey).AddFlat(tempMap[index])...)

		if err != nil {
			common.Errorf("fail to update.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
			code = ErrDatabase
			return
		}
	}

	strKey := fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_USER_MSG_UNREAD_LIST, info.ObjectId)
	_, err = conn.Do("ZREM", redis.Args{}.Add(strKey).AddFlat(temps)...)

	if err != nil {
		common.Errorf("fail to update.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
		code = ErrDatabase
	}
	return
}

//获取与某个用户已读消息
func (info *UserMsg) FindUserMsg(PageSzie int32, PageIndex int32) (Recodes []StUserMsg, code int, err error) {
	conn := myredis.GetConn()
	err = nil
	code = 0
	if conn.Err() != nil {
		common.Errorf("fail to find.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, conn.Err().Error())
	} else {
		defer conn.Close()

		strKey := fmt.Sprintf("%d:%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_USER_MSG_READ_LIST, info.UId, info.ObjectId)
		keys, errTemp := redis.Values(conn.Do("ZRANGE", strKey, PageSzie*(PageIndex-1), PageSzie*PageIndex))

		if errTemp != nil {
			common.Errorf("fail to find UserMsg.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, errTemp.Error())
		} else {
			var mapMsg map[string][]string
			mapMsg = make(map[string][]string)
			var msgTemp []string
			for _, value := range keys {
				msgTemp = strings.Split(string(value.([]byte)), ":")
				iter, ok := mapMsg[msgTemp[1]]
				if ok {
					mapMsg[msgTemp[1]] = append(iter, msgTemp[0])
				} else {
					mapMsg[msgTemp[1]] = []string{msgTemp[0]}
				}
			}

			for value := range mapMsg {
				strKey = fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_HASH, myredis.IM_DATA_USER_MSG_INFO, value)
				iResult, errTemp := redis.Values(conn.Do("HMGET", redis.Args{}.Add(strKey).AddFlat(mapMsg[value])...))
				if errTemp != nil {
					err = errTemp
					common.Errorf("fail to find UserMsg.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, errTemp.Error())
					break
				}

				for _, value := range iResult {
					temp := StUserMsg{}
					json.Unmarshal(value.([]byte), &temp)
					Recodes = append(Recodes, temp)
				}
			}
			if nil == err {
				return
			}
		}
	}

	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(user_msg)
	var res []UserMsg
	err = c.Find(bson.M{"$or": []bson.M{bson.M{"UId": info.UId, "ObjectId": info.ObjectId, "Status": info.Status},
		bson.M{"UId": info.UId, "ObjectId": info.ObjectId, "Status": info.Status}}}).Sort("-CreateTime").Skip(int(PageSzie * (PageIndex - 1))).Limit(int(PageSzie)).All(&res)

	if err != nil {
		if err == mgo.ErrNotFound {
			code = ErrNotFound
		} else {
			code = ErrDatabase
		}
		common.Errorf("fail to find UserMsg.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
		return
	}

	var mapInfo map[uint64][]string
	mapInfo = make(map[uint64][]string)
	for _, value := range res {
		bytes, _ := json.Marshal(value)
		strMember := fmt.Sprintf("%s", bytes)
		iter, ok := mapInfo[value.ObjectId]
		if ok {
			mapInfo[value.ObjectId] = append(iter, value.Id.Hex(), strMember)
		} else {
			mapInfo[value.ObjectId] = []string{value.Id.Hex(), strMember}
		}
		Recodes = append(Recodes, StUserMsg{value.Id.Hex(), value.MsgType, value.ContentType, value.Content, value.CreateTime})
	}

	for value := range mapInfo {
		strKey := fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_HASH, myredis.IM_DATA_USER_MSG_INFO, value)

		_, err = conn.Do("HMSET", redis.Args{}.Add(strKey).AddFlat(mapInfo[value])...)
		if err != nil {
			common.Errorf("fail to find UserMsg.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
			code = ErrDatabase
			break
		}
	}

	return
}

//获取用户未读消息
func (info *UserMsg) FindUserUnreadMsg(PageSzie int32, PageIndex int32) (Recodes []StUserUnreadMsg, code int, err error) {
	conn := myredis.GetConn()
	err = nil
	code = 0
	if conn.Err() != nil {
		common.Errorf("fail to find.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, conn.Err().Error())
	} else {
		defer conn.Close()

		strKey := fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_USER_MSG_UNREAD_LIST, info.ObjectId)
		keys, errTemp := redis.Values(conn.Do("ZRANGE", strKey, PageSzie*(PageIndex-1), PageSzie*PageIndex))

		if errTemp != nil {
			common.Errorf("fail to find UserMsg.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, errTemp.Error())
		} else {
			var mapMsg map[string][]string
			mapMsg = make(map[string][]string)
			var msgTemp []string
			for _, value := range keys {
				msgTemp = strings.Split(string(value.([]byte)), ":")
				iter, ok := mapMsg[msgTemp[1]]
				if ok {
					mapMsg[msgTemp[1]] = append(iter, msgTemp[0])
				} else {
					mapMsg[msgTemp[1]] = []string{msgTemp[0]}
				}
			}

			for value := range mapMsg {
				strKey = fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_HASH, myredis.IM_DATA_USER_MSG_INFO, value)
				iResult, errTemp := redis.Values(conn.Do("HMGET", redis.Args{}.Add(strKey).AddFlat(mapMsg[value])...))
				if errTemp != nil {
					err = errTemp
					common.Errorf("fail to find UserMsg.uid=%d,objectid=%s,err=%s", info.UId, value, errTemp.Error())
					break
				}

				for _, value := range iResult {
					temp := StUserUnreadMsg{}
					json.Unmarshal(value.([]byte), &temp)
					Recodes = append(Recodes, temp)
				}
			}
			if nil == err {
				return
			}
		}
	}

	mConn := mymongo.Conn()
	defer mConn.Close()
	var res []UserMsg
	c := mConn.DB(db).C(user_msg)
	err = c.Find(bson.M{"ObjectId": info.ObjectId, "Status": info.Status}).Sort("-CreateTime").Skip(int(PageSzie * (PageIndex - 1))).Limit(int(PageSzie)).All(&res)

	if err != nil {
		if err == mgo.ErrNotFound {
			code = ErrNotFound
		} else {
			code = ErrDatabase
		}
		common.Errorf("fail to find UserMsg.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
		return
	}

	var mapInfo map[uint64][]string
	mapInfo = make(map[uint64][]string)
	for _, value := range res {
		bytes, _ := json.Marshal(value)
		strMember := fmt.Sprintf("%s", bytes)
		iter, ok := mapInfo[value.ObjectId]
		if ok {
			mapInfo[value.ObjectId] = append(iter, value.Id.Hex(), strMember)
		} else {
			mapInfo[value.ObjectId] = []string{value.Id.Hex(), strMember}
		}
		Recodes = append(Recodes, StUserUnreadMsg{value.Id.Hex(), value.UId, value.MsgType, value.ContentType, value.Content, value.CreateTime})
	}

	for value := range mapInfo {
		strKey := fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_HASH, myredis.IM_DATA_USER_MSG_INFO, value)

		_, err = conn.Do("HMSET", redis.Args{}.Add(strKey).AddFlat(mapInfo[value])...)
		if err != nil {
			common.Errorf("fail to find UserMsg.uid=%d,objectid=%d,err=%s", info.UId, value, err.Error())
			code = ErrDatabase
			break
		}
	}
	return
}

func (info *UserMsg) FindUserMsgCount() (count int, code int, err error) {
	conn := myredis.GetConn()
	err = nil
	code = 0
	if conn.Err() != nil {
		common.Errorf("fail to find UserMsg.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, conn.Err().Error())
	} else {
		defer conn.Close()
		strKey := fmt.Sprintf("%d:%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_USER_MSG_READ_LIST, info.UId, info.ObjectId)
		Count, errTemp := redis.Int(conn.Do("ZCARD", strKey))

		if errTemp != nil {
			common.Errorf("fail to find UserMsg.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, errTemp.Error())
		} else {
			count = Count
			return
		}
	}

	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(user_msg)
	count, err = c.Find(bson.M{"$or": []bson.M{bson.M{"UId": info.UId, "ObjectId": info.ObjectId, "Status": info.Status},
		bson.M{"UId": info.UId, "ObjectId": info.ObjectId, "Status": info.Status}}}).Count()

	if err != nil {
		if err == mgo.ErrNotFound {
			code = ErrNotFound
		} else {
			code = ErrDatabase
		}
		common.Errorf("fail to find UserMsg.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
	} else {
		code = 0
	}
	return
}

func (info *UserMsg) FindUserMsgUnreadCount() (count int, code int, err error) {
	conn := myredis.GetConn()
	err = nil
	code = 0
	if conn.Err() != nil {
		common.Errorf("fail to find UserMsg.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, conn.Err().Error())
	} else {
		defer conn.Close()
		strKey := fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_USER_MSG_UNREAD_LIST, info.ObjectId)
		Count, errTemp := redis.Int(conn.Do("ZCARD", strKey))

		if errTemp != nil {
			common.Errorf("fail to find UserMsg.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, errTemp.Error())
		} else {
			count = Count
			return
		}
	}
	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(user_msg)
	count, err = c.Find(bson.M{"ObjectId": info.ObjectId, "Status": info.Status}).Count()

	if err != nil {
		if err == mgo.ErrNotFound {
			code = ErrNotFound
		} else {
			code = ErrDatabase
		}
		common.Errorf("fail to find UserMsg.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
	} else {
		code = 0
	}
	return
}

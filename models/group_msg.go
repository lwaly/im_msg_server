package models

//群消息表结构定义，表操作文件
import (
	"encoding/json"
	"fmt"
	"time"

	"im_msg_server/common"
	"im_msg_server/models/mymongo"
	"im_msg_server/models/myredis"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/gomodule/redigo/redis"
)

const (
	VALID   = 0
	INVALID = 1
)

type GroupMsg struct {
	Id          bson.ObjectId `bson:"_id" json:"Id,omitempty"`
	UId         uint64        `bson:"UId" json:"UId,omitempty"`
	GroupId     string        `bson:"GroupId" json:"GroupId,omitempty"`
	MsgType     uint8         `bson:"MsgType" json:"MsgType,omitempty"`         //消息类型，通知，聊天消息
	ContentType uint8         `bson:"ContentType" json:"ContentType,omitempty"` //消息内容类型，子段意义具体看数据库word文档
	Content     string        `bson:"Content" json:"Content,omitempty"`
	Status      uint8         `bson:"Status" json:"Status,omitempty"`
	CreateTime  int64         `bson:"CreateTime" json:"CreateTimeId,omitempty"`
	UpdateTime  int64         `bson:"UpdateTime" json:"UpdateTime,omitempty"`
}

type StGroupMsgUnread struct {
	Id          string `bson:"_id" json:"Id,omitempty"`
	UId         uint64 `bson:"UId" json:"UId,omitempty"`
	GroupId     string `bson:"GroupId" json:"GroupId,omitempty"`
	MsgType     uint8  `bson:"MsgType" json:"MsgType,omitempty"`         //消息类型，通知，聊天消息
	ContentType uint8  `bson:"ContentType" json:"ContentType,omitempty"` //消息内容类型，子段意义具体看数据库word文档
	Content     string `bson:"Content" json:"Content,omitempty"`
	CreateTime  int64  `bson:"CreateTime" json:"CreateTimeId,omitempty"`
}

type StGroupMsg struct {
	Id          string `bson:"_id" json:"Id,omitempty"`
	UId         uint64 `bson:"UId" json:"UId,omitempty"`
	ContentType uint8  `bson:"ContentType" json:"ContentType,omitempty"` //消息内容类型，子段意义具体看数据库word文档
	Content     string `bson:"Content" json:"Content,omitempty"`
	CreateTime  int64  `bson:"CreateTime" json:"CreateTimeId,omitempty"`
}

func (info *GroupMsg) Insert() (code int, err error) {
	mConn := mymongo.Conn()
	defer mConn.Close()
	code = common.OK
	c := mConn.DB(db).C(group_msg)
	err = c.Insert(info)

	if err != nil {
		if mgo.IsDup(err) {
			code = ErrDupRows
		} else {
			code = ErrDatabase
		}
		common.Errorf("fail to insert.uid=%d,objectid=%s,err=%s", info.UId, info.GroupId, err.Error())
		return
	}

	conn := myredis.GetConn()
	if conn.Err() != nil {
		common.Errorf("fail to insert.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, conn.Err().Error())
		err = conn.Err()
		code = ErrDatabase
		return
	}
	defer conn.Close()

	strKey := fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_HASH, myredis.IM_DATA_GROUP_MSG_INFO, info.GroupId)
	bytes, _ := json.Marshal(info)
	strMember := fmt.Sprintf("%s", bytes)

	_, err = conn.Do("HSET", strKey, info.Id.Hex(), strMember)

	if err != nil {
		common.Errorf("fail to insert.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		code = ErrDatabase
	}
	return
}

func (info *GroupMsg) Update() (code int, err error) {
	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(group_msg)
	err = c.UpdateId(info.Id, bson.M{"$set": bson.M{"Status": info.Status, "UpdateTime": time.Now()}})
	if err != nil {
		common.Errorf("fail to Update.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		if err == mgo.ErrNotFound {
			return ErrNotFound, err
		}

		return ErrDatabase, err
	}

	conn := myredis.GetConn()
	if conn.Err() != nil {
		err = conn.Err()
		code = ErrDatabase
		common.Errorf("fail to Update.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, conn.Err().Error())
		return
	}
	defer conn.Close()

	strKey := fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MSG_UNREAD_LIST, info.GroupId)
	_, err = conn.Do("ZREM", strKey, info.Id.Hex())

	if err != nil {
		common.Errorf("fail to Update.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		code = ErrDatabase
	}
	return
}

func (info *GroupMsg) FindGroupMsg(PageSzie int32, PageIndex int32) (Recodes []StGroupMsg, code int, err error) {
	conn := myredis.GetConn()
	err = nil
	code = 0
	if conn.Err() != nil {
		common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, conn.Err().Error())
	} else {
		defer conn.Close()

		strKey := fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MSG_READ_LIST, info.GroupId)
		keys, errTemp := redis.Values(conn.Do("ZRANGE", strKey, PageSzie*(PageIndex-1), PageSzie*PageIndex))

		if errTemp != nil {
			common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, errTemp.Error())
		} else {
			var msgTemp []string
			for _, value := range keys {
				msgTemp = append(msgTemp, string(value.([]byte)))
			}
			iResult, errTemp := redis.Values(conn.Do("HMGET", redis.Args{}.Add(strKey).AddFlat(msgTemp)...))
			if errTemp != nil {
				common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, errTemp.Error())
			} else {
				for _, value := range iResult {
					temp := StGroupMsg{}
					json.Unmarshal(value.([]byte), &temp)
					Recodes = append(Recodes, temp)
				}
				return
			}
		}
	}

	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(group_msg)
	err = c.Find(bson.M{"GroupId": info.GroupId, "Status": info.Status}).Sort("-CreateTime").Skip(int(PageSzie * (PageIndex - 1))).Limit(int(PageSzie)).All(&Recodes)

	if err != nil {
		if err == mgo.ErrNotFound {
			code = ErrNotFound
		} else {
			code = ErrDatabase
		}
		common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
	}
	return
}

func (info *GroupMsg) FindGroupMsgCount() (count int, code int, err error) {
	conn := myredis.GetConn()
	err = nil
	code = 0
	if conn.Err() != nil {
		common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, conn.Err().Error())
	} else {
		defer conn.Close()
		strKey := fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MSG_READ_LIST, info.GroupId)
		Count, errTemp := redis.Int(conn.Do("ZCARD", strKey))

		if errTemp != nil {
			common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, errTemp.Error())
		} else {
			count = Count
			return
		}
	}

	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(group_msg)
	count, err = c.Find(bson.M{"GroupId": info.GroupId, "Status": info.Status}).Count()

	if err != nil {
		if err == mgo.ErrNotFound {
			code = ErrNotFound
		} else {
			code = ErrDatabase
		}
		common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
	}
	return
}

func (info *GroupMsg) FindReadMsgByMsgId(Recodes map[string][]string) (Res []StGroupMsg, code int, err error) {
	conn := myredis.GetConn()
	err = nil
	code = 0
	if conn.Err() != nil {
		common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, conn.Err().Error())
	} else {
		defer conn.Close()

		for key, value := range Recodes {
			strKey := fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_HASH, myredis.IM_DATA_GROUP_MSG_INFO, key)
			iResult, errTemp := redis.Values(conn.Do("HMGET", redis.Args{}.Add(strKey).AddFlat(value)...))
			if errTemp != nil {
				common.Errorf("fail to find.msgid=%v,groupid=%s,err=%s", value, key, errTemp.Error())
				err = errTemp
				break
			} else {
				for _, value := range iResult {
					temp := StGroupMsg{}
					json.Unmarshal(value.([]byte), &temp)
					Res = append(Res, temp)
				}
			}
		}
		if nil == err {
			return
		}
	}

	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(group_msg)
	var temp GroupMsg
	for value := range Recodes {
		for _, msgId := range Recodes[value] {
			err = c.FindId(bson.ObjectIdHex(msgId)).One(&temp)

			if err != nil {
				if err == mgo.ErrNotFound {
					code = ErrNotFound
				} else {
					code = ErrDatabase
				}
				common.Errorf("fail to find.groupid=%s,msgId=%s,err=%s", value, msgId, err.Error())
				return
			} else {
				strKey := fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_HASH, myredis.IM_DATA_GROUP_MSG_INFO, temp.GroupId)
				bytes, _ := json.Marshal(temp)
				strMember := fmt.Sprintf("%s", bytes)

				iResult, errTemp := conn.Do("HSET", strKey, temp.Id.Hex(), strMember)

				if errTemp != nil {
					common.Errorf("fail to find group msg. groupid=%s,msgid=%s,err=%s code=%d", temp.GroupId, strMember, errTemp.Error(), iResult)
					code = ErrDatabase
					err = errTemp
					return
				}
				Res = append(Res, StGroupMsg{temp.Id.Hex(), temp.UId, temp.ContentType, temp.Content, temp.CreateTime})
			}
		}
	}

	return
}

func (info *GroupMsg) FindUnReadMsgByMsgId(Recodes map[string][]string) (Res []StGroupMsgUnread, code int, err error) {
	conn := myredis.GetConn()
	err = nil
	code = 0
	if conn.Err() != nil {
		common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, conn.Err().Error())
	} else {
		defer conn.Close()

		for key, value := range Recodes {
			strKey := fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_HASH, myredis.IM_DATA_GROUP_MSG_INFO, key)
			iResult, errTemp := redis.Values(conn.Do("HMGET", redis.Args{}.Add(strKey).AddFlat(value)...))
			if errTemp != nil {
				common.Errorf("fail to find.msgid=%v,groupid=%s,err=%s", value, key, errTemp.Error())
				err = errTemp
			} else {
				for _, value := range iResult {
					temp := StGroupMsgUnread{}
					json.Unmarshal(value.([]byte), &temp)
					Res = append(Res, temp)
				}
			}
		}
		if nil == err {
			return
		}
	}

	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(group_msg)
	var temp GroupMsg

	for index, value := range Recodes {
		for _, msgId := range value {
			err = c.FindId(bson.ObjectIdHex(msgId)).One(&temp)
			if err != nil {
				if err == mgo.ErrNotFound {
					code = ErrNotFound
				} else {
					code = ErrDatabase
				}
				common.Errorf("fail to find.msgId=%s,groupid=%s,err=%s", msgId, index, err.Error())
				return
			} else {
				strKey := fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_HASH, myredis.IM_DATA_GROUP_MSG_INFO, temp.GroupId)
				bytes, _ := json.Marshal(temp)
				strMember := fmt.Sprintf("%s", bytes)

				iResult, errTemp := conn.Do("HSET", strKey, info.Id.Hex(), strMember)

				if errTemp != nil {
					common.Errorf("fail to find group msg. groupid=%s,msgid=%s,err=%s code=%d", temp.GroupId, strMember, errTemp.Error(), iResult)
					code = ErrDatabase
					err = errTemp
					return
				}
				Res = append(Res, StGroupMsgUnread{temp.Id.Hex(), temp.UId, temp.GroupId, temp.MsgType, temp.ContentType, temp.Content, temp.CreateTime})
			}
		}
	}

	return
}

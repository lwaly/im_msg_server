package models

//用户群组消息表结构定义，表操作文件
import (
	"fmt"
	"strings"

	"im_msg_server/models/mymongo"
	"im_msg_server/models/myredis"

	"im_msg_server/common"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/gomodule/redigo/redis"
)

type GroupUserMsg struct {
	Id         bson.ObjectId `bson:"_id" json:"Id,omitempty"`
	UId        uint64        `bson:"UId" json:"UId,omitempty"`
	GroupId    string        `bson:"GroupId" json:"GroupId,omitempty"`
	MsgId      string        `bson:"MsgId" json:"MsgId,omitempty"`
	Type       uint8         `bson:"Type" json:"Type,omitempty"` //业务类型0:创建;1：申请;2:同意;3：拒绝；4:解散；5：主动退群；6：踢人；7：邀请；8：邀请审核同意；9：邀请审核拒绝；20：聊天消息
	Status     uint8         `bson:"Status" json:"Status,omitempty"`
	CreateTime int64         `bson:"CreateTime" json:"CreateTime,omitempty"`
	UpdateTime int64         `bson:"UpdateTime" json:"UpdateTime,omitempty"`
}

type GroupUserUnreadMsgId struct {
	MsgId   string `bson:"MsgId" json:"MsgId,omitempty"`
	GroupId string `bson:"GroupId" json:"GroupId,omitempty"`
}

type GroupUserMsgId struct {
	MsgId string `bson:"MsgId" json:"MsgId,omitempty"`
}

//插入用户未读消息
func (info *GroupUserMsg) InsertUnRead(records []GroupMem) (code int, err error) {
	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(group_user_msg)
	tempUid := info.UId
	for _, value := range records {
		if tempUid == value.UId {
			continue
		}
		info.Id = bson.NewObjectId()
		info.UId = value.UId
		if err = c.Insert(info); nil != err {
			common.Errorf("fail to insert.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
			break
		}
	}

	if err != nil {
		if mgo.IsDup(err) {
			code = ErrDupRows
		} else {
			code = ErrDatabase
		}
		common.Errorf("fail to insert.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		return
	} else {
		code = 0
	}

	conn := myredis.GetConn()
	if conn.Err() != nil {
		common.Errorf("fail to insert.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, conn.Err().Error())
		err = conn.Err()
		code = ErrDatabase
		return
	}

	defer conn.Close()

	var strKey string
	strMember := fmt.Sprintf("%s:%s", info.MsgId, info.GroupId)
	for _, value := range records {
		if tempUid == value.UId {
			continue
		}

		strKey = fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MSG_UNREAD_LIST, value.UId)
		_, err = conn.Do("ZADD", strKey, info.CreateTime, strMember)

		if err != nil {
			common.Errorf("fail to insert.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
			code = ErrDatabase
			return
		}
	}
	return
}

//插入用户已读读消息
func (info *GroupUserMsg) InsertRead() (code int, err error) {
	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(group_user_msg)
	info.Id = bson.NewObjectId()
	if err = c.Insert(info); nil != err {
		if mgo.IsDup(err) {
			code = ErrDupRows
		} else {
			code = ErrDatabase
		}
		common.Errorf("fail to insert.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		return
	} else {
		code = 0
	}

	conn := myredis.GetConn()
	if conn.Err() != nil {
		common.Errorf("fail to insert.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, conn.Err().Error())
		code = ErrDatabase
		err = conn.Err()
		return
	}
	defer conn.Close()

	strKey := fmt.Sprintf("%d:%d:%d:%s", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MSG_READ_LIST, info.UId, info.GroupId)
	_, err = conn.Do("ZADD", strKey, info.CreateTime, info.MsgId)

	if err != nil {
		common.Errorf("fail to insert.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		code = ErrDatabase
	}
	return
}

func (info *GroupUserMsg) Update(Recodes map[string][]string) (code int, err error) {
	conn := myredis.GetConn()
	if conn.Err() != nil {
		err = conn.Err()
		code = ErrDatabase
		common.Errorf("fail to Update.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		return
	} else {
		defer conn.Close()
	}

	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(group_user_msg)

	strKey := fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MSG_UNREAD_LIST, info.UId)

	for groupId, value := range Recodes {
		var tempRedis []string
		for _, msgId := range value {
			err = c.Update(bson.M{"UId": info.UId, "MsgId": msgId}, bson.M{"$set": bson.M{"Status": READ}})
			if err != nil {
				if err == mgo.ErrNotFound {
					code = ErrNotFound
				} else {
					code = ErrDatabase
				}
				common.Errorf("fail to Update.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
				return
			}
			tempRedis = append(tempRedis, msgId+":"+groupId)
		}

		_, err = conn.Do("ZREM", redis.Args{}.Add(strKey).AddFlat(tempRedis)...)

		if err != nil {
			common.Errorf("fail to Update.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
			return ErrDatabase, err
		}
	}
	return
}

//未读消息
func (info *GroupUserMsg) FindUserUnreadGroupMsg(PageSzie int) (Recodes map[string][]string, code int, err error) {
	conn := myredis.GetConn()
	err = nil
	code = 0
	if conn.Err() != nil {
		common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, conn.Err().Error())
	} else {
		defer conn.Close()

		strKey := fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MSG_UNREAD_LIST, info.UId)
		keys, errTemp := redis.Values(conn.Do("ZRANGE", strKey, 0, PageSzie))

		if errTemp != nil {
			common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, errTemp.Error())
		} else {
			var msgTemp []string
			Recodes = make(map[string][]string)
			for _, value := range keys {
				msgTemp = strings.Split(string(value.([]byte)), ":")
				iter, ok := Recodes[msgTemp[1]]
				if ok {
					Recodes[msgTemp[1]] = append(iter, msgTemp[0])
				} else {
					Recodes[msgTemp[1]] = []string{msgTemp[0]}
				}
			}
			return
		}
	}
	mConn := mymongo.Conn()
	defer mConn.Close()

	var res []GroupUserUnreadMsgId
	c := mConn.DB(db).C(group_user_msg)
	err = c.Find(bson.M{"UId": info.UId, "Status": UNREAD}).Sort("-CreateTime").Limit(PageSzie).All(&res)

	if err != nil {
		if err == mgo.ErrNotFound {
			code = ErrNotFound
		} else {
			code = ErrDatabase
		}
		common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		return
	}

	for _, value := range res {
		iter, ok := Recodes[value.GroupId]
		if ok {
			Recodes[value.GroupId] = append(iter, value.MsgId)
		} else {
			Recodes[value.GroupId] = []string{value.MsgId}
		}
	}

	return
}

//未读消息数
func (info *GroupUserMsg) FindUserUnreadGroupMsgCount() (count int, code int, err error) {
	conn := myredis.GetConn()
	err = nil
	code = 0
	if conn.Err() != nil {
		common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, conn.Err().Error())
	} else {
		defer conn.Close()
		strKey := fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MSG_UNREAD_LIST, info.UId)
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

	c := mConn.DB(db).C(group_user_msg)
	count, err = c.Find(bson.M{"UId": info.UId, "Status": UNREAD}).Count()

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

//已读消息
func (info *GroupUserMsg) FindUserGroupMsg(PageSzie int32, PageIndex int32) (Recodes map[string][]string, code int, err error) {
	conn := myredis.GetConn()
	err = nil
	code = 0

	if conn.Err() != nil {
		common.Errorf("fail to find GroupUserMsg.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, conn.Err().Error())
	} else {
		defer conn.Close()

		strKey := fmt.Sprintf("%d:%d:%d:%s", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MSG_READ_LIST, info.UId, info.GroupId)
		keys, errTemp := redis.Values(conn.Do("ZRANGE", strKey, PageSzie*(PageIndex-1), PageSzie*PageIndex))

		if errTemp != nil {
			common.Errorf("fail to find GroupUserMsg.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, errTemp.Error())
		} else {
			Recodes = make(map[string][]string)
			for _, value := range keys {
				iter, ok := Recodes[info.GroupId]
				if ok {
					Recodes[info.GroupId] = append(iter, string(value.([]byte)))
				} else {
					tempMsgId := []string{string(value.([]byte))}
					Recodes[info.GroupId] = tempMsgId
				}
			}
			return
		}
	}
	mConn := mymongo.Conn()
	defer mConn.Close()

	var res []GroupUserMsgId
	c := mConn.DB(db).C(group_user_msg)
	err = c.Find(bson.M{"UId": info.UId, "Status": READ}).Sort("-CreateTime").Limit(int(PageSzie)).All(&res)

	if err != nil {
		if err == mgo.ErrNotFound {
			code = ErrNotFound
		} else {
			code = ErrDatabase
		}
		common.Errorf("fail to find GroupUserMsg.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		return
	}

	for _, value := range res {
		Recodes[info.GroupId] = append(Recodes[info.GroupId], value.MsgId)
	}

	return
}

//已读消息数
func (info *GroupUserMsg) FindUserGroupMsgCount() (count int, code int, err error) {
	conn := myredis.GetConn()
	err = nil
	code = 0
	if conn.Err() != nil {
		common.Errorf("fail to find GroupUserMsg.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, conn.Err().Error())
	} else {
		defer conn.Close()
		strKey := fmt.Sprintf("%d:%d:%d:%s", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MSG_READ_LIST, info.UId, info.GroupId)
		Count, errTemp := redis.Int(conn.Do("ZCARD", strKey))

		if errTemp != nil {
			common.Errorf("fail to find GroupUserMsg.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, errTemp.Error())
		} else {
			count = Count
			return
		}
	}

	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(group_user_msg)
	count, err = c.Find(bson.M{"UId": info.UId, "Status": READ}).Count()

	if err != nil {
		if err == mgo.ErrNotFound {
			code = ErrNotFound
		} else {
			code = ErrDatabase
		}
		common.Errorf("fail to find GroupUserMsg.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
	}
	return
}

package models

//用户关系表结构定义，表操作文件
import (
	"fmt"

	"im_msg_server/models/mymongo"
	"im_msg_server/models/myredis"

	"im_msg_server/common"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/gomodule/redigo/redis"
)

type UserRelation struct {
	Id         bson.ObjectId `bson:"_id" json:"Id,omitempty"`
	UId        uint64        `bson:"UId" json:"UId,omitempty"`
	ObjectId   uint64        `bson:"ObjectId" json:"ObjectId,omitempty"`
	Status     uint8         `bson:"Status" json:"Status,omitempty"`
	CreateTime int64         `bson:"CreateTime" json:"CreateTime,omitempty"`
	UpdateTime int64         `bson:"UpdateTime" json:"UpdateTime,omitempty"`
}

func (info *UserRelation) Insert() (code int, err error) {
	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(user_relation)
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
		common.Errorf("fail to get redis connect.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, conn.Err().Error())
		code = ErrDatabase
		return
	}
	defer conn.Close()

	strKey := fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_USER_RELATION_LIST, info.UId)
	_, err = conn.Do("ZADD", strKey, info.Status, info.ObjectId)

	if err != nil {
		common.Errorf("fail to insert.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
		code = ErrDatabase
	}

	strKey = fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_USER_RELATION_LIST, info.ObjectId)
	_, err = conn.Do("ZADD", strKey, info.Status, info.UId)

	if err != nil {
		common.Errorf("fail to insert.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
		code = ErrDatabase
	}
	return
}

func (info *UserRelation) Update() (code int, err error) {
	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(user_relation)
	err = c.Update(bson.M{"UId": info.UId, "ObjectId": info.ObjectId}, bson.M{"$set": bson.M{"Status": info.Status}})

	if err != nil {
		common.Errorf("fail to update.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
		if err == mgo.ErrNotFound {
			return ErrNotFound, err
		}

		return ErrDatabase, err
	}

	conn := myredis.GetConn()
	if conn.Err() != nil {
		common.Errorf("fail to update.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, conn.Err().Error())
		return ErrDatabase, err
	}
	defer conn.Close()

	strKey := fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_USER_RELATION_LIST, info.UId)
	_, err = conn.Do("ZADD", strKey, info.Status, info.ObjectId)

	if err != nil {
		common.Errorf("fail to update.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
		return ErrDatabase, err
	}

	strKey = fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_USER_RELATION_LIST, info.ObjectId)
	_, err = conn.Do("ZADD", strKey, info.Status, info.UId)

	if err != nil {
		common.Errorf("fail to update.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
	}
	return ErrDatabase, err
}

func (info *UserRelation) FindByField() (code int, err error) {
	conn := myredis.GetConn()
	var strKey string
	err = nil
	code = 0
	if conn.Err() != nil {
		common.Errorf("fail to find.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, conn.Err().Error())
	} else {
		defer conn.Close()

		strKey = fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_USER_RELATION_LIST, info.UId)
		score, errTemp := redis.Int(conn.Do("ZSCORE", strKey, info.ObjectId))

		if errTemp != nil {
			common.Errorf("fail to find relation.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, errTemp.Error())
		} else {
			info.Status = uint8(score)
			return
		}
	}

	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(user_relation)
	err = c.Find(bson.M{"UId": info.UId, "ObjectId": info.ObjectId}).One(info)

	if err != nil {
		if err == mgo.ErrNotFound {
			code = ErrNotFound
		} else {
			code = ErrDatabase
		}
		common.Errorf("fail to find.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
		return
	} else {
		code = 0
	}

	_, err = conn.Do("ZADD", strKey, info.Status, info.ObjectId)

	if err != nil {
		common.Errorf("fail to find.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
		return ErrDatabase, err
	}

	strKey = fmt.Sprintf("%d:%d:%d", myredis.REDIS_T_SORTSET, myredis.IM_DATA_USER_RELATION_LIST, info.ObjectId)
	_, err = conn.Do("ZADD", strKey, info.Status, info.UId)

	if err != nil {
		common.Errorf("fail to find.uid=%d,objectid=%d,err=%s", info.UId, info.ObjectId, err.Error())
	}

	return ErrDatabase, err
}

package models

//群成员关系表结构定义，表操作文件
import (
	"fmt"
	"strconv"

	"im_msg_server/models/mymongo"
	"im_msg_server/models/myredis"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"im_msg_server/common"

	"github.com/gomodule/redigo/redis"
)

const (
	GROUP_ROLE_MEMBER = 1
	GROUP_ROLE_ADMIN  = 2
	GROUP_ROLE_OWNER  = 4
)

type GroupRelation struct {
	Id         bson.ObjectId `bson:"_id" json:"Id,omitempty"`
	UId        uint64        `bson:"UId" json:"UId,omitempty"`
	GroupId    string        `bson:"GroupId" json:"GroupId,omitempty"`
	Status     uint8         `bson:"Status" json:"Status,omitempty"`
	CreateTime int64         `bson:"CreateTime" json:"CreateTime,omitempty"`
	UpdateTime int64         `bson:"UpdateTime" json:"UpdateTime,omitempty"`
}

type GroupMem struct {
	UId uint64 `bson:"UId" json:"object_id,omitempty"`
}

type StObjectId struct {
	ObjectId uint64 `json:"object_id" valid:"Required"`
}

//批量插入群组关系
func (info *GroupRelation) Insert(records []GroupMem) (code int, err error) {
	mConn := mymongo.Conn()
	defer mConn.Close()

	var temp []uint64
	c := mConn.DB(db).C(group_relation)
	for _, value := range records {
		info.Id = bson.NewObjectId()
		info.UId = value.UId
		if err = c.Insert(info); nil != err {
			if mgo.IsDup(err) {
				code = ErrDupRows
			} else {
				code = ErrDatabase
			}
			common.Errorf("fail to insert.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
			return
		}
		temp = append(temp, uint64(info.Status), info.UId)
	}

	conn := myredis.GetConn()
	if conn.Err() != nil {
		err = conn.Err()
		code = ErrDatabase
		common.Errorf("fail to insert.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		return
	}
	defer conn.Close()

	strKey := fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MEM_LIST, info.GroupId)
	_, err = conn.Do("ZADD", redis.Args{}.Add(strKey).AddFlat(temp)...)

	if err != nil {
		common.Errorf("fail to insert.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		code = ErrDatabase
	}
	return
}

//批量更新群组成员权限
func (info *GroupRelation) Update(records []GroupMem) (code int, err error) {
	mConn := mymongo.Conn()
	defer mConn.Close()

	var temp []uint64
	c := mConn.DB(db).C(group_relation)
	for _, value := range records {
		if err = c.Update(bson.M{"UId": value.UId, "GroupId": info.GroupId}, bson.M{"$set": bson.M{"Status": info.Status}}); nil != err {
			common.Errorf("fail to Update.uid=%d,groupid=%s,err=%s", value.UId, info.GroupId, err.Error())
			if err == mgo.ErrNotFound {
				return ErrNotFound, err
			}
			return ErrDatabase, err
		}
		temp = append(temp, uint64(info.Status), info.UId)
	}

	conn := myredis.GetConn()
	if conn.Err() != nil {
		err = conn.Err()
		code = ErrDatabase
		common.Errorf("fail to Update.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		return
	}
	defer conn.Close()

	strKey := fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MEM_LIST, info.GroupId)
	_, err = conn.Do("ZADD", redis.Args{}.Add(strKey).AddFlat(temp)...)

	if err != nil {
		common.Errorf("fail to Update.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		code = ErrDatabase
	}

	return
}

//批量删除群组成员
func (info *GroupRelation) DelGroupMem(records []GroupMem) (code int, err error) {
	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(group_relation)

	for _, value := range records {
		if err = c.Remove(bson.M{"UId": value.UId, "GroupId": info.GroupId}); nil != err {
			common.Errorf("fail to DelGroupMem.uid=%d,groupid=%s,err=%s", value.UId, info.GroupId, err.Error())
			if err == mgo.ErrNotFound {
				return ErrNotFound, err
			}

			return ErrDatabase, err
		}
	}

	conn := myredis.GetConn()
	if conn.Err() != nil {
		err = conn.Err()
		code = ErrDatabase
		common.Errorf("fail to DelGroupMem.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		return
	}
	defer conn.Close()

	strKey := fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MEM_LIST, info.GroupId)
	_, err = conn.Do("ZREM", redis.Args{}.Add(strKey).AddFlat(records)...)

	if err != nil {
		common.Errorf("fail to DelGroupMem.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		code = ErrDatabase
	}
	return
}

//删除群
func (info *GroupRelation) DelGroup() (code int, err error) {
	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(group_relation)
	_, err = c.RemoveAll(bson.M{"GroupId": info.GroupId})

	if err != nil {
		common.Errorf("fail to DelGroup.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		if err == mgo.ErrNotFound {
			return ErrNotFound, err
		}

		return ErrDatabase, err
	}

	conn := myredis.GetConn()
	if conn.Err() != nil {
		err = conn.Err()
		code = ErrDatabase
		common.Errorf("fail to DelGroup.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		return
	}
	defer conn.Close()

	strKey := fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MEM_LIST, info.GroupId)
	_, err = conn.Do("DEL", strKey)

	if err != nil {
		common.Errorf("fail to DelGroup.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
		code = ErrDatabase
	}
	return
}

//查询某个用户在某群的信息
func (info *GroupRelation) FindByField() (code int, err error) {
	conn := myredis.GetConn()
	var strKey string
	err = nil
	code = 0
	if conn.Err() != nil {
		common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
	} else {
		defer conn.Close()

		strKey = fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MEM_LIST, info.GroupId)
		score, errTemp := redis.Int(conn.Do("ZSCORE", strKey, info.UId))

		if errTemp != nil {
			common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, errTemp.Error())
		} else {
			info.Status = uint8(score)
			return
		}
	}

	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(group_relation)
	err = c.Find(bson.M{"UId": info.UId, "GroupId": info.GroupId}).One(info)

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

//获取群所有成员id
func (info *GroupRelation) FindGroupMem() (Recodes []GroupMem, code int, err error) {
	conn := myredis.GetConn()
	var strKey string
	err = nil
	code = 0
	if conn.Err() != nil {
		common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
	} else {
		defer conn.Close()

		strKey = fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MEM_LIST, info.GroupId)
		results, errTemp := redis.Int64s(conn.Do("ZREVRANGEBYSCORE", strKey, "+inf", "-inf"))

		if errTemp != nil {
			common.Errorf("fail to find.uid=%d,groupid=%s,errTemp=%s", info.UId, info.GroupId, errTemp.Error())
		} else {
			for _, value := range results {
				Recodes = append(Recodes, GroupMem{uint64(value)})
			}
			return
		}
	}

	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(group_relation)
	err = c.Find(bson.M{"GroupId": info.GroupId}).All(&Recodes)

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

//获取群特定权限成员id
func (info *GroupRelation) FindGroupMemByRole(role uint32) (Recodes []GroupMem, code int, err error) {
	conn := myredis.GetConn()
	var strKey string
	err = nil
	code = 0
	if conn.Err() != nil {
		common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, conn.Err().Error())
	} else {
		defer conn.Close()

		strKey = fmt.Sprintf("%d:%d:%s", myredis.REDIS_T_SORTSET, myredis.IM_DATA_GROUP_MEM_LIST, info.GroupId)

		if (GROUP_ROLE_MEMBER | GROUP_ROLE_OWNER) == role {
			results, errTemp := redis.Int64Map(conn.Do("ZREVRANGEBYSCORE", strKey, "+inf", "-inf", "WITHSCORES"))
			if errTemp != nil {
				common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, errTemp.Error())
			} else {
				var tempUid int64
				for index, value := range results {
					if GROUP_MEMBER == value || GROUP_OWNER == value {
						tempUid, err = strconv.ParseInt(index, 10, 64)
						if nil != err {
							common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
							err = errTemp
							break
						}
						Recodes = append(Recodes, GroupMem{uint64(tempUid)})
					}
				}
				if nil == err {
					return
				}
			}
		} else {
			var results []int64
			switch role {
			case GROUP_ROLE_MEMBER:
				results, err = redis.Int64s(conn.Do("ZREVRANGEBYSCORE", strKey, "1", "1"))
			case GROUP_ROLE_ADMIN:
				results, err = redis.Int64s(conn.Do("ZREVRANGEBYSCORE", strKey, "2", "2"))
			case GROUP_ROLE_OWNER:
				results, err = redis.Int64s(conn.Do("ZREVRANGEBYSCORE", strKey, "3", "3"))
			case GROUP_ROLE_MEMBER | GROUP_ROLE_ADMIN:
				results, err = redis.Int64s(conn.Do("ZREVRANGEBYSCORE", strKey, "2", "1"))
			case GROUP_ROLE_OWNER | GROUP_ROLE_ADMIN:
				results, err = redis.Int64s(conn.Do("ZREVRANGEBYSCORE", strKey, "3", "2"))
			case GROUP_ROLE_MEMBER | GROUP_ROLE_OWNER | GROUP_ROLE_ADMIN:
				results, err = redis.Int64s(conn.Do("ZREVRANGEBYSCORE", strKey, "+inf", "-inf"))
			}

			if err != nil {
				common.Errorf("fail to find.uid=%d,groupid=%s,err=%s", info.UId, info.GroupId, err.Error())
			} else {
				for _, value := range results {
					Recodes = append(Recodes, GroupMem{uint64(value)})
				}
				return
			}
		}
	}

	mConn := mymongo.Conn()
	defer mConn.Close()

	c := mConn.DB(db).C(group_relation)
	switch role {
	case GROUP_ROLE_MEMBER:
		err = c.Find(bson.M{"GroupId": info.GroupId, "Status": 1}).All(&Recodes)
	case GROUP_ROLE_ADMIN:
		err = c.Find(bson.M{"GroupId": info.GroupId, "Status": 2}).All(&Recodes)
	case GROUP_ROLE_OWNER:
		err = c.Find(bson.M{"GroupId": info.GroupId, "Status": 3}).All(&Recodes)
	case GROUP_ROLE_MEMBER | GROUP_ROLE_ADMIN:
		err = c.Find(bson.M{"$or": []bson.M{bson.M{"GroupId": info.GroupId, "Status": 1}, bson.M{"GroupId": info.GroupId, "Status": 2}}}).All(&Recodes)
	case GROUP_ROLE_MEMBER | GROUP_ROLE_OWNER:
		err = c.Find(bson.M{"$or": []bson.M{bson.M{"GroupId": info.GroupId, "Status": 1}, bson.M{"GroupId": info.GroupId, "Status": 3}}}).All(&Recodes)
	case GROUP_ROLE_OWNER | GROUP_ROLE_ADMIN:
		err = c.Find(bson.M{"$or": []bson.M{bson.M{"GroupId": info.GroupId, "Status": 2}, bson.M{"GroupId": info.GroupId, "Status": 3}}}).All(&Recodes)
	case GROUP_ROLE_MEMBER | GROUP_ROLE_OWNER | GROUP_ROLE_ADMIN:
		err = c.Find(bson.M{"$or": []bson.M{bson.M{"GroupId": info.GroupId, "Status": 1}, bson.M{"GroupId": info.GroupId, "Status": 2}, bson.M{"GroupId": info.GroupId, "Status": 3}}}).All(&Recodes)
	}

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

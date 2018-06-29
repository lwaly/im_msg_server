package logic

//与其他子系统交互的业务处理文件
import (
	"encoding/json"
	"fmt"
	"im_msg_server/models/myredis"
	"time"

	"im_msg_server/common"
	"im_msg_server/models"

	"github.com/gomodule/redigo/redis"

	"gopkg.in/mgo.v2/bson"
)

const (
	MQ_CMD_RELATION = 12001 //关系变更功能
	WAIT_TIME       = 60
)

var mapUserRelationType map[uint8]string
var mapGroupRelationType map[uint8]string

func init() {
	mapUserRelationType = make(map[uint8]string)
	mapUserRelationType[models.USER_ATTENTION] = "0" //关注
	mapUserRelationType[models.USER_APPLY_FOR] = "1" //申请好友
	mapUserRelationType[models.USER_AGREE] = "2"     //同意添加
	mapUserRelationType[models.USER_REFUSE] = "3"    //拒绝
	mapUserRelationType[models.USER_DELETE] = "4"    //删除
	mapUserRelationType[models.USER_BLACKLIST] = "5" //拉黑

	//群通知：0:创建;1：申请;2:同意;3：拒绝；4:解散；5：主动退群；6：踢人；7：邀请；8：邀请审核同意；9：邀请审核拒绝
	mapGroupRelationType = make(map[uint8]string)
	mapGroupRelationType[GROUP_CREATE] = "0"
	mapGroupRelationType[GROUP_APPLY_FOR] = "1"
	mapGroupRelationType[GROUP_AGREE] = "2"
	mapGroupRelationType[GROUP_REFUSE] = "3"
	mapGroupRelationType[GROUP_DISSOLVE] = "4"
	mapGroupRelationType[GROUP_LEAVE] = "5"
	mapGroupRelationType[GROUP_KICK] = "6"
	mapGroupRelationType[GROUP_INVITE] = "7"
	mapGroupRelationType[GROUP_INVITE_AGREE] = "8"
	mapGroupRelationType[GROUP_INVITE_REFUSE] = "9"
	mapGroupRelationType[GROUP_INVITE_ENTER] = "10"
	mapGroupRelationType[GROUP_ROLE_MEMBER] = "29"
	mapGroupRelationType[GROUP_ROLE_ADMIN] = "30"
	mapGroupRelationType[GROUP_ROLE_OWNER] = "31"
}

const (
	MQ_OBJ_USER     = 1 //用户系统
	MQ_OBJ_RELATION = 2 //关系服务系统
	MQ_OBJ_MESSAGE  = 3 //消息系统
	MQ_OBJ_RELAY    = 4 //转发系统
)

//初始化队列消息
func InitQueue() {
	consumer()
}

//消费redis队列消息
func consumer() {
	strKey := fmt.Sprintf("%d:%d:%d:%d", myredis.REDIS_T_LIST, MQ_CMD_RELATION, MQ_OBJ_RELATION, MQ_OBJ_MESSAGE)
	for {
		conn := myredis.GetQueueConn()
		if conn.Err() != nil {
			common.Errorf(conn.Err().Error())
			continue
		}

		strMember, err := redis.Values(conn.Do("BRPOP", strKey, WAIT_TIME))
		if err != nil {
			if err != redis.ErrNil {
				common.Errorf("error: %v", err)
			}
			conn.Close()
			continue
		}

		if len(strMember) != 2 {
			common.Errorf("error: param wrong")
			conn.Close()
			continue
		}
		conn.Close()

		p := StQueueMessage{}
		if err := unpack(strMember[1].([]byte), &p); err != nil {
			continue
		}

		switch p.Cmd {
		case OP_RELATION_UPDATE:
			ProcessUserRelation(&p)
		case OP_GROUP_UPDATE:
			ProcessGroupRelation(&p)
		case OP_GROUP_ROLE_UPDATE:
			ProcessGroupRoleChange(&p)
		}
	}
}

// 处理用户之间关系变更
func ProcessUserRelation(p *StQueueMessage) {
	msg := StUserRelationBodyReq{}
	var code int
	var err error
	if err = json.Unmarshal(p.Body, &msg); err != nil {
		common.Errorf("err=%s code=%d", err.Error(), code)
		return
	}

	userRelation := models.UserRelation{
		UId:        msg.UId,
		ObjectId:   msg.ObjectId,
		Status:     msg.RelationType,
		UpdateTime: time.Now().UnixNano(),
	}
	userMsg := models.UserMsg{
		Id:          bson.NewObjectId(),
		UId:         msg.UId,
		ObjectId:    msg.ObjectId,
		MsgType:     models.NOTIFY,
		ContentType: msg.RelationType,
		Content:     mapUserRelationType[msg.RelationType],
		Status:      models.UNREAD,
		UpdateTime:  userRelation.UpdateTime,
		CreateTime:  userRelation.UpdateTime,
	}

	//处理关系变更
	switch msg.RelationType {
	case models.USER_ATTENTION, models.USER_APPLY_FOR:
		code, err = processUserRelationInsert(&userRelation)
	case models.USER_AGREE, models.USER_REFUSE, models.USER_DELETE, models.USER_BLACKLIST:
		userRelation.UId, userRelation.ObjectId = userRelation.ObjectId, userRelation.UId
		code, err = userRelation.Update()
	}
	if err != nil {
		common.Errorf("err=%s code=%d", err.Error(), code)
		return
	}
	//生成通知消息
	if code, err = userMsg.Insert(); err != nil {
		common.Errorf("err=%s code=%d", err.Error(), code)
	}
	return
}

//处理用户关系数据库插入
func processUserRelationInsert(userRelation *models.UserRelation) (code int, err error) {
	if code, err = userRelation.FindByField(); err != nil {
		if code == models.ErrNotFound {
			userRelation.Id = bson.NewObjectId()
			userRelation.CreateTime = userRelation.UpdateTime
			if code, err = userRelation.Insert(); err != nil {
				common.Errorf("err=%s code=%d", err.Error(), code)
				return
			}
		}
	} else {
		if code, err = userRelation.Update(); err != nil {
			common.Errorf("err=%s code=%d", err.Error(), code)
			return
		}
	}

	return
}

//处理群组关系变更
func ProcessGroupRelation(p *StQueueMessage) {
	msg := StGroupRelationBodyReq{}

	var code int
	var err error

	if err = json.Unmarshal(p.Body, &msg); err != nil {
		common.Errorf("err=%s code=%d", err.Error(), code)
		return
	}

	groupRelation := models.GroupRelation{
		GroupId:    msg.GroupId,
		UpdateTime: time.Now().UnixNano(),
	}

	//生成群通知消息
	groupMsg := models.GroupMsg{
		UId:         msg.UId,
		GroupId:     msg.GroupId,
		MsgType:     models.NOTIFY,
		ContentType: msg.Type,
		Status:      models.VALID,
		UpdateTime:  groupRelation.UpdateTime,
		CreateTime:  groupRelation.UpdateTime,
	}

	var Recodes []models.GroupMem
	//获取通知列表
	switch msg.Type {
	case GROUP_AGREE, GROUP_DISSOLVE, GROUP_LEAVE, GROUP_KICK, GROUP_INVITE_AGREE, GROUP_INVITE_ENTER:
		if Recodes, code, err = groupRelation.FindGroupMemByRole(models.GROUP_ROLE_MEMBER | models.GROUP_ROLE_OWNER | models.GROUP_ROLE_ADMIN); nil != err {
			break
		}
		//申请通知管理员与群主
	case GROUP_APPLY_FOR, GROUP_REFUSE, GROUP_INVITE, GROUP_INVITE_REFUSE:
		if Recodes, code, err = groupRelation.FindGroupMemByRole(models.GROUP_ROLE_OWNER | models.GROUP_ROLE_ADMIN); nil != err {
			break
		}
	}
	if err != nil {
		common.Errorf("err=%s code=%d", err.Error(), code)
		return
	}

	switch msg.Type {
	case GROUP_APPLY_FOR:
		groupMsg.Id = bson.NewObjectId()
		groupMsg.Content = fmt.Sprintf("%d:%s", msg.UId, msg.Content)
		if code, err = groupMsg.Insert(); err != nil {
			common.Errorf("err=%s code=%d", err.Error(), code)
			break
		}

		groupUserMsg := models.GroupUserMsg{
			UId:        msg.UId,
			GroupId:    msg.GroupId,
			MsgId:      groupMsg.Id.Hex(),
			Type:       groupMsg.MsgType,
			Status:     models.UNREAD,
			CreateTime: groupRelation.UpdateTime,
			UpdateTime: groupRelation.UpdateTime,
		}
		if code, err = groupUserMsg.InsertUnRead(Recodes); nil != err {
			common.Errorf("err=%s code=%d", err.Error(), code)
			break
		}
		groupUserMsg.Status = models.READ
		if code, err = groupUserMsg.InsertRead(); nil != err {
			common.Errorf("err=%s code=%d", err.Error(), code)
			break
		}
	case GROUP_DISSOLVE:
		groupMsg.Id = bson.NewObjectId()
		groupMsg.Content = fmt.Sprintf("%d:%s", msg.UId, msg.Content)
		if code, err = groupMsg.Insert(); err != nil {
			common.Errorf("err=%s code=%d", err.Error(), code)
			break
		}

		groupUserMsg := models.GroupUserMsg{
			UId:        msg.UId,
			GroupId:    msg.GroupId,
			MsgId:      groupMsg.Id.Hex(),
			Type:       groupMsg.MsgType,
			Status:     models.UNREAD,
			CreateTime: groupRelation.UpdateTime,
			UpdateTime: groupRelation.UpdateTime,
		}
		if code, err = groupUserMsg.InsertUnRead(Recodes); nil != err {
			common.Errorf("err=%s code=%d", err.Error(), code)
			break
		}
		groupUserMsg.Status = models.READ
		if code, err = groupUserMsg.InsertRead(); nil != err {
			common.Errorf("err=%s code=%d", err.Error(), code)
			break
		}
	case GROUP_INVITE_ENTER:
		//被执行人添加未读消息
		groupMsg.Id = bson.NewObjectId()
		groupMsg.Content = fmt.Sprintf("%d:%s", msg.UId, msg.Content)
		if code, err = groupMsg.Insert(); err != nil {
			common.Errorf("err=%s code=%d", err.Error(), code)
			break
		}

		groupUserMsg := models.GroupUserMsg{
			UId:        msg.UId,
			GroupId:    msg.GroupId,
			MsgId:      groupMsg.Id.Hex(),
			Type:       groupMsg.MsgType,
			Status:     models.UNREAD,
			CreateTime: groupRelation.UpdateTime,
			UpdateTime: groupRelation.UpdateTime,
		}
		//
		Recodes = append(Recodes, models.GroupMem{msg.UId})
		if code, err = groupUserMsg.InsertUnRead(Recodes); nil != err {
			common.Errorf("err=%s code=%d", err.Error(), code)
			break
		}
		groupUserMsg.Status = models.READ
		if code, err = groupUserMsg.InsertRead(); nil != err {
			common.Errorf("err=%s code=%d", err.Error(), code)
			break
		}
	case GROUP_LEAVE:
		groupMsg.Id = bson.NewObjectId()
		groupMsg.Content = fmt.Sprintf("%d:%s", msg.UId, msg.Content)
		if code, err = groupMsg.Insert(); err != nil {
			common.Errorf("err=%s code=%d", err.Error(), code)
			break
		}

		groupUserMsg := models.GroupUserMsg{
			UId:        msg.UId,
			GroupId:    msg.GroupId,
			MsgId:      groupMsg.Id.Hex(),
			Type:       groupMsg.MsgType,
			Status:     models.UNREAD,
			CreateTime: groupRelation.UpdateTime,
			UpdateTime: groupRelation.UpdateTime,
		}
		if code, err = groupUserMsg.InsertUnRead(Recodes); nil != err {
			common.Errorf("err=%s code=%d", err.Error(), code)
			break
		}
		groupUserMsg.Status = models.READ
		if code, err = groupUserMsg.InsertRead(); nil != err {
			common.Errorf("err=%s code=%d", err.Error(), code)
			break
		}
	default:
		for _, value := range msg.List {
			groupMsg.Id = bson.NewObjectId()
			groupMsg.Content = fmt.Sprintf("%d:%s", value.UId, msg.Content)
			if code, err = groupMsg.Insert(); err != nil {
				common.Errorf("err=%s code=%d", err.Error(), code)
				break
			}

			groupUserMsg := models.GroupUserMsg{
				UId:        msg.UId,
				GroupId:    msg.GroupId,
				MsgId:      groupMsg.Id.Hex(),
				Type:       groupMsg.MsgType,
				Status:     models.UNREAD,
				CreateTime: groupRelation.UpdateTime,
				UpdateTime: groupRelation.UpdateTime,
			}

			//群组成员被动接受的消息
			switch msg.Type {
			//同时进群，全体获取通知
			case GROUP_CREATE:
				if code, err = groupUserMsg.InsertUnRead(msg.List); nil != err {
					common.Errorf("err=%s code=%d", err.Error(), code)
					break
				}
			case GROUP_AGREE, GROUP_INVITE_AGREE:
				//被执行人添加未读消息
				if code, err = groupUserMsg.InsertUnRead(Recodes); nil != err {
					common.Errorf("err=%s code=%d", err.Error(), code)
					break
				}
				Recodes = append(Recodes, value) //上一个人进群，下一次群消息需得到通知
				groupUserMsg.Status = models.READ
				if code, err = groupUserMsg.InsertRead(); nil != err {
					common.Errorf("err=%s code=%d", err.Error(), code)
					break
				}
			case GROUP_KICK:
				if code, err = groupUserMsg.InsertUnRead(Recodes); nil != err {
					common.Errorf("err=%s code=%d", err.Error(), code)
					break
				}
				groupUserMsg.Status = models.READ
				if code, err = groupUserMsg.InsertRead(); nil != err {
					common.Errorf("err=%s code=%d", err.Error(), code)
					break
				}
				//剔除离开群成员
				for index1, value1 := range Recodes {
					if value.UId == value1.UId {
						Recodes = append(Recodes[:index1], Recodes[index1+1:]...)
					}
				}

				//拒绝，邀请拒绝
			case GROUP_REFUSE, GROUP_INVITE_REFUSE:
				if code, err = groupUserMsg.InsertUnRead(Recodes); nil != err {
					common.Errorf("err=%s code=%d", err.Error(), code)
					break
				}
				//都未进群，需单独添加未读消息
				if code, err = groupUserMsg.InsertUnRead([]models.GroupMem{value}); nil != err {
					common.Errorf("err=%s code=%d", err.Error(), code)
					break
				}
				groupUserMsg.Status = models.READ
				if code, err = groupUserMsg.InsertRead(); nil != err {
					common.Errorf("err=%s code=%d", err.Error(), code)
					break
				}
			case GROUP_INVITE:
				if code, err = groupUserMsg.InsertUnRead(Recodes); nil != err {
					common.Errorf("err=%s code=%d", err.Error(), code)
					break
				}
				//都未进群，需单独添加未读消息
				if code, err = groupUserMsg.InsertUnRead([]models.GroupMem{value}); nil != err {
					common.Errorf("err=%s code=%d", err.Error(), code)
					break
				}
				groupUserMsg.Status = models.READ
				if code, err = groupUserMsg.InsertRead(); nil != err {
					common.Errorf("err=%s code=%d", err.Error(), code)
					break
				}
			}

			if err != nil {
				common.Errorf("err=%s code=%d", err.Error(), code)
				return
			}
		}
	}

	//群关系变更处理
	switch msg.Type {
	//建群
	case GROUP_CREATE:
		groupRelation.Status = models.GROUP_MEMBER
		groupRelation.CreateTime = groupRelation.UpdateTime
		if code, err = groupRelation.Insert(msg.List); nil != err {
			common.Errorf("err=%s code=%d", err.Error(), code)
			break
		}
		groupRelation.Status = models.GROUP_OWNER
		code, err = groupRelation.Insert([]models.GroupMem{models.GroupMem{msg.UId}})
	//加群
	case GROUP_AGREE, GROUP_INVITE_AGREE, GROUP_INVITE_ENTER:
		groupRelation.Status = models.GROUP_MEMBER
		groupRelation.CreateTime = groupRelation.UpdateTime
		code, err = groupRelation.Insert(msg.List)
		//退出群
	case GROUP_LEAVE, GROUP_KICK:
		code, err = groupRelation.DelGroupMem(msg.List)
	//解散群
	case GROUP_DISSOLVE:
		groupRelation.UId = msg.List[0].UId
		code, err = groupRelation.DelGroup()
	}
	if err != nil {
		common.Errorf("err=%s code=%d", err.Error(), code)
	}
	return
}

//处理群内成员角色变更
func ProcessGroupRoleChange(p *StQueueMessage) {
	msg := StGroupRoleChange{}

	var code int
	var err error

	if err = json.Unmarshal(p.Body, &msg); err != nil {
		common.Errorf("err=%s code=%d", err.Error(), code)
		return
	}

	groupRelation := models.GroupRelation{
		GroupId:    msg.GroupId,
		UpdateTime: time.Now().UnixNano(),
	}

	//生成群通知消息
	groupMsg := models.GroupMsg{
		UId:        msg.UId,
		GroupId:    msg.GroupId,
		MsgType:    models.NOTIFY,
		Status:     models.VALID,
		UpdateTime: groupRelation.UpdateTime,
		CreateTime: groupRelation.UpdateTime,
	}
	//加一与数据库权限位一致
	msg.Role++
	switch msg.Role {
	case models.GROUP_MEMBER:
		groupMsg.ContentType = GROUP_ROLE_MEMBER
	case models.GROUP_ADMIN:
		groupMsg.ContentType = GROUP_ROLE_ADMIN
	case models.GROUP_OWNER:
		groupMsg.ContentType = GROUP_ROLE_OWNER
	}

	var Recodes []models.GroupMem
	if Recodes, code, err = groupRelation.FindGroupMemByRole(models.GROUP_ROLE_MEMBER | models.GROUP_ROLE_OWNER | models.GROUP_ROLE_ADMIN); nil != err {
		common.Errorf("err=%s code=%d", err.Error(), code)
		return
	}

	if err != nil {
		common.Errorf("err=%s code=%d", err.Error(), code)
		return
	}

	for _, value := range msg.List {
		groupMsg.Id = bson.NewObjectId()
		groupMsg.Content = fmt.Sprintf("%d:%s", value.UId, msg.Content)
		if code, err = groupMsg.Insert(); err != nil {
			common.Errorf("err=%s code=%d", err.Error(), code)
			break
		}

		groupUserMsg := models.GroupUserMsg{
			GroupId:    msg.GroupId,
			MsgId:      groupMsg.Id.Hex(),
			Type:       groupMsg.MsgType,
			Status:     models.UNREAD,
			CreateTime: groupRelation.UpdateTime,
			UpdateTime: groupRelation.UpdateTime,
		}
		if code, err = groupUserMsg.InsertUnRead(Recodes); nil != err {
			common.Errorf("err=%s code=%d", err.Error(), code)
			break
		}
	}
	//群内角色变更处理

	groupRelation.Status = msg.Role
	if code, err = groupRelation.Update(msg.List); nil != err {
		common.Errorf("err=%s code=%d", err.Error(), code)
	}

	return
}

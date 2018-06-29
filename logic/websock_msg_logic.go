package logic

//处理客户端逻辑
import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"im_msg_server/common"
	"im_msg_server/models"

	"github.com/gorilla/websocket"
	"gopkg.in/mgo.v2/bson"
)

const PAGE_SIZE_MAX = 30

var upgrader = websocket.Upgrader{}

//接收客户端消息
func ProcessClient(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		common.Errorf("upgrade err:%v", err)
		http.Error(w, "error token (code: 1)", http.StatusBadRequest)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			common.Errorf("read err:%v", err)
			break
		}

		head := StQueueMessage{}
		if err := unpack(message, &head); err != nil {
			continue
		}

		switch head.Cmd {
		case UserMsgAdd:
			ProcessUserMsgAdd(&head, message[10:], mt, c)
		case UserMsgUnreadGet:
			ProcessUserMsgUnreadGet(&head, message[10:], mt, c)
		case UserMsgCountGet:
			ProcessUserMsgCountGet(&head, message[10:], mt, c)
		case UserMsgGet:
			ProcessUserMsgGet(&head, message[10:], mt, c)
		case GroupMsgAdd:
			ProcessGroupMsgAdd(&head, message[10:], mt, c)
		case GroupMsgUnreadGet:
			ProcessGroupMsgUnreadGet(&head, message[10:], mt, c)
		case GroupMsgCountGet:
			ProcessGroupMsgCountGet(&head, message[10:], mt, c)
		case GroupMsgGet:
			ProcessGroupMsgGet(&head, message[10:], mt, c)
		}
	}
}

//发送消息给客户端
func SendMsg(head *StQueueMessage, rspCode int, info error, msg []byte, mt int, c *websocket.Conn) (code int, err error) {
	rsp := StMessageRsp{head.Version, head.Cmd + 1, head.Seq, rspCode, info.Error(), msg}

	tempMsg, _ := json.Marshal(rsp)
	err = c.WriteMessage(mt, tempMsg)
	if err != nil {
		common.Errorf("SendMsg err:%v", err)
	}
	return
}

//发送处理出错消息给客户端
func SendErrMsg(head *StQueueMessage, rspCode int, info error, mt int, c *websocket.Conn) (code int, err error) {
	var msg []byte
	rsp := StMessageRsp{head.Version, head.Cmd + 1, head.Seq, rspCode, info.Error(), msg}

	tempMsg, _ := json.Marshal(rsp)
	err = c.WriteMessage(mt, tempMsg)
	if err != nil {
		common.Errorf("SendMsg err:%v", err)
	}
	return
}

//处理消息添加
func ProcessUserMsgAdd(head *StQueueMessage, msg []byte, mt int, c *websocket.Conn) (code int, err error) {
	req := StUserMsgAddReq{}
	err = json.Unmarshal(head.Body, &req)
	if nil != err {
		common.Errorf("parameter err")
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("parameter err"), mt, c)
		return
	}

	if "" == req.Content || models.CONTENT_TYPE_MIN >= req.ContentType || models.CONTENT_TYPE_MAX <= req.ContentType {
		common.Errorf("parameter err.uid=%d,objectid=%d,content=%s,content_type=%d.", req.UId, req.ObjectId, req.Content, req.ContentType)
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("parameter err"), mt, c)
		return
	}

	_, err = common.Token_auth(req.Token)
	if err != nil {
		common.Errorf("error token")
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("error token"), mt, c)
		return
	}

	info := models.UserMsg{
		UId:         req.UId,
		ObjectId:    req.ObjectId,
		Content:     req.Content,
		ContentType: req.ContentType,
	}

	userRelation := models.UserRelation{
		UId:      info.UId,
		ObjectId: info.ObjectId,
	}
	if code, err = userRelation.FindByField(); nil != err {
		SendErrMsg(head, code, err, mt, c)
		return
	}

	if models.USER_AGREE != userRelation.Status {
		common.Errorf("parameter err.uid=%d,objectid=%d,content=%s,content_type=%d.", info.UId, info.ObjectId, info.Content, info.ContentType)
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("parameter err"), mt, c)
		return
	}

	info.Id = bson.NewObjectId()
	info.CreateTime = time.Now().UnixNano()
	info.UpdateTime = info.CreateTime
	info.Status = models.UNREAD
	info.MsgType = models.MESSAGE

	if code, err = info.Insert(); nil != err {
		SendErrMsg(head, code, err, mt, c)
		return
	}

	msg, err = json.Marshal(StUserMsgAddRsp{Id: info.Id.Hex()})
	code, err = SendMsg(head, common.OK, errors.New("ok"), msg, mt, c)
	if err != nil {
		common.Errorf("SendMsg err:%v.uid=%d,objectid=%d,content=%s,content_type=%d.", err, info.UId, info.ObjectId, info.Content, info.ContentType)
		return
	}
	return
}

//处理未读消息获取
func ProcessUserMsgUnreadGet(head *StQueueMessage, msg []byte, mt int, c *websocket.Conn) (code int, err error) {
	req := StUserMsgUnreadGetReq{}
	err = json.Unmarshal(head.Body, &req)

	if nil != err {
		common.Errorf("parameter err")
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("parameter err"), mt, c)
		return
	}

	if PAGE_SIZE_MAX < req.PageSize || 0 >= req.PageSize {
		common.Errorf("parameter err.uid=%d,pagesize=%d.", req.UId, req.PageSize)
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("parameter err"), mt, c)
		return
	}

	_, err = common.Token_auth(req.Token)
	if err != nil {
		common.Errorf("error token.uid=%d,pagesize=%d.", req.UId, req.PageSize)
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("error token"), mt, c)
		return
	}

	info := models.UserMsg{
		Status:   models.UNREAD,
		ObjectId: req.UId,
	}

	info.MsgType = models.MESSAGE

	if records, code, err := info.FindUserUnreadMsg(req.PageSize, 1); nil != err {
		SendErrMsg(head, code, err, mt, c)
		return code, err
	} else {
		if count, code, err := info.FindUserMsgUnreadCount(); nil != err {
			SendErrMsg(head, code, err, mt, c)
			return code, err
		} else {
			if int32(count)-req.PageSize > 0 {
				msg, err = json.Marshal(StUserMsgUnreadGetRsp{Total: int32(count) - req.PageSize, Data: records})
			} else {
				msg, err = json.Marshal(StUserMsgUnreadGetRsp{Total: int32(0), Data: records})
			}

			code, err = SendMsg(head, common.OK, errors.New("ok"), msg, mt, c)
			if err != nil {
				common.Errorf("SendMsg err:%v.uid=%d,pagesize=%d.", err, req.UId, req.PageSize)
				return code, err
			} else {
				if code, err = info.Updates(records); nil != err {
					common.Errorf("updates err:%v", err)
				}
				return code, err
			}
		}
	}
}

//处理已读消息获取
func ProcessUserMsgGet(head *StQueueMessage, msg []byte, mt int, c *websocket.Conn) (code int, err error) {
	req := StUserMsgGetReq{}
	err = json.Unmarshal(head.Body, &req)

	if nil != err {
		common.Errorf("parameter err")
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("parameter err"), mt, c)
		return
	}

	if PAGE_SIZE_MAX < req.PageSize || req.PageSize <= 0 || req.PageIndex < 0 {
		common.Errorf("parameter err.uid=%d,objectid=%d,PageSize=%d,PageIndex=%d.", req.UId, req.ObjectId, req.PageSize, req.PageIndex)
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("parameter err"), mt, c)
		return
	}

	_, err = common.Token_auth(req.Token)
	if err != nil {
		common.Errorf("error token.uid=%d,objectid=%d,PageSize=%d,PageIndex=%d.", req.UId, req.ObjectId, req.PageSize, req.PageIndex)
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("error token"), mt, c)
		return
	}

	info := models.UserMsg{
		Status:   models.READ,
		UId:      req.UId,
		ObjectId: req.ObjectId,
	}

	info.MsgType = models.MESSAGE

	if records, code, err := info.FindUserMsg(req.PageSize, req.PageIndex); nil != err {
		SendErrMsg(head, code, err, mt, c)
		return code, err
	} else {
		if count, code, err := info.FindUserMsgCount(); nil != err {
			SendErrMsg(head, code, err, mt, c)
			return code, err
		} else {
			msg, err = json.Marshal(StUserMsgGetRsp{Total: int32(count), Data: records})
			code, err = SendMsg(head, common.OK, errors.New("ok"), msg, mt, c)
			if err != nil {
				common.Errorf("SendMsg err:%v.uid=%d,objectid=%d,PageSize=%d,PageIndex=%d.", err, req.UId, req.ObjectId, req.PageSize, req.PageIndex)
			}
			return code, err
		}
	}
}

//处理已读消息数获取
func ProcessUserMsgCountGet(head *StQueueMessage, msg []byte, mt int, c *websocket.Conn) (code int, err error) {
	req := StUserMsgCountGetReq{}
	err = json.Unmarshal(head.Body, &req)

	if nil != err {
		common.Errorf("parameter err.")
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("parameter err"), mt, c)
		return
	}

	_, err = common.Token_auth(req.Token)
	if err != nil {
		common.Errorf("error token.uid=%d,objectid=%d.", req.UId, req.ObjectId)
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("error token"), mt, c)
		return
	}

	info := models.UserMsg{
		Status:   models.READ,
		UId:      req.UId,
		ObjectId: req.ObjectId,
	}
	info.MsgType = models.MESSAGE

	if count, code, err := info.FindUserMsgCount(); nil != err {
		SendErrMsg(head, code, err, mt, c)
		return code, err
	} else {
		msg, err = json.Marshal(StUserMsgCountGetRsp{Total: int32(count)})
		code, err = SendMsg(head, common.OK, errors.New("ok"), msg, mt, c)
		if err != nil {
			common.Errorf("SendMsg err:%v.uid=%d,objectid=%d.", err, req.UId, req.ObjectId)
		}
		return code, err
	}
}

//处理群消息添加
func ProcessGroupMsgAdd(head *StQueueMessage, msg []byte, mt int, c *websocket.Conn) (code int, err error) {
	req := StGroupMsgAddReq{}
	err = json.Unmarshal(head.Body, &req)

	if nil != err {
		common.Errorf("parameter err")
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("parameter err"), mt, c)
		return
	}

	if "" == req.Content || models.CONTENT_TYPE_MIN >= req.ContentType || models.CONTENT_TYPE_MAX <= req.ContentType || "" == req.GroupId {
		common.Errorf("parameter err.uid=%d,GroupId=%s,content=%s,content_type=%d.", req.UId, req.GroupId, req.Content, req.ContentType)
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("parameter err"), mt, c)
		return
	}

	_, err = common.Token_auth(req.Token)
	if err != nil {
		common.Errorf("error token")
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("error token"), mt, c)
		return
	}

	info := models.GroupMsg{
		UId:         req.UId,
		GroupId:     req.GroupId,
		ContentType: req.ContentType,
		Content:     req.Content,
	}
	userInfo := models.GroupRelation{
		UId:     info.UId,
		GroupId: info.GroupId,
	}

	//验证是否有这人
	if code, err = userInfo.FindByField(); nil != err {
		SendErrMsg(head, code, err, mt, c)
	}

	info.Id = bson.NewObjectId()
	info.CreateTime = time.Now().UnixNano()
	info.UpdateTime = info.CreateTime
	info.Status = models.VALID
	info.MsgType = models.MESSAGE

	if code, err = info.Insert(); nil != err {
		SendErrMsg(head, code, err, mt, c)
		return
	}

	var recodes []models.GroupMem
	groupRelation := models.GroupRelation{
		GroupId: info.GroupId,
	}

	if recodes, code, err = groupRelation.FindGroupMem(); nil != err {
		SendErrMsg(head, code, err, mt, c)
		return
	} else {
		groupread := models.GroupUserMsg{
			UId:        info.UId,
			GroupId:    info.GroupId,
			MsgId:      info.Id.Hex(),
			Type:       models.MESSAGE,
			Status:     models.UNREAD,
			CreateTime: info.CreateTime,
			UpdateTime: info.CreateTime,
		}

		if code, err = groupread.InsertUnRead(recodes); nil != err {
			SendErrMsg(head, code, err, mt, c)
			return
		}
		groupread.Status = models.READ
		if code, err = groupread.InsertRead(); nil != err {
			SendErrMsg(head, code, err, mt, c)
			return
		}
	}

	msg, err = json.Marshal(StGroupMsgAddRsp{Id: info.Id.Hex()})
	code, err = SendMsg(head, common.OK, errors.New("ok"), msg, mt, c)
	if err != nil {
		common.Errorf("SendMsg err:%v.uid=%d,GroupId=%s,content=%s,content_type=%d.", err, req.UId, req.GroupId, req.Content, req.ContentType)
		return
	}
	return
}

//处理未读消息获取
func ProcessGroupMsgUnreadGet(head *StQueueMessage, msg []byte, mt int, c *websocket.Conn) (code int, err error) {
	req := StGroupMsgUnreadGetReq{}
	err = json.Unmarshal(head.Body, &req)

	if nil != err {
		common.Errorf("parameter err")
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("parameter err"), mt, c)
		return
	}

	if PAGE_SIZE_MAX < req.PageSize || req.PageSize < 0 {
		common.Errorf("parameter err.uid=%d,pagesize=%d.", req.UId, req.PageSize)
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("parameter err"), mt, c)
		return
	}

	_, err = common.Token_auth(req.Token)
	if err != nil {
		common.Errorf("error token.uid=%d,pagesize=%d.", req.UId, req.PageSize)
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("error token"), mt, c)
		return
	}

	info := models.GroupUserMsg{
		UId: req.UId,
	}

	if records, code, err := info.FindUserUnreadGroupMsg(int(req.PageSize)); nil != err {
		SendErrMsg(head, code, err, mt, c)
		return code, err
	} else {
		if count, code, err := info.FindUserUnreadGroupMsgCount(); nil != err {
			SendErrMsg(head, code, err, mt, c)
			return code, err
		} else {
			groupMsg := models.GroupMsg{}
			if groupMsgRecords, code, err := groupMsg.FindUnReadMsgByMsgId(records); nil != err {
				SendErrMsg(head, code, err, mt, c)
				return code, err
			} else {
				if int32(count) > req.PageSize {
					msg, err = json.Marshal(StGroupMsgUnreadGetRsp{Total: int32(count) - req.PageSize, Data: groupMsgRecords})
				} else {
					msg, err = json.Marshal(StGroupMsgUnreadGetRsp{Total: int32(0), Data: groupMsgRecords})
				}
				code, err = SendMsg(head, common.OK, errors.New("ok"), msg, mt, c)
				if err != nil {
					common.Errorf("SendMsg err:%v.uid=%d,pagesize=%d.", err, req.UId, req.PageSize)
					return code, err
				} else {
					if code, err = info.Update(records); nil != err {
						common.Errorf("updates err:%v.uid=%d,pagesize=%d.", err, req.UId, req.PageSize)
					}
					return code, err
				}
			}
		}
	}
}

//处理已读消息获取
func ProcessGroupMsgGet(head *StQueueMessage, msg []byte, mt int, c *websocket.Conn) (code int, err error) {
	req := StGroupMsgGetReq{}
	err = json.Unmarshal(head.Body, &req)

	if nil != err {
		common.Errorf("parameter err")
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("parameter err"), mt, c)
		return
	}

	if PAGE_SIZE_MAX < req.PageSize || req.PageSize <= 0 || req.PageIndex < 0 || "" == req.GroupId {
		common.Errorf("parameter err.uid=%d,GroupId=%s,PageSize=%d,PageIndex=%d.", req.UId, req.GroupId, req.PageSize, req.PageIndex)
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("parameter err"), mt, c)
		return
	}

	_, err = common.Token_auth(req.Token)
	if err != nil {
		common.Errorf("error token.uid=%d,GroupId=%s,PageSize=%d,PageIndex=%d.", req.UId, req.GroupId, req.PageSize, req.PageIndex)
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("error token"), mt, c)
		return
	}

	info := models.GroupUserMsg{
		UId:     req.UId,
		GroupId: req.GroupId,
	}

	if records, code, err := info.FindUserGroupMsg(req.PageSize, req.PageIndex); nil != err {
		SendErrMsg(head, code, err, mt, c)
		return code, err
	} else {
		if count, code, err := info.FindUserGroupMsgCount(); nil != err {
			SendErrMsg(head, code, err, mt, c)
			return code, err
		} else {
			groupMsg := models.GroupMsg{}
			if groupMsgRecords, code, err := groupMsg.FindReadMsgByMsgId(records); nil != err {
				SendErrMsg(head, code, err, mt, c)
				return code, err
			} else {
				msg, err = json.Marshal(StGroupMsgGetRsp{Total: int32(count), Data: groupMsgRecords})
				code, err = SendMsg(head, common.OK, errors.New("ok"), msg, mt, c)
				if err != nil {
					common.Errorf("SendMsg err:%v.uid=%d,GroupId=%s,PageSize=%d,PageIndex=%d.", err, req.UId, req.GroupId, req.PageSize, req.PageIndex)
				}
				return code, err
			}
		}
	}
}

//处理已读消息数获取
func ProcessGroupMsgCountGet(head *StQueueMessage, msg []byte, mt int, c *websocket.Conn) (code int, err error) {
	req := StGroupMsgCountGetReq{}
	err = json.Unmarshal(head.Body, &req)

	if nil != err {
		common.Errorf("parameter err")
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("parameter err"), mt, c)
		return
	}

	if "" == req.GroupId {
		common.Errorf("parameter err.uid=%d,GroupId=%s.", req.UId, req.GroupId)
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("parameter err"), mt, c)
		return
	}

	_, err = common.Token_auth(req.Token)
	if err != nil {
		common.Errorf("error token")
		SendErrMsg(head, common.ERRINPUTDATA, errors.New("error token"), mt, c)
		return
	}

	info := models.GroupUserMsg{
		UId:     req.UId,
		GroupId: req.GroupId,
	}

	if count, code, err := info.FindUserGroupMsgCount(); nil != err {
		SendErrMsg(head, code, err, mt, c)
		return code, err
	} else {
		msg, err = json.Marshal(StUserMsgCountGetRsp{Total: int32(count)})
		code, err = SendMsg(head, common.OK, errors.New("ok"), msg, mt, c)
		if err != nil {
			common.Errorf("SendMsg err:%v.uid=%d,GroupId=%s.", err, req.UId, req.GroupId)
		}
		return code, err
	}
}

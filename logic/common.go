package logic

import (
	"encoding/json"
	"im_msg_server/common"
)

const (
	UserMsgAdd        = 2001
	UserMsgUnreadGet  = 2003
	UserMsgGet        = 2005
	UserMsgCountGet   = 2007
	GroupMsgAdd       = 2009
	GroupMsgUnreadGet = 2011
	GroupMsgGet       = 2013
	GroupMsgCountGet  = 2015
)

//群通知：0:创建;1：申请;2:同意;3：拒绝；4:解散；5：主动退群；6：踢人；7：邀请；8：邀请审核同意；9：邀请审核拒绝；10:直接申请进群不需审核；29：角色变更普通成员；30：角色变更普通成员；31：角色变更普通成员
const (
	GROUP_CREATE        = 0
	GROUP_APPLY_FOR     = 1
	GROUP_AGREE         = 2
	GROUP_REFUSE        = 3
	GROUP_DISSOLVE      = 4
	GROUP_LEAVE         = 5
	GROUP_KICK          = 6
	GROUP_INVITE        = 7
	GROUP_INVITE_AGREE  = 8
	GROUP_INVITE_REFUSE = 9
	GROUP_INVITE_ENTER  = 10
	GROUP_ROLE_MEMBER   = 29
	GROUP_ROLE_ADMIN    = 30
	GROUP_ROLE_OWNER    = 31
)

type StQueueMessage struct {
	Version int16           `json:"ver" valid:"Required"`
	Cmd     int32           `json:"cmd" valid:"Required"`
	Seq     int32           `json:"seq" valid:"Required"`
	Body    json.RawMessage `json:"body"`
}

type StMessageRsp struct {
	Version int16           `json:"ver" valid:"Required"`
	Cmd     int32           `json:"cmd" valid:"Required"`
	Seq     int32           `json:"seq" valid:"Required"`
	Code    int             `json:"Code" valid:"Required"`
	Info    string          `json:"Info" valid:"Required"`
	Body    json.RawMessage `json:"body"`
}

//解包消息包
func unpack(sData []byte, p *StQueueMessage) (err error) {
	if p == nil || len(sData) == 0 {
		common.Errorf("msg queue data invalid")
		return
	}

	if err = json.Unmarshal(sData, p); err != nil {
		common.Errorf("err=%s", err.Error())
		return
	}
	return
}

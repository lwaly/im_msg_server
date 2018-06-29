package logic

//与其他子系统交互的tcp协议结构体
import (
	"im_msg_server/models"
)

//tcp协议结构体
const (
	OP_RELATION_UPDATE     = int32(1001)
	OP_RELATION_UPDATE_ACK = int32(1002)

	OP_GROUP_UPDATE     = int32(1003)
	OP_GROUP_UPDATE_ACK = int32(1004)

	OP_GROUP_ROLE_UPDATE     = int32(1005)
	OP_GROUP_ROLE_UPDATE_ACK = int32(1006)
)

type StUserRelationBodyReq struct {
	MsgFlag      string `json:"msg_flag" valid:"Required"`
	RelationType uint8  `json:"relation_type"    valid:"Required"`
	UId          uint64 `json:"userId"    valid:"Required"`
	ObjectId     uint64 `json:"object_id"    valid:"Required"`
	Content      string `json:"message"`
	Ver          string `json:"data_ver"`
}

type StUserRelationReq struct {
	Version int16                 `json:"ver" valid:"Required"`
	Cmd     int32                 `json:"cmd" valid:"Required"`
	Seq     int32                 `json:"seq" valid:"Required"`
	Body    StUserRelationBodyReq `json:"body"`
}

type StRelationBodyRsp struct {
	MsgFlag string `json:"msg_flag" valid:"Required"`
	Code    int    `json:"code"    valid:"Required"`
	Info    string `json:"info"`
}

type StUserRelationRsp struct {
	Version int16             `json:"ver" valid:"Required"`
	Cmd     int32             `json:"cmd" valid:"Required"`
	Seq     int32             `json:"seq" valid:"Required"`
	Body    StRelationBodyRsp `json:"body"`
}

type StGroupRelationBodyReq struct {
	MsgFlag     string            `json:"msg_flag" valid:"Required"`
	GroupType   uint8             `json:"group_type"    valid:"Required"`
	GroupId     string            `json:"group_id"    valid:"Required"`
	UId         uint64            `json:"userid"    valid:"Required"`
	Type        uint8             `json:"type"    valid:"Required"`
	Content     string            `bson:"message" valid:"Required"`
	List        []models.GroupMem `json:"list"    valid:"Required"`
	DataVersion string            `json:"data_ver" valid:"Required"`
}

type StGroupRelationReq struct {
	Version int16                  `json:"ver" valid:"Required"`
	Cmd     int32                  `json:"cmd" valid:"Required"`
	Seq     int32                  `json:"seq" valid:"Required"`
	Body    StGroupRelationBodyReq `json:"body"`
}

type StGroupRoleChange struct {
	MsgFlag   string            `json:"msg_flag"`
	GroupType uint8             `json:"group_type"`
	GroupId   string            `json:"group_id"`
	UId       uint64            `json:"userid"`
	Role      uint8             `json:"role"`
	Content   string            `json:"message"`
	Ver       string            `json:"data_ver"`
	List      []models.GroupMem `json:"list"`
}

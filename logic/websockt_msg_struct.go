package logic

//与客户端交互协议结构体定义文档
import (
	"im_msg_server/models"
)

//websock协议结构体
type StMsgRsp struct {
	Info string `json:"InfoId" valid:"Required"`
	Code int32  `json:"Code" valid:"Required"`
}

//消息添加
type StUserMsgAddReq struct {
	UId         uint64 `json:"UId"    valid:"Required"`
	ObjectId    uint64 `json:"ObjectId" valid:"Required"`
	ContentType uint8  `json:"ContentType"    valid:"Required"`
	Content     string `json:"Content" valid:"Required"`
	Token       string `json:"Token" valid:"Required"`
}

type StUserMsgAddRsp struct {
	Id string `json:"Id" valid:"Required"`
}

//未读消息获取
type StUserMsgUnreadGetReq struct {
	UId      uint64 `json:"UId"    valid:"Required"`
	PageSize int32  `json:"PageSize" valid:"Required"`
	Token    string `json:"Token" valid:"Required"`
}

type StUserMsgUnreadGetRsp struct {
	Total int32                    `json:"Total" valid:"Required"`
	Data  []models.StUserUnreadMsg `json:"Data" valid:"Required"`
}

//已读消息获取
type StUserMsgGetReq struct {
	UId       uint64 `json:"UId"    valid:"Required"`
	ObjectId  uint64 `json:"ObjectId"    valid:"Required"`
	PageSize  int32  `json:"PageSize" valid:"Required"`
	PageIndex int32  `json:"PageIndex" valid:"Required"`
	Token     string `json:"Token" valid:"Required"`
}

type StUserMsgGetRsp struct {
	Total int32              `json:"Total" valid:"Required"`
	Data  []models.StUserMsg `json:"Data" valid:"Required"`
}

//已读消息数获取
type StUserMsgCountGetReq struct {
	UId      uint64 `json:"UId"    valid:"Required"`
	ObjectId uint64 `json:"ObjectId"    valid:"Required"`
	Token    string `json:"Token" valid:"Required"`
}

type StUserMsgCountGetRsp struct {
	Total int32 `json:"Total" valid:"Required"`
}

//群消息
//消息添加
type StGroupMsgAddReq struct {
	UId         uint64 `json:"UId"    valid:"Required"`
	GroupId     string `json:"GroupId" valid:"Required"`
	ContentType uint8  `json:"ContentType"    valid:"Required"`
	Content     string `json:"Content" valid:"Required"`
	Token       string `json:"Token" valid:"Required"`
}

type StGroupMsgAddRsp struct {
	Id string `json:"Id" valid:"Required"`
}

//未读消息获取
type StGroupMsgUnreadGetReq struct {
	UId      uint64 `json:"UId"    valid:"Required"`
	PageSize int32  `json:"PageSize" valid:"Required"`
	Token    string `json:"Token" valid:"Required"`
}

type StGroupMsgUnreadGetRsp struct {
	Total int32                     `json:"Total" valid:"Required"`
	Data  []models.StGroupMsgUnread `json:"Data" valid:"Required"`
}

//已读消息数获取
type StGroupMsgCountGetReq struct {
	UId     uint64 `json:"UId"    valid:"Required"`
	GroupId string `json:"GroupId"    valid:"Required"`
	Token   string `json:"Token" valid:"Required"`
}

type StGroupMsgCountGetRsp struct {
	Total int32 `json:"Total" valid:"Required"`
}

//已读消息获取
type StGroupMsgGetReq struct {
	UId       uint64 `json:"UId"    valid:"Required"`
	GroupId   string `json:"GroupId"    valid:"Required"`
	PageSize  int32  `json:"PageSize" valid:"Required"`
	PageIndex int32  `json:"PageIndex" valid:"Required"`
	Token     string `json:"Token" valid:"Required"`
}

type StGroupMsgGetRsp struct {
	Total int32               `json:"Total" valid:"Required"`
	Data  []models.StGroupMsg `json:"Data" valid:"Required"`
}

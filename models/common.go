package models

// Predefined model error codes.
const (
	ErrDatabase = -1
	ErrSystem   = -2
	ErrDupRows  = -3
	ErrNotFound = -4
	ErrInput    = -5
)

const (
	db             = "im_msg_server"
	user_relation  = "user_relation"
	group_relation = "group_relation"
	group_msg      = "group_msg"
	group_user_msg = "group_user_msg"
	user_msg       = "user_msg"
)

const (
	MESSAGE = 1
	NOTIFY  = 2
)

//消息内容类型1：文本；2：图片；3：文件；
const (
	CONTENT_TYPE_MIN = 0
	TEXT             = 1
	PICTURE          = 2
	FILE             = 3
	CONTENT_TYPE_MAX = 4
)

//用户关系变更类型。0：关注；1：申请添加好友；2：同意添加好友；3：拒绝好友；4：删除；5：拉黑名单
const (
	USER_ATTENTION = 0
	USER_APPLY_FOR = 1
	USER_AGREE     = 2
	USER_REFUSE    = 3
	USER_DELETE    = 4
	USER_BLACKLIST = 5
)

//群内容身份0：普通成员；1：管理员；2：群主
const (
	GROUP_MEMBER = 1
	GROUP_ADMIN  = 2
	GROUP_OWNER  = 3
)

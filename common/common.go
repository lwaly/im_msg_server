package common

import (
	"crypto/md5"
	"encoding/base64"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type ControllerError struct {
	Status   int    `json:"status"`
	Code     int    `json:"code"`
	Message  string `json:"message"`
	DevInfo  string `json:"dev_info"`
	MoreInfo string `json:"more_info"`
}

var (
	Err404          = &ControllerError{404, 404, "page not found", "page not found", ""}
	ErrInputData    = &ControllerError{400, 10001, "数据输入错误", "客户端参数错误", ""}
	ErrDatabase     = &ControllerError{500, 10002, "服务器错误", "数据库操作错误", ""}
	ErrDupUser      = &ControllerError{400, 10003, "用户信息已存在", "数据库记录重复", ""}
	ErrNoUser       = &ControllerError{400, 10004, "用户信息不存在", "数据库记录不存在", ""}
	ErrPass         = &ControllerError{400, 10005, "用户信息不存在或密码不正确", "密码不正确", ""}
	ErrNoUserPass   = &ControllerError{400, 10006, "用户信息不存在或密码不正确", "数据库记录不存在或密码不正确", ""}
	ErrNoUserChange = &ControllerError{400, 10007, "用户信息不存在或数据未改变", "数据库记录不存在或数据未改变", ""}
	ErrInvalidUser  = &ControllerError{400, 10008, "用户信息不正确", "Session信息不正确", ""}
	ErrOpenFile     = &ControllerError{500, 10009, "服务器错误", "打开文件出错", ""}
	ErrWriteFile    = &ControllerError{500, 10010, "服务器错误", "写文件出错", ""}
	ErrSystem       = &ControllerError{500, 10011, "服务器错误", "操作系统错误", ""}
	ErrExpired      = &ControllerError{400, 10012, "登录已过期", "验证token过期", ""}
	ErrPermission   = &ControllerError{400, 10013, "没有权限", "没有操作权限", ""}
	Actionsuccess   = &ControllerError{200, 90000, "操作成功", "操作成功", ""}
)

const (
	OK              = 0
	ERRINPUTDATA    = 10001
	ERRDATABASE     = 10002
	ERRDUPUSER      = 10003
	ERRNOUSER       = 10004
	ERRPASS         = 10005
	ERRNOUSERPASS   = 10006
	ERRNOUSERCHANGE = 10007
	ERRINVALIDUSER  = 10008
	ERROPENFILE     = 10009
	ERRWRITEFILE    = 10010
	ERRSYSTEM       = 10011
	ERREXPIRED      = 10012
	ERRPERMISSION   = 10013
	ACTIONSUCCESS   = 90000
)

const (
	ErrSend = 1
)

const (
	Select_all_user = "查找全部用户"
	secret          = "test"
)

type Claims struct {
	//Appid string `json:"Appid"`
	// recommended having
	Userid int `json:"Userid"`
	jwt.StandardClaims
}

func Base64Encode(src []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(src))
}
func To_md5(encode string) (decode string) {
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(encode))
	cipherStr := md5Ctx.Sum(nil)
	return string(Base64Encode(cipherStr))
}

func Create_token(id int) string {
	expireToken := time.Now().Add(time.Hour * 24).Unix()
	claims := Claims{
		//info.Appid,
		id,
		jwt.StandardClaims{
			ExpiresAt: expireToken,
		},
	}

	// Create the token using your claims
	c_token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Signs the token with a secret.
	signedToken, _ := c_token.SignedString([]byte(secret))

	return signedToken
}

func Token_auth(signedToken string) (int, error) {
	token, err := jwt.ParseWithClaims(signedToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		//fmt.Printf("%v %v", claims.Username, claims.StandardClaims.ExpiresAt)
		//fmt.Println(reflect.TypeOf(claims.StandardClaims.ExpiresAt))
		//return claims.Appid, err
		return claims.Userid, err
	}
	return 0, err
}

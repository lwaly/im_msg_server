package mymongo

import (
	"im_msg_server/common"

	"gopkg.in/mgo.v2"
)

var session *mgo.Session

// Conn return mongodb session.
func Conn() *mgo.Session {
	return session.Copy()
}

/*
func Close() {
	session.Close()
}
*/

func init() {

	if str1, err := common.Conf.GetValue("mongodb", "url"); nil != err {
		panic(err)
	} else {
		session, err = mgo.Dial(str1)
		if err != nil {
			panic(err)
		}
		session.SetMode(mgo.Monotonic, true)
	}
}

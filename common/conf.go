package common

import "im_msg_server/common/conf"

var Conf *goconfig.ConfigFile

func init() {
	var err error
	Conf, err = goconfig.LoadConfigFile("./conf/conf")
	if err != nil {
		panic(err)
	}
}

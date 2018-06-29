package main

import (
	"flag"
	"fmt"
	"net/http"

	"im_msg_server/common"
	"im_msg_server/logic"
)

func main() {
	//日志系统初始化
	str, err := common.Conf.GetValue("log", "path")
	if nil != err {
		fmt.Printf("fail to get log path")
		return
	}
	defer common.Start(common.LogFilePath(str), common.EveryHour).Stop()

	//客户端监听启动
	httpAddr, err := common.Conf.GetValue("http", "addr")
	if nil != err {
		common.Errorf("error: %v %s", err, httpAddr)
		return
	}
	var addr = flag.String("addr", httpAddr, "http service address")
	flag.Parse()
	http.HandleFunc("/im/msg_server", logic.ProcessClient)
	go http.ListenAndServe(*addr, nil)

	//队列初始化启动
	logic.InitQueue()
}

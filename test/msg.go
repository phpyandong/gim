package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/phpyandong/gim/model"
	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "127.0.0.1:3101", "http service address")

func main() {
	appNameFlag := flag.String("app", "", "请输入要启动的app名称")
	flag.Parse()

	//App名称
	appName := *appNameFlag
	switch appName {
	case "all":
		{
			stop := make(chan os.Signal)
			go boardcast("1", "ios")
			go climsg("2", "ios", "ios", 1)
			go joingrp("3", "ios", 99)
			go joingrp("3", "android", 99)
			go grpmsg("4", "ios", 99)
			go climsg("5", "ios", "android", 1)
			go boardcast("1", "android")
			<-stop
			return
		}
	case "1":
		{
			//1号ios设备,全员广播
			boardcast("1", "ios")
		}
	case "2":
		{
			//2号ios设备向1号ios设备发送私聊
			climsg("2", "ios", "ios", 1)
		}
	case "3":
		{
			//3号ios设备，加入99号群,只接受消息
			joingrp("3", "ios", 99)
		}
	case "4":
		{
			//3号android设备，加入99号群,只接受消息
			joingrp("3", "android", 99)
		}
	case "5":
		{
			//4号ios设备，加入99号群，并且发送消息
			grpmsg("4", "ios", 99)
		}
	case "6":
		{
			//5号ios设备向1号ios设备发送私聊
			climsg("5", "ios", "android", 1)
		}
	case "7":
		{
			//1号android设备,全员广播
			boardcast("1", "android")
		}

	case "33":
		{
			//2号ios设备向1号ios设备发送私聊
			climsg("33", "ios", "ios", 1)
		}
	}
}

//广播
func boardcast(token, tag string) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	vals := url.Values{}
	vals.Add("token", token)
	vals.Add("tag", tag)
	u.RawQuery = vals.Encode()
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s,%v", message, time.Now())
		}
	}()
	// j, _ := json.Marshal(&model.DTO{
	// 	Type: model.CliJoinGrp,
	// 	Msg: &model.Msg{
	// 		GroupID: 1,
	// 	},
	// })
	// err = c.WriteMessage(websocket.TextMessage, j)
	// if err != nil {
	// 	log.Println("write:", err)
	// 	return
	// }

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// 广播
			{
				j, _ := json.Marshal(&model.DTO{
					Type: model.CliBcastMsg,
					Msg: &model.Msg{
						Content: fmt.Sprintf("这里是%s号,%s设备,在发送广播消息...", token, tag),
					},
				})
				err := c.WriteMessage(websocket.TextMessage, j)
				if err != nil {
					log.Println("write:", err)
					return
				}
			}
		case <-interrupt:
			return
		}
	}
}

//单播
func climsg(token, tag, toUserTag string, toUserID int64) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	vals := url.Values{}
	vals.Add("token", token)
	vals.Add("tag", tag)
	u.RawQuery = vals.Encode()
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s,%v", message, time.Now())
		}
	}()
	// j, _ := json.Marshal(&model.DTO{
	// 	Type: model.CliJoinGrp,
	// 	Msg: &model.Msg{
	// 		GroupID: 1,
	// 	},
	// })
	// err = c.WriteMessage(websocket.TextMessage, j)
	// if err != nil {
	// 	log.Println("write:", err)
	// 	return
	// }

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// 广播
			{
				j, _ := json.Marshal(&model.DTO{
					Type: model.CliCliMsg,
					Msg: &model.Msg{
						ToUserID:  toUserID,
						ToUserTag: toUserTag,
						Content:   fmt.Sprintf("这里是%s号,%s设备,在向%d号,%s设备发送单播消息...", token, tag, toUserID, toUserTag),
					},
				})
				err := c.WriteMessage(websocket.TextMessage, j)
				if err != nil {
					log.Println("write:", err)
					return
				}
			}
		case <-interrupt:
			return
		}
	}
}

//组播
func grpmsg(token, tag string, grpid int64) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	vals := url.Values{}
	vals.Add("token", token)
	vals.Add("tag", tag)
	u.RawQuery = vals.Encode()
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s,%v", message, time.Now())
		}
	}()
	j, _ := json.Marshal(&model.DTO{
		Type: model.CliJoinGrp,
		Msg: &model.Msg{
			GroupID: grpid,
		},
	})
	err = c.WriteMessage(websocket.TextMessage, j)
	if err != nil {
		log.Println("write:", err)
		return
	}

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			//组播
			{
				j, _ := json.Marshal(&model.DTO{
					Type: model.CliGrpMsg,
					Msg: &model.Msg{
						GroupID: grpid,
						Content: fmt.Sprintf("这里是%s号%s设备,在向%d号群发送群消息...", token, tag, grpid),
					},
				})
				err := c.WriteMessage(websocket.TextMessage, j)
				if err != nil {
					log.Println("write:", err)
					return
				}
			}
		case <-interrupt:
			return
		}
	}
}

//加入群
func joingrp(token, tag string, grpid int64) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	vals := url.Values{}
	vals.Add("token", token)
	vals.Add("tag", tag)
	u.RawQuery = vals.Encode()
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s,%v", message, time.Now())
		}
	}()
	j, _ := json.Marshal(&model.DTO{
		Type: model.CliJoinGrp,
		Msg: &model.Msg{
			GroupID: grpid,
		},
	})
	err = c.WriteMessage(websocket.TextMessage, j)
	if err != nil {
		log.Println("write:", err)
		return
	}

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			//组播
			// {
			// 	j, _ := json.Marshal(&model.DTO{
			// 		Type: model.CliGrpMsg,
			// 		Msg: &model.Msg{
			// 			GroupID: grpid,
			// 			Content: fmt.Sprintf("这里是%s号%s设备,在向%d号群发送群消息...", token, tag, grpid),
			// 		},
			// 	})
			// 	err := c.WriteMessage(websocket.TextMessage, j)
			// 	if err != nil {
			// 		log.Println("write:", err)
			// 		return
			// 	}
			// }
		case <-interrupt:
			return
		}
	}
}

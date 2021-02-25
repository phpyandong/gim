package logic

import (
	"encoding/json"
	"log"
	"time"

	"github.com/phpyandong/gim/model"
	"github.com/gorilla/websocket"
)
//logic server中的comet 客户端
type comet struct {
	logic  *logic	//该logic服务端
	conn   *websocket.Conn
	ch     chan *model.DTO  //消息体
	stop   chan error
	source string
}

func newcomet(l *logic, conn *websocket.Conn, source string) *comet {
	return &comet{
		logic:  l,
		conn:   conn,
		ch:     make(chan *model.DTO),
		stop:   make(chan error),
		source: source,
	}
}
//启动服务的方法
func (c *comet) run() {
	go c.recv() //启动一个goroutinue接收消息
	go c.consume()	//启动一个goroutinue 消费消息
	go c.hb()		//启动一个goroutinue 进行心跳检测
	<-c.stop
}
//logic 服务接收comet服务的消息
func (c *comet) recv() {
	for {
		j := &model.DTO{}
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("logic recv error:%v", err)
			c.logic.comets.Delete(c.conn.RemoteAddr().String())
			return
		}
		json.Unmarshal(message, j)
		ch := c.logic.getch()
		ch <- j
	}
}
//消费消息，包括发送到regist服务的心跳检测
func (c *comet) consume() {
	for {
		select {
		case dto := <-c.ch:
			{
				var err error
				//lgc recv hb from reg
				if dto.Type == model.RegHbToLgc {
					log.Print("lgc recv hb from reg")
				}
				//lgc recv hb from cmt
				if dto.Type == model.CmtHbToLgc {
					log.Print("lgc recv hb from cmt")
				}
				//lgc send cli msg to cmt
				if dto.Type == model.LgcCliMsg {
					j, _ := json.Marshal(dto)
					err = c.conn.WriteMessage(websocket.TextMessage, j)
					log.Printf("logic send cli msg to %s:%s, error:%v", c.source, c.conn.RemoteAddr().String(), err)
				}
				//lgc send grp msg to cmt
				if dto.Type == model.LgcGrpMsg {
					j, _ := json.Marshal(dto)
					err = c.conn.WriteMessage(websocket.TextMessage, j)
					log.Printf("logic send grp msg to %s:%s, error:%v", c.source, c.conn.RemoteAddr().String(), err)
				}
				//lgc send bcast msg to cmt
				if dto.Type == model.LgcBcastMsg {
					j, _ := json.Marshal(dto)
					err = c.conn.WriteMessage(websocket.TextMessage, j)
					log.Printf("logic send bcast msg to %s:%s, error:%v", c.source, c.conn.RemoteAddr().String(), err)
				}
				//lgc send hb to cmt or reg，心跳检测消息的处理
				if dto.Type == model.LgcHbToCmt || dto.Type == model.LgcHbToReg {
					j, _ := json.Marshal(dto)
					err = c.conn.WriteMessage(websocket.TextMessage, j)
					if err != nil {
						log.Printf("logic send heart beat to %s:%s, error:%v", c.source, c.conn.RemoteAddr().String(), err)
						if c.source == "registry" {
							c.logic.isregistered = false
						}
						return
					}
					log.Printf("send heart beat to %s:%s", c.source, c.conn.RemoteAddr().String())
				}
				if c.source == "comet" && err != nil {
					c.logic.comets.Delete(c.conn.RemoteAddr().String())
					return
				}
			}
		}
	}
}
func (c *comet) hb() {
	hb := c.logic.hbcomet
	tp := model.LgcHbToCmt
	if c.source == "registry" {
		hb = c.logic.hbregistry
		tp = model.LgcHbToReg
	}
	t := time.NewTicker(time.Second * time.Duration(hb))
	for {
		select {
		case <-t.C:
			{
				dto := &model.DTO{
					Type: tp,
					Msg:  nil,
				}
				c.ch <- dto
			}
		}
	}
}

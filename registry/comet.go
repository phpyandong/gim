package registry

import (
	"encoding/json"
	"log"

	"github.com/phpyandong/gim/model"
	"github.com/gorilla/websocket"
)

type comet struct {
	conn *websocket.Conn
	r    *registry
	ch   chan *model.DTO
}

func (c *comet) consume() {
	for {
		select {
		case dto := <-c.ch:
			{
				var err error
				//reg recv from comet
				if dto.Type == model.CmtHbToReg {
					log.Print("recv heart beat from comet")
				}
				//reg send hb to comet
				if dto.Type == model.RegHbToCmt {
					j, _ := json.Marshal(dto)
					err = c.conn.WriteMessage(websocket.TextMessage, j)
					log.Printf("registry send heart beat to comet:%s,err:%v ", c.conn.RemoteAddr().String(), err)
				}
				//reg bcast lgc svrs to comet
				if dto.Type == model.RegBcastLgcSvrs {
					j, _ := json.Marshal(dto)
					err = c.conn.WriteMessage(websocket.TextMessage, j)
					log.Printf("registry boardcast logic servers:%s, to comet:%s,err:%v", dto.Msg.Content, c.conn.RemoteAddr().String(), err)
				}
				if err != nil {
					c.r.comets.Delete(c.conn.RemoteAddr().String())
				}
			}
		}
	}
}

func (c *comet) recv() {
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			c.r.comets.Delete(c.conn.RemoteAddr().String())
			return
		}
		j := &model.DTO{}
		err = json.Unmarshal(message, j)
		if err == nil {
			c.ch <- j
		}
	}
}

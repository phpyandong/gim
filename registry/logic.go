package registry

import (
	"encoding/json"
	"log"
	"net/url"
	"time"

	"github.com/phpyandong/gim/model"
	"github.com/gorilla/websocket"
)

type logic struct {
	conn *websocket.Conn
	r    *registry
	ch   chan *model.DTO
}

func (l *logic) dial(addr string) {
	//连接
	u := url.URL{Scheme: "ws", Host: addr, Path: "/ws"}
	vals := url.Values{}
	vals.Add("t", "registry")
	u.RawQuery = vals.Encode()
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("registry dial err:%v ", err)
		return
	}
	log.Println("dial lgc success:", u.String())
	l.r.logics.Store(c.RemoteAddr().String(), &logic{
		conn: c,
	})
	defer c.Close()

	//消费消息
	go func() {
		for {
			_, message, err := c.ReadMessage() //msg is hb
			log.Printf("registry recv logic msg:%s,logic conns:%v", string(message), l.r.logics)
			if err != nil {
				l.r.logics.Delete(c.RemoteAddr().String())
				log.Printf("registry recv logic msg:%s,logic conns:%v,err:%v", string(message), l.r.logics, err)
				return
			}
		}
	}()
	//心跳
	{
		t := time.NewTicker(time.Second * time.Duration(l.r.hblogic))
		defer t.Stop()
		//心跳
		for {
			select {
			case <-t.C:
				msg, _ := json.Marshal(&model.DTO{
					Type: model.RegHbToLgc,
					Msg:  nil,
				})
				err := c.WriteMessage(websocket.TextMessage, msg)
				log.Printf("registry send to logic:%s heart beat", c.RemoteAddr().String())
				if err != nil {
					log.Printf("registry send to logic:%s heart beat,err:%v", c.RemoteAddr().String(), err)
					l.r.logics.Delete(c.RemoteAddr().String())
					return
				}
			}
		}
	}
}

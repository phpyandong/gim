package comet

import (
	"encoding/json"
	"log"
	"net/url"

	"github.com/phpyandong/gim/model"
	"github.com/gorilla/websocket"
)

type logic struct {
	comet     *comet
	logicaddr string
	conn      *websocket.Conn
	ch        chan *model.DTO
}

func newlogic(addr string, c *comet) *logic {
	return &logic{
		comet:     c,
		logicaddr: addr,
		ch:        make(chan *model.DTO),
	}
}

func (l *logic) run() {
	//dial
	u := url.URL{Scheme: "ws", Host: l.logicaddr, Path: "/ws"}
	vals := url.Values{}
	vals.Add("t", "comet")
	u.RawQuery = vals.Encode()

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("comet dial logic server err:%v ", err)
		return
	}
	defer c.Close()
	l.conn = c
	l.comet.logics.LoadOrStore(c.RemoteAddr().String(), l)
	go l.send()
	//recv
	for {
		j := &model.DTO{}
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Printf("comet recv msg from logic error:%v", err)
			l.comet.logics.Delete(c.RemoteAddr().String())
			return
		}
		json.Unmarshal(message, j)
		ch := l.comet.getch() //notice: comet ch
		ch <- j
	}
}

func (l *logic) send() {
	for {
		select {
		case dto := <-l.ch:
			{
				//cmt msg to lgc
				// log.Printf("comet send msg to logic :%s", l.conn.RemoteAddr().String())
				j, _ := json.Marshal(dto)
				err := l.conn.WriteMessage(websocket.TextMessage, j)
				if err != nil {
					log.Printf("comet send msg to logic :%s,err:%v", l.conn.RemoteAddr().String(), err)
					l.comet.logics.Delete(l.conn.RemoteAddr().String())
					return
				}
			}
		}
	}
}

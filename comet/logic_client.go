package comet

import (
	"encoding/json"
	"log"
	"net/url"

	"github.com/phpyandong/gim/model"
	"github.com/gorilla/websocket"
)
//comet下logic的client
type logiclient struct {
	comet     *cometServer  //绑定的comet服务
	logicaddr string
	conn      *websocket.Conn
	ch        chan *model.DTO
}

func NewLogicClient(addr string, c *cometServer) *logiclient {
	return &logiclient{
		comet:     c,
		logicaddr: addr,
		ch:        make(chan *model.DTO),
	}
}
//启动一个logic客服端,中转，将comet数据转发给 logic 服务
func (l *logiclient) run() {
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
			l.comet.logics.Delete(c.RemoteAddr().String())//服务出错。
			// 则将本地存储的logic server ip 删除
			return
		}
		json.Unmarshal(message, j)
		ch := l.comet.getch() //notice: comet ch
		ch <- j //接收到消息后通过channel转发给comet；
		// todo 这里是往channal里添加数据，
		// 消费代码在哪里，如何才能快速找到呢？
	}
}
//将comet消息转发给logic
func (l *logiclient) send() {
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

package registry

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/phpyandong/gim/model"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}
//本地版的注册中心，后期使用压缩字典树实现
type registry struct {
	port          string `json:"addr"`
	stop          chan error
	logics        sync.Map
	comets        sync.Map
	hbcastlgcsvrs int64
	hblogic       int64
	hbcomet       int64
}

func New(conf *model.Conf) *registry {
	return &registry{
		port:          conf.Registry.Port,
		stop:          make(chan error),
		logics:        sync.Map{},
		comets:        sync.Map{},
		hbcastlgcsvrs: conf.Registry.HBcastLgcSvrs,
		hbcomet:       conf.Registry.HBComet,
		hblogic:       conf.Registry.HBLogic,
	}
}

func (r *registry) Run() {
	go func() {
		http.HandleFunc("/ws", func(w http.ResponseWriter, req *http.Request) {
			r.serve(w, req)
		})
		r.stop <- http.ListenAndServe(fmt.Sprintf(":%s", r.port), nil)
	}()
	go r.bcastcomet()
	go r.comethb()
	<-r.stop
}

func (r *registry) serve(w http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Println("cli upgrade err:", err)
		return
	}
	defer conn.Close()
	query := req.URL.Query()
	t := query.Get("t")
	if t == "" || (t != "comet" && t != "logic") {
		return
	}

	//recv comet cli
	if t == "comet" {
		addr := conn.RemoteAddr().String()
		c := &comet{
			conn: conn,
			r:    r,
			ch:   make(chan *model.DTO),
		}
		r.comets.LoadOrStore(addr, c)
		go c.recv()
		c.consume()
	}

	//dail logic server
	if t == "logic" {
		port := query.Get("port")
		if port == "" {
			return
		}
		host := strings.Split(conn.RemoteAddr().String(), ":")[0]
		l := &logic{
			conn: conn,
			r:    r,
			ch:   make(chan *model.DTO),
		}
		go l.dial(fmt.Sprintf("%s:%s", host, port))
	}

}
//广播地址 comet
func (r *registry) bcastcomet() {
	t := time.NewTicker(time.Second * time.Duration(r.hbcastlgcsvrs))
	for {
		select {
		case <-t.C:
			{
				// reg boardcast logic to comet
				s := make([]string, 0)
				r.logics.Range(func(key, value interface{}) bool {
					if v, ok := key.(string); ok {
						s = append(s, v)
					}
					return true
				})
				if len(s) > 0 {
					dto := &model.DTO{
						Type: model.RegBcastLgcSvrs,
						Msg: &model.Msg{
							Content: strings.Join(s, ","),
						},
					}
					r.comets.Range(func(key, value interface{}) bool {
						if v, ok := value.(*comet); ok {
							v.ch <- dto
						}
						return true
					})
				}
			}
		}
	}
}
//comet 的心跳检测
func (r *registry) comethb() {
	t := time.NewTicker(time.Second * time.Duration(r.hbcomet))
	for {
		select {
		case <-t.C:
			{
				//reg hb  to comet
				dto := &model.DTO{
					Type: model.RegHbToCmt,
					Msg:  nil,
				}
				r.comets.Range(func(key, value interface{}) bool {
					if v, ok := value.(*comet); ok {
						v.ch <- dto
					}
					return true
				})
			}
		}
	}
}

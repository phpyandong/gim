package logic

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/phpyandong/gim/model"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{} // use default options

type logic struct {
	port         string
	stop         chan error
	isregistered bool
	chs          []chan *model.DTO //recv ( logic or cli ) msg chs
	chsCnt       int64             //recv ( logic or cli ) msg ch cnt
	comets       sync.Map
	hbregistry   int64
	hbcomet      int64
	hbwatchreg   int64
}

func New(conf *model.Conf) *logic {
	l := &logic{
		port:         conf.Logic.Port,
		stop:         make(chan error),
		isregistered: false,
		chs:          make([]chan *model.DTO, 0),
		chsCnt:       2,
		comets:       sync.Map{},
		hbregistry:   conf.Logic.HBRegistry,
		hbcomet:      conf.Logic.HBComet,
		hbwatchreg:   conf.Logic.HBWatchReg,
	}
	regAddr := fmt.Sprintf("%s:%s", conf.Registry.Host, conf.Registry.Port)
	lgcSvrPort := fmt.Sprintf("%s", conf.Logic.Port)
	go l.watchreg(regAddr, lgcSvrPort)
	return l
}
//运行logic服务
func (l *logic) Run() {

	for i := int64(0); i < l.chsCnt; i++ {
		l.chs = append(l.chs, make(chan *model.DTO))
		go l.consume(l.chs[i])
	}

	go func() {
		http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
			l.serve(w, r)
		})
		http.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello world"))
		})
		l.stop <- http.ListenAndServe(fmt.Sprintf(":%s", l.port), nil)
	}()

	<-l.stop
}
//启动服务
func (l *logic) serve(w http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Println("cli upgrade:", err)
		return
	}
	defer conn.Close()
	addr := conn.RemoteAddr().String()
	query := req.URL.Query()
	t := query.Get("t")
	c := newcomet(l, conn, t)

	if t == "comet" {
		l.comets.LoadOrStore(addr, c)
	}
	if t == "registry" {
		l.isregistered = true
	}
	c.run()
}

//consume msg 消费消息
func (l *logic) consume(ch chan *model.DTO) {
	for {
		select {
		case dto := <-ch:
			{
				if dto.Type == model.RegHbToLgc {
					log.Print("lgc recv hb from reg")
				}
				if dto.Type == model.CmtHbToLgc {
					log.Print("lgc recv hb from cmt")
				}
				//cli cli msg
				if dto.Type == model.CliCliMsg {
					dto.Type = model.LgcCliMsg
					l.comets.Range(func(key, value interface{}) bool {
						if cli, ok := value.(*comet); ok {
							cli.ch <- dto
						}
						return true
					})
				}
				//cli grp msg
				if dto.Type == model.CliGrpMsg {
					dto.Type = model.LgcGrpMsg
					l.comets.Range(func(key, value interface{}) bool {
						if cli, ok := value.(*comet); ok {
							cli.ch <- dto
						}
						return true
					})
				}
				//cli bacast msg
				if dto.Type == model.CliBcastMsg {
					dto.Type = model.LgcBcastMsg
					l.comets.Range(func(key, value interface{}) bool {
						if cli, ok := value.(*comet); ok {
							cli.ch <- dto
						}
						return true
					})
				}
			}
		}
	}
}

//rand ch
func (l *logic) getch() (ch chan *model.DTO) {
	ch = l.chs[rand.Int63n(l.chsCnt-1)]
	return
}

//lgc watch reg
func (l *logic) watchreg(regAddr, lgcSvrPort string) {
	t := time.NewTicker(time.Second * time.Duration(l.hbwatchreg))
	for {
		select {
		case <-t.C:
			{
				if !l.isregistered {
					u := url.URL{Scheme: "ws", Host: regAddr, Path: "/ws"}
					vals := url.Values{}
					vals.Add("t", "logic")
					vals.Add("port", lgcSvrPort)
					u.RawQuery = vals.Encode()
					_, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
					if err != nil {
						log.Printf("logic dial registry err:%v ,url:%s", err, u.String())
					}
				}
			}
		}
	}
}

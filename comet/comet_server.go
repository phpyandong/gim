package comet

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/phpyandong/gim/model"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{} // use default options

type cometServer struct { //Bucket

	regaddr      string	//comet服务的地址
	port         string
	stop         chan error	//该服务停止信号



	cligroup     sync.Map //bucket cli groups (sync.map->groupid:[]*client)
						//comet 服务 存储群组id对应的client
	clibkts      []*clibkt  //一个comet服务对应多个client  重要
	chs          []chan *model.DTO //recv ( logic or cli ) msg chs
	chsCnt       int64             //recv ( logic or cli ) msg ch cnt

	logics       sync.Map //logic conns (sync.map->addr:logic) //维护ip对应的logic服务
	logicCnt     int64
	clibckcnt    int64    //config cli bucket size
	cliCnt       int64    //client cnt
	isregistered bool	//是否注册

	hbregistry   int64
	hblogic      int64		//与logic服务的心跳检测
	hbclient     int64		//一个心跳检测的客户端
	hbwatchreg   int64		//xin
	grprw        sync.RWMutex
}
type clibkt struct {
	rw      sync.RWMutex
	clients sync.Map // client conns (sync.map->uid:[]*client) key to a channel
}

func NewCometServer(conf *model.Conf) *cometServer {
	if conf == nil || conf.Registry == nil || conf.Registry.Host == "" {
		panic("conf argument error")
	}
	addr := fmt.Sprintf("%s:%s", conf.Registry.Host, conf.Registry.Port)
	c := &cometServer{
		cliCnt:       0,
		clibckcnt:    conf.Comet.CliBckCnt,
		cligroup:     sync.Map{},
		logics:       sync.Map{},
		logicCnt:     0,
		stop:         make(chan error),
		regaddr:      addr,
		isregistered: false,
		chs:          make([]chan *model.DTO, 0),
		chsCnt:       10,
		port:         conf.Comet.Port,
		hbregistry:   conf.Comet.HBRegistry,
		hblogic:      conf.Comet.HBLogic,
		hbclient:     conf.Comet.HBClient,
		hbwatchreg:   conf.Comet.HBWatchReg,
		grprw:        sync.RWMutex{},
		clibkts:      make([]*clibkt, 0),
	}
	for i := int64(0); i < conf.Comet.CliBckCnt; i++ {
		c.clibkts = append(c.clibkts, &clibkt{
			rw:      sync.RWMutex{},
			clients: sync.Map{},
		})
	}
	return c
}
//启动comet服务
func (c *cometServer) Run() {
	go func() {
		http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
			Serve(w, r, c)
		})
		c.stop <- http.ListenAndServe(fmt.Sprintf(":%s", c.port), nil)
	}()

	go c.registry() //注册到发现服务

	go c.watchreg()	//检测是否注册到服务。否则重新注册

	go c.recv()	//接收数据

	go c.hb()	//心跳检测

	go c.statistics()

	<-c.stop
}

//batch recv msg
func (c *cometServer) recv() {
	if c.chsCnt == 0 {
		c.chsCnt = 1024
	}
	c.chs = make([]chan *model.DTO, 0)
	for i := int64(0); i < c.chsCnt; i++ {
		c.chs = append(c.chs, make(chan *model.DTO))
		go c.consume(c.chs[i])
	}
}

//conn to  registry
//comet 连接注册服务
func (comet *cometServer) registry() {
	//连接
	u := url.URL{Scheme: "ws", Host: comet.regaddr, Path: "/ws"}
	vals := url.Values{}
	vals.Add("t", "comet")
	u.RawQuery = vals.Encode()
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("registry dial err:%v ", err)
		return
	}
	defer c.Close()
	comet.isregistered = true

	//消费消息
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Printf("comet recv registry msg error:%v", err)
				comet.isregistered = false
				return
			}
			j := &model.DTO{}
			json.Unmarshal(message, j)
			ch := comet.getch()
			ch <- j
		}
	}()
	//心跳
	{
		t := time.NewTicker(time.Second * time.Duration(comet.hbregistry))
		defer t.Stop()
		//心跳
		for {
			select {
			case <-t.C:
				msg, _ := json.Marshal(&model.DTO{
					Type: model.CmtHbToReg,
					Msg:  nil,
				})
				err := c.WriteMessage(websocket.TextMessage, msg)
				log.Printf("comet send heart beat to registry")
				if err != nil {
					log.Printf("comet send heart beat to registry,error:%v", err)
					comet.isregistered = false
					return
				}
			}
		}
	}
}

//watch registry conn   监控注册是否成功，否则重新注册
func (c *cometServer) watchreg() {
	t := time.NewTicker(time.Second * time.Duration(c.hbwatchreg))
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if !c.isregistered {
				c.registry()
			}
		}
	}
}

//statistics logic conn cnt ,client conn cnt
func (c *cometServer) statistics() {
	t := time.NewTicker(time.Second * 1)
	defer t.Stop()
	//定时计数
	for {
		select {
		case <-t.C:
			{
				cliCnt := 0
				logicCnt := 0
				groupCnt := 0
				for _, bkt := range c.clibkts {
					bkt.rw.RLock()
					bkt.clients.Range(func(key, value interface{}) bool {
						if clis, ok := value.([]*client); ok {
							cliCnt += len(clis)
							// for _, cli := range clis {
							// 	fmt.Println("clis cli:", cli.id, " ", cli.tag)
							// }
							// fmt.Println("clis len:", len(clis))
						}
						return true
					})
					bkt.rw.RUnlock()
				}
				c.logics.Range(func(key, value interface{}) bool {
					logicCnt++
					return true
				})

				c.cligroup.Range(func(key, value interface{}) bool {
					if clis, ok := value.([]*client); ok {

						groupCnt += len(clis)
						// for _, cli := range clis {
						// 	fmt.Println("grp", cli.id, " ", cli.tag)
						// }
						// fmt.Println("grp len clis", "key:", key, "len:", len(clis))
					}

					return true
				})
				fmt.Println("statistics:", cliCnt, logicCnt, groupCnt, "", time.Now())
			}
		}
	}
}

//consume msg  消费消息
func (c *cometServer) consume(ch chan *model.DTO) {
	for {
		select {
		case dto := <-ch:
			{
				//cmt
				{
					//cmt hb to lgc
					if dto.Type == model.CmtHbToLgc {
						c.logics.Range(func(key, value interface{}) bool {
							if v, ok := value.(*logic); ok {
								v.ch <- dto
							}
							return true
						})
					}
					//cmt hb to cli
					if dto.Type == model.CmtHbToCli {
						for _, bkt := range c.clibkts {
							bkt.rw.Lock()
							bkt.clients.Range(func(key, value interface{}) bool {
								if v, ok := value.(*client); ok {
									v.ch <- dto
								}
								return true
							})
							bkt.rw.Unlock()
						}
					}
				}
				//registry
				{
					//registry hb to comet
					if dto.Type == model.RegHbToCmt {
						log.Print("comet recv registry heart beat")
					}
				}
				//lgc
				{
					//lgc hb to comet
					if dto.Type == model.LgcHbToCmt {
						log.Print("comet recv logic heart beat")
					}
					//registry bcast lgc svrs to comet
					if dto.Type == model.RegBcastLgcSvrs {
						c.connlogic(dto)
					}
					//lgc cli msg to cmt
					if dto.Type == model.LgcCliMsg {
						d := &model.DTO{
							Type: model.CliCliMsg,
							Msg:  dto.Msg,
						}
						// fmt.Println("************* lgc cli msg to cmt:", d.Msg.Content, d.Msg.ToUserID, time.Now())

						id := d.Msg.ToUserID
						bkt := c.clibkts[c.modclidx(id)]
						if bkt != nil {
							bkt.rw.Lock()
							if value, ok := bkt.clients.Load(id); ok {
								if clis, ok := value.([]*client); ok {
									for _, cli := range clis {
										if cli.id == d.Msg.ToUserID && d.Msg.ToUserTag == cli.tag {
											cli.ch <- d
											break
										}
									}
								}
							}
							bkt.rw.Unlock()
						}
					}
					//lgc grp msg to cmt
					if dto.Type == model.LgcGrpMsg {
						d := &model.DTO{
							Type: model.CliGrpMsg,
							Msg:  dto.Msg,
						}
						grpid := d.Msg.GroupID
						if value, ok := c.cligroup.Load(grpid); ok {
							if clis, ok := value.([]*client); ok {
								for _, cli := range clis {
									cli.ch <- d
								}
							}
						}
					}
					//lgc bcastmsg to cmt
					if dto.Type == model.LgcBcastMsg {
						d := &model.DTO{
							Type: model.CliBcastMsg,
							Msg:  dto.Msg,
						}
						// fmt.Println("************* lgc bcastmsg to cmt:", d.Msg.Content, time.Now(), len(c.clients))
						for _, bkt := range c.clibkts {
							bkt.rw.Lock()
							bkt.clients.Range(func(key, value interface{}) bool {
								if clis, ok := value.([]*client); ok {
									for _, cli := range clis {
										cli.ch <- d
									}
								}
								return true
							})
							bkt.rw.Unlock()
						}
					}
					//lgc join grp msg
					if dto.Type == model.LgcJoinGrp {
						d := &model.DTO{
							Type: model.CliJoinGrp,
							Msg:  dto.Msg,
						}
						// fmt.Println("************* lgc join grp msg to cmt:", d.Msg.Content, time.Now())
						grpid := d.Msg.GroupID
						if value, ok := c.cligroup.Load(grpid); ok {
							if clis, ok := value.([]*client); ok {
								for _, cli := range clis {
									cli.ch <- d
								}
							}
						}
					}
					//lgc quit grp msg
					if dto.Type == model.LgcQuitGrp {
						d := &model.DTO{
							Type: model.CliCliMsg,
							Msg:  dto.Msg,
						}
						// fmt.Println("************* lgc quit grp msg to cmt:", d.Msg.Content, time.Now())
						grpid := d.Msg.GroupID
						if value, ok := c.cligroup.Load(grpid); ok {
							if clis, ok := value.([]*client); ok {
								for _, cli := range clis {
									cli.ch <- d
								}
							}
						}
					}
				}
				//client
				{
					//client send cli msg to cmt
					if dto.Type == model.CliCliMsg {
						//cmt rand lgc-cli delivery msg to lgc
						// fmt.Println("^^^^^^^^^ Comet Cli Recv:", dto.Msg.Content, time.Now())
						c.logics.Range(func(key, value interface{}) bool {
							if v, ok := value.(*logic); ok {
								v.ch <- dto
							}
							return false
						})
					}
					//cli send boardcast msg to cmt
					if dto.Type == model.CliBcastMsg {
						// fmt.Println("^^^^^^^^^  Comet BcastMsg Recv:", dto.Msg.Content, time.Now())
						//cmt rand lgc-cli delivery msg to lgc
						c.logics.Range(func(key, value interface{}) bool {
							if v, ok := value.(*logic); ok {
								v.ch <- dto
							}
							return false
						})
					}
					//cli send group msg to cmt
					if dto.Type == model.CliGrpMsg {
						// fmt.Println("^^^^^^^^^  Comet Group Recv:", dto.Msg.Content, time.Now())
						//cmt rand lgc-cli delivery msg to lgc
						c.logics.Range(func(key, value interface{}) bool {
							if v, ok := value.(*logic); ok {
								v.ch <- dto
							}
							return false
						})
					}
					//cli send join grp
					if dto.Type == model.CliJoinGrp {
						c.logics.Range(func(key, value interface{}) bool {
							if v, ok := value.(*logic); ok {
								v.ch <- dto
							}
							return false
						})
					}
					//cli quit grp
					if dto.Type == model.CliQuitGrp {
						c.logics.Range(func(key, value interface{}) bool {
							if v, ok := value.(*logic); ok {
								v.ch <- dto
							}
							return false
						})
					}
				}
			}
		}
	}
}

//conns to logic   与logic服务建立连接，在这里new logic client
func (c *cometServer) connlogic(dto *model.DTO) {
	if dto != nil && dto.Msg != nil && dto.Msg.Content != "" {
		addrs := strings.Split(dto.Msg.Content, ",")
		if len(addrs) > 0 {
			for _, addr := range addrs {
				if _, ok := c.logics.Load(addr); !ok {
					//conn to logic
					if addr != "" {
						go newlogic(addr, c).run()
					}
				}
			}
		}
	}
}

//rand a ch  多个channel中随机选择一个进行传递消息
func (c *cometServer) getch() (ch chan *model.DTO) {
	ch = c.chs[rand.Int63n(c.chsCnt-1)]
	return
}

//heartbeat 本服务的心跳检测
func (c *cometServer) hb() {
	//hb to client
	go func() {
		t := time.NewTicker(time.Second * time.Duration(c.hbclient))
		for {
			select {
			case <-t.C:
				{
					ch := c.getch()
					ch <- &model.DTO{
						Type: model.CmtHbToCli,
						Msg:  nil,
					}
				}
			}
		}
	}()

	//hb to logic
	go func() {
		t := time.NewTicker(time.Second * time.Duration(c.hblogic))
		for {
			select {
			case <-t.C:
				{
					ch := c.getch()
					ch <- &model.DTO{
						Type: model.CmtHbToLgc,
						Msg:  nil,
					}
				}
			}
		}
	}()
}
//hash算法，获取客户端的index
func (c *cometServer) modclidx(id int64) (idx int64) {
	return id % c.clibckcnt
}

//clis mv cli
func (c *cometServer) mvcli(cli *client) {
	if cli != nil {
		idx := c.modclidx(cli.id)
		bkt := c.clibkts[idx]
		idx1, isexist := 0, false
		bkt.rw.Lock()
		if value, ok := bkt.clients.Load(cli.id); ok {
			clis, ok := value.([]*client)
			if ok {
				for i, v := range clis {
					if v.tag == cli.tag {
						idx1, isexist = i, true
					}
				}
				if isexist {
					clis = append(clis[:idx1], clis[idx1+1:]...)
					bkt.clients.Store(cli.id, clis)
				}
			}
		}
		bkt.rw.Unlock()
	}
}

//clis add cli
func (c *cometServer) addcli(cli *client) {
	if cli != nil {
		idx := c.modclidx(cli.id)
		bkt := c.clibkts[idx]
		newclis := make([]*client, 0)
		idx1, isexist := 0, false
		bkt.rw.Lock()
		if value, ok := bkt.clients.Load(cli.id); ok {
			clis, ok := value.([]*client)
			if ok {
				for i, v := range clis {
					if v.tag == cli.tag {
						idx1, isexist = i, true
					}
				}
			}
			if isexist {
				newclis = append(clis[:idx1], clis[idx1+1:]...)
			} else {
				newclis = append(newclis, clis[0:]...)
			}
		}
		newclis = append(newclis, cli)
		bkt.clients.Store(cli.id, newclis)
		bkt.rw.Unlock()
	}
}

//grp add cli
func (c *cometServer) grpaddcli(cli *client, grpid int64) {
	if cli != nil {
		c.grprw.Lock()
		newclis := make([]*client, 0)
		if value, ok := c.cligroup.Load(grpid); ok {
			if clis, ok := value.([]*client); ok {
				for i, v := range clis {
					if v.id == cli.id && cli.tag == v.tag {
						newclis = append(clis[:i], clis[i+1:]...)
						break
					} else {
						newclis = append(newclis, clis[0:]...)
						break
					}
				}
			}
		}
		newclis = append(newclis, cli)
		c.cligroup.Store(grpid, newclis)
		c.grprw.Unlock()
	}
}

//group remove special cli
func (c *cometServer) grpmvcli(cli *client, grpid int64) {
	if cli != nil {
		c.grprw.Lock()
		defer c.grprw.Unlock()
		idx, isexist := 0, false
		if clis, ok := c.cligroup.Load(grpid); ok {
			clis, ok := clis.([]*client)
			if ok {
				for i, v := range clis {
					if v.id == cli.id && cli.tag == v.tag {
						idx, isexist = i, true
					}
				}
			}
			if isexist {
				clis = append(clis[:idx], clis[idx+1:]...)
				c.cligroup.Store(grpid, clis)
			}
		}
	}
}
//group remove special cli
func (c *cometServer) grpmvcli2(cli *client, grpid int64) {
	if cli != nil {
		c.grprw.Lock()
		defer c.grprw.Unlock()
		idx, isexist := 0, false
		if clis, ok := c.cligroup.Load(grpid); ok {
			clis, ok := clis.([]*client)
			if ok {
				for i, v := range clis {
					if v.id == cli.id && cli.tag == v.tag {
						idx, isexist = i, true
					}
				}
			}
			if isexist {
				clis = append(clis[:idx], clis[idx+1:]...)
				c.cligroup.Store(grpid, clis)
			}
		}
	}
}

//isexist in clis
func (c *cometServer) isexist(cli *client) bool {
	if cli != nil {
		idx := c.modclidx(cli.id)
		bkt := c.clibkts[idx]
		bkt.rw.Lock()
		defer bkt.rw.Unlock()
		if clis, ok := bkt.clients.Load(cli.id); ok {
			if clis, ok := clis.([]*client); ok {
				for _, v := range clis {
					if v.tag == cli.tag {
						return true
					}
				}
			}
		}
	}
	return false
}
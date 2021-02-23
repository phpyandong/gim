package model

type Conf struct {
	Registry *RegistryConf `json:"registry"`
	Comet    *CometConf    `json:"comet"`
	Logic    *LogicConf    `json:"logic"`
}

type RegistryConf struct {
	Host          string `json:"host"`
	Port          string `json:"port"`
	HBComet       int64  `json:"hb_comet"`         //heart beat to comet
	HBLogic       int64  `json:"hb_logic"`         //heart beat to logic
	HBcastLgcSvrs int64  `json:"hb_bcast_lgcsvrs"` //heart beat broadcast logic server addrs to comet
}

type CometConf struct {
	Port       string `json:"port"`
	HBClient   int64  `json:"hb_client"`    //heart beat to client
	HBLogic    int64  `json:"hb_logic"`     // heart beat to logic
	HBRegistry int64  `json:"hb_registry"`  //heart beat to registry
	HBWatchReg int64  `json:"hb_watch_reg"` //watch is connected reg
	CliBckCnt  int64  `json:"cli_bck_cnt"`  //cli bucket cnt
}

type LogicConf struct {
	Port       string `json:"port"`
	HBComet    int64  `json:"hb_comet"`     //heart beat to comet
	HBRegistry int64  `json:"hb_registry"`  //heart beat to registry
	HBWatchReg int64  `json:"hb_watch_reg"` //watch is connected reg
}

func NewConf() *Conf {
	return &Conf{
		Registry: &RegistryConf{
			Host:          "127.0.0.1",
			Port:          "7171",
			HBcastLgcSvrs: 10,
			HBComet:       60,
			HBLogic:       60,
		},
		Logic: &LogicConf{
			Port:       "3109",
			HBComet:    60,
			HBRegistry: 60,
			HBWatchReg: 3,
		},
		Comet: &CometConf{
			Port:       "3101",
			HBClient:   5,
			HBLogic:    60,
			HBRegistry: 60,
			HBWatchReg: 3,
			CliBckCnt:  1024,
		},
	}
}

package model

const (
	//comet
	CmtHbToCli = "100"
	CmtHbToLgc = "101"
	CmtHbToReg = "102"
	//registry
	RegHbToCmt      = "200"
	RegHbToLgc      = "201"
	RegBcastLgcSvrs = "202"
	//logic
	LgcHbToCmt  = "300"
	LgcHbToReg  = "301"
	LgcCliMsg   = "302"
	LgcGrpMsg   = "303"
	LgcBcastMsg = "304"
	LgcJoinGrp  = "305"
	LgcQuitGrp  = "306"
	//client
	CliHbToCmt  = "400"
	CliCliMsg   = "401"
	CliGrpMsg   = "402"
	CliBcastMsg = "403"
	CliJoinGrp  = "404"
	CliQuitGrp  = "405"
)

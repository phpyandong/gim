package main

import (
	"github.com/phpyandong/gim/logic"
	"github.com/phpyandong/gim/model"
)

func main() {
	l := logic.New(model.NewConf())
	l.Run()
}

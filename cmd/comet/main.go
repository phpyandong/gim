package main

import (
	"github.com/phpyandong/gim/comet"
	"github.com/phpyandong/gim/model"
)

func main() {
	c := comet.NewCometServer(model.NewConf())
	c.Run()
}

package main

import (
	"github.com/phpyandong/gim/comet"
	"github.com/phpyandong/gim/model"
)

func main() {
	c := comet.New(model.NewConf())
	c.Run()
}

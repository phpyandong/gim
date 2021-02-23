package main

import (
	"github.com/phpyandong/gim/model"
	"github.com/phpyandong/gim/registry"
)

func main() {
	r := registry.New(model.NewConf())
	r.Run()
}

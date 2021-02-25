package comet

import (
	"github.com/phpyandong/gim/model"
	"sync"
)

type Group struct{
	id int64
}
type Groups struct {
	groups []*Group
	rw 	sync.Mutex
}

//退出指定群
func (groups *Groups) quitgrp(dto *model.DTO) {
	//cli.rw.Lock()
	groups.rw.Lock()
	idx, isexist := 0, false
	for i, grp := range groups.groups {
		if dto.Msg.GroupID == grp.id {
			idx, isexist = i, true
		}
	}
	if isexist {
		groups.groups = append(groups.groups[:idx], groups.groups[idx+1:]...)
	}
	groups.rw.Unlock()
}

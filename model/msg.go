package model

import (
	"github.com/hicsgo/glib"
)

type (
	Msg struct {
		FromUserID  int64  `json:"from_user_id"`
		FromUserTag string `json:"from_user_tag"`
		ToUserID    int64  `json:"to_user_id"`  //用户
		ToUserTag   string `json:"to_user_tag"` //web,ios,android,mini,h5,web
		GroupID     int64  `json:"group_id"`    //群消息ID
		Content     string `json:"content"`     //消息内容
	}
	DTO struct {
		Msg  *Msg   `json:"msg"`
		Type string `json:"type"`
	}
)

func (dto *DTO) ToJson() string {
	jsonString, err := glib.ToJson(dto)
	if err != nil {
		return ""
	}

	return jsonString
}

func (msg *Msg) ToJson() string {
	jsonString, err := glib.ToJson(msg)
	if err != nil {
		return ""
	}
	return jsonString
}

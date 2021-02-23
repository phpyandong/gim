go version 1.13.8

goimx
#### 支持:
ws
水平扩展
注册中心，服务发现，心跳监测
单发，群发，广播

#### 暂不支持:
消息存储，敏感词过滤

## goimx 

### 使用说明

```
构建
make build
运行
make run
停止
make stop
```

```
连接地址：
ws://127.0.0.1:3101/ws?tag=ios&token=1
token 是jwt token 目前使用user_id代替
tag 是指终端类别有如下：ios,android,mini,web,h5,ipad
```
### type说明
```
400 发送心跳
401 用户消息
402 群组消息
403 广播消息
404 加入群
405 退出群
```
### example
```
发送心跳
{
	"msg": nil,
	"type": "400"
}

全员广播消息
{
	"msg": {
		"content": "hello world..."
	},
	"type": "403"
}

向1号用户，ios设备发送消息
{
	"msg": {
		"to_user_id": 1,
		"to_user_tag": "ios",
		"content":"单播消息..."
	},
	"type": "401"
}

加入2号群
{
	"msg": {
		"group_id": 2
	},
	"type": "404"
}

退出2号群
{
	"msg": {
		"group_id": 2
	},
	"type": "405"
}

向2群组发消息
{
	"msg": {
		"group_id": 2,
		"content": "hello world..."
	},
	"type": "403"
}
```

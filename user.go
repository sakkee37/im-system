package main

import (
	"net"
	"strings"
)

type User struct {
	Name   string
	Addr   string
	C      chan string
	conn   net.Conn
	server *Server
}

// 创建一个用户并启动监听消息协程
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name:   userAddr,
		Addr:   userAddr,
		C:      make(chan string),
		conn:   conn,
		server: server,
	}

	go user.ListenMessage()

	return user
}

// 用户上线：加入在线表并广播
func (this *User) Online() {
	this.server.mapLock.Lock()
	this.server.OnlineMap[this.Name] = this
	this.server.mapLock.Unlock()

	this.server.BroadCast(this, "已上线")
}

// 用户下线：从在线表删除并广播
func (this *User) Offline() {
	this.server.mapLock.Lock()
	delete(this.server.OnlineMap, this.Name)
	this.server.mapLock.Unlock()

	this.server.BroadCast(this, "下线")
}

// 给当前用户对应的客户端发送消息
func (this *User) SendMsg(msg string) {
	this.conn.Write([]byte(msg))
}

// 处理当前用户发来的消息
func (this *User) DoMessage(msg string) {
	if msg == "who" {
		// 查询当前在线用户
		this.server.mapLock.Lock()
		for _, user := range this.server.OnlineMap {
			onlineMsg := "[" + user.Addr + "]" + user.Name + ":" + "在线...\n"
			this.SendMsg(onlineMsg)
		}
		this.server.mapLock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		// 消息格式：rename|张三
		newName := strings.Split(msg, "|")[1]
		// 判断 name 是否已经存在
		this.server.mapLock.Lock()
		_, ok := this.server.OnlineMap[newName]
		if ok {
			this.SendMsg("当前用户名被使用\n")
		} else {
			delete(this.server.OnlineMap, this.Name)
			this.server.OnlineMap[newName] = this
			this.Name = newName
			this.SendMsg("您已经更新用户名:" + this.Name + "\n")
		}
		this.server.mapLock.Unlock()
	} else if len(msg) > 4 && msg[:3] == "to|" {
		// 消息格式：to|张三|消息内容
		parts := strings.Split(msg, "|")
		if len(parts) < 3 || parts[1] == "" {
			this.SendMsg("消息格式不正确，请使用 \"to|张三|你好啊\" 格式。\n")
			return
		}
		remoteName := parts[1]
		content := parts[2]
		// 根据用户名得到对方 User 对象
		this.server.mapLock.Lock()
		remoteUser, ok := this.server.OnlineMap[remoteName]
		this.server.mapLock.Unlock()
		if !ok {
			this.SendMsg("该用户名不存在\n")
			return
		}
		// 获取消息内容，通过对方的 User 对象将消息发送过去
		if content == "" {
			this.SendMsg("无消息内容，请重发\n")
			return
		}
		remoteUser.SendMsg(this.Name + "对您说:" + content + "\n")
	} else {
		this.server.BroadCast(this, msg)
	}
}

// 监听当前 User 的 channel，有消息就写回客户端
func (this *User) ListenMessage() {
	for {
		msg := <-this.C
		this.conn.Write([]byte(msg + "\n"))
	}
}

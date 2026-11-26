package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip        string
	Port      int
	OnlineMap map[string]*User
	mapLock   sync.RWMutex
	Message   chan string
}

// 创建一个 Server 实例
func NewServer(ip string, port int) *Server {
	return &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
}

// 监听 Message 通道，有消息就转发给所有在线用户
func (this *Server) ListenMessager() {
	for {
		msg := <-this.Message

		this.mapLock.Lock()
		for _, cli := range this.OnlineMap {
			cli.C <- msg
		}
		this.mapLock.Unlock()
	}
}

// 广播消息到全局 Message 通道
func (this *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	this.Message <- sendMsg
}

// 处理单个客户端连接的生命周期
func (this *Server) Handler(conn net.Conn) {
	user := NewUser(conn, this)
	user.Online()

	// 监听当前用户是否活跃的 channel
	isLive := make(chan bool)
	// 接收客户端发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}

			if err != nil && err != io.EOF {
				fmt.Println("Conn Read err:", err)
				return
			}

			msg := string(buf[:n-1])

			user.DoMessage(msg)

			// 用户的任意消息，代表当前用户是一个活跃的
			isLive <- true
		}
	}()
	for {
		select {
		case <-isLive:
			// 当前用户是活跃的，不做任何操作，为了激活 select，更新下面的定时器
		case <-time.After(time.Second * 10):
			// 已经超时，将当前的 User 强制关闭
			user.SendMsg("你被踢了\n")
			// 销毁用的资源
			close(user.C)
			// 关闭连接
			conn.Close()
			// 退出当前 Handler
			return
		}
	}
}

// 启动服务并循环接收客户端连接
func (this *Server) Start() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}
	defer listener.Close()

	go this.ListenMessager()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener accept err:", err)
			continue
		}
		go this.Handler(conn)
	}
}

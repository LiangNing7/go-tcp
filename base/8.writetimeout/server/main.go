package main

import (
	"fmt"
	"net"
	"time"
)

func handleConn(c net.Conn) {
	defer c.Close()
	time.Sleep(time.Second * 10)
	fmt.Println("准备读取...")
	for {
		time.Sleep(5 * time.Second)
		buf := make([]byte, 60000)
		n, err := c.Read(buf)
		if err != nil {
			fmt.Printf("读取发生错误: %+v\n", err)
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			}
			return
		}
		fmt.Printf("%s 已读取 %d 字节\n", time.Now().Format("2006 15:04:05.000"), n)
	}
}

func main() {
	l, err := net.Listen("tcp", ":8888")
	if err != nil {
		fmt.Println("listen error:", err)
		return
	}
	fmt.Println("已启动 tcp server")
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("accept error", err)
			break
		}
		fmt.Println("接收到一个新的连接")
		// start a new goroutine to handle
		// the new connection.
		go handleConn(c)
	}
}

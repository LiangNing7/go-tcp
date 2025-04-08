package main

import (
	"fmt"
	"net"
	"time"
)

func handleConn(c net.Conn) {
	defer c.Close()

	fmt.Println("准备读取...")

	for {
		buf := make([]byte, 10)
		c.SetReadDeadline(time.Now().Add(time.Duration(3) * time.Second)) // 设置超时时间.
		n, err := c.Read(buf)
		if err != nil {
			fmt.Printf("读取发生错误: %+v\n", err)
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				fmt.Printf("%s %s", time.Now().Format("2006-01-02 15:04:05.000"), "发生读超时")
				continue
			}
			return
		}
		fmt.Printf("%s 已读取 %d 字节\n", time.Now().Format("2006-01-02 15:04:05.000"), n)
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
			fmt.Println("accept error:", err)
			break
		}
		fmt.Println("接收到一个新的连接")
		// start a new goroutine to handle
		// the new connection.
		go handleConn(c)
	}
}

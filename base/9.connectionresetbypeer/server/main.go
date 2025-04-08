package main

import (
	"fmt"
	"net"
	"time"
)

func hanleConn(c net.Conn) {
	defer c.Close()

	fmt.Println("准备数据...")

	for {
		buf := make([]byte, 10)
		n, err := c.Read(buf)
		if err != nil {
			fmt.Printf("%s 读取发生错误: %+v \n", time.Now().Format("2006-01-02 15:04:05.000"), err)
			return
		}
		fmt.Printf("%s 已读取 %d 字节，内容是：%s \n", time.Now().Format("2006-01-02 15:04:05.000"), n, string(buf))
		write, err := c.Write(buf[0:n])
		if err != nil {
			fmt.Printf("%s 写入发生错误: %+v \n", time.Now().Format("2006-01-02 15:04:05.000"), err)
			return
		}
		fmt.Printf("%s 已写入 %d 字节\n", time.Now().Format("2006-01-02 15:04:05.000"), write)
	}
}

func main() {
	l, err := net.Listen("tcp", ":8888")
	if err != nil {
		fmt.Println("listen error", err)
		return
	}
	fmt.Println("已启动 tcp server")
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("accpt error:", err)
			break
		}
		fmt.Println("接收到一个新的连接")
		// start a new goroutine to handle
		// the new connection.
		go hanleConn(c)
	}
}

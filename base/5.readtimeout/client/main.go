package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	fmt.Println("begin dial...")
	conn, err := net.Dial("tcp", ":8888")
	if err != nil {
		log.Println("dial error:", err)
		return
	}
	defer conn.Close()
	fmt.Println("dial ok")

	time.Sleep(5 * time.Second)

	var n int
	if n, err = conn.Write([]byte("hello")); err != nil {
		fmt.Printf("写入发生错误：%+v", err)
		return
	}

	fmt.Printf("已写入 %d 字节", n)
}

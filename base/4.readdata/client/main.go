package main

import (
	"fmt"
	"log"
	"net"
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

	var n int
	if n, err = conn.Write([]byte("hello world! Gopher")); err != nil {
		fmt.Printf("写入发生错误：%+v", err)
		return
	}

	fmt.Printf("已写入 %d 字节", n)
}

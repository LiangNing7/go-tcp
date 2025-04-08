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

	var n int
	message := "hello world Gopher123"
	if n, err = conn.Write([]byte(message)); err != nil {
		fmt.Printf("写入发生错误: %+v", err)
		return
	}

	fmt.Printf("已写入 %d 字节", n)

	time.Sleep(time.Second * 3)

	// 开启这段代码可解决 connection reset by perr.
	/* buf := make([]byte, 10)
	total := 0
	for total < len(message) {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("读取发生错误：", err)
			break
		}
		fmt.Println(fmt.Sprintf("收到 %d 字节：%s", n, string(buf[:n])))
		total += n
	}

	time.Sleep(time.Second * 1) */

	fmt.Println(time.Now().Format("2006-01-02 15:04:05.000"), "关闭")
}

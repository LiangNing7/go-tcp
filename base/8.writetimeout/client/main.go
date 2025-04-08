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

	data := make([]byte, 65536)
	var total int
	for {
		conn.SetWriteDeadline(time.Now().Add(time.Second * 1))
		n, err := conn.Write(data)
		if err != nil {
			total += n
			fmt.Printf("write %d bytes, error: %+v\n", n, err)
			break
		}
		total += n
		fmt.Printf("write %d bytes this time, %d bytes in total\n", n, total)
	}
	fmt.Printf("write %d bytes in total\n", total)
}

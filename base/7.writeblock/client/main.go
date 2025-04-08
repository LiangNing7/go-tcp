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
		n, err := conn.Write(data)
		if err != nil {
			total += n
			fmt.Printf("write %d bytes, error: %+v\n", n, err)
			break
		}
		total += n
		fmt.Printf("%s write %d bytes this time, %d bytes in total\n", time.Now().Format("2006-01-02 15:04:05.000"), n, total)
	}
	fmt.Printf("write %d bytes in total\n", total)
}

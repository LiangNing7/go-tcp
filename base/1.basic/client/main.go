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
		log.Panicln("dial error:", err)
		return
	}
	defer conn.Close()
	fmt.Println("dial ok")
}

package main

import (
	"fmt"
	"net"

	"github.com/LiangNing7/go-tcp/frame"
	"github.com/LiangNing7/go-tcp/packet"
)

// handlePacket 对于客户端发送来的消息请求包进行响应
// framePayload 是请求包中的 Frame 将 totalLength 处理后，得到的 Packet.
// 返回 ackFramePayload err.
// 其中 ackFramePayload 是消息响应包中的 Packet【即 framePayload】.
func handlePacket(framePayload []byte) (ackFramePayload []byte, err error) {
	var p packet.Packet

	// 对消息请求包中的 packet 进行解码.
	// Packet ===> commandID,ID,payload.
	p, err = packet.Decode(framePayload)
	if err != nil {
		fmt.Println("handleConn: packet decode error:", err)
		return
	}

	switch p := p.(type) {
	case *packet.Submit: // 只需要对消息请求包进行响应.
		fmt.Printf("recv submit: id = %s, payload=%s\n", p.ID, string(p.Payload))
		submitAck := &packet.SubmitAck{ // 返回消息请求的响应包，只包含 Packet 中的 Body.
			ID:     p.ID, // 消息流水号.
			Result: 0,    // 成功.
		}
		// 对消息请求的响应包 Packet.Body 进行编码
		// 并加上 commandID 从而生成完整的 Packet，也就是 Frame 中的 Payload.
		ackFramePayload, err = packet.Encode(submitAck)
		if err != nil {
			fmt.Println("handleConn: packet encode error:", err)
			return nil, err
		}
		return ackFramePayload, nil
	default:
		return nil, fmt.Errorf("unknown packet type")
	}
}

// handleConn 接收客户端发来消息请求包 Submit 的 Frame
// 接收到消息后，将 SubmitAck 写回 conn.
func handleConn(c net.Conn) {
	defer c.Close()
	// 初始化 Frame 编解码器.
	frameCodec := frame.NewMyFrameCodec()

	for {
		// read from the connection

		// decode the frame to get the payload.
		// Submit Frame ===> Packet【Submit FramePayload】.
		framePayload, err := frameCodec.Decode(c)
		if err != nil {
			fmt.Println("handleConn: frame decode error:", err)
			return
		}

		// do something with the packet
		// Packet ===>  SubmitAck FramePayload
		ackFramePayload, err := handlePacket(framePayload)
		if err != nil {
			fmt.Println("handleConn: handle packet error:", err)
			return
		}

		// write ack frame to the connection.
		// 将 SubmitAck FramePayload 再次编码【添加 totalLength】写回 conn.
		err = frameCodec.Encode(c, ackFramePayload)
		if err != nil {
			fmt.Println("handleConn: frame encode error:", err)
			return
		}
	}
}

func main() {
	l, err := net.Listen("tcp", ":8888")
	if err != nil {
		fmt.Println("listen error:", err)
		return
	}

	fmt.Println("server start ok(on *.8888)")

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			break
		}
		// start a new goroutine to handle
		// the new connection.
		go handleConn(c)
	}
}

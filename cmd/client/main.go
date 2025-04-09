package main

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/LiangNing7/go-tcp/frame"
	"github.com/LiangNing7/go-tcp/packet"
	"github.com/lucasepe/codename"
)

func main() {
	var wg sync.WaitGroup
	clientNum := 5
	wg.Add(clientNum)

	// 创建多个客户端并行测试.
	for i := range clientNum {
		go func(i int) {
			defer wg.Done()
			startClient(i) //  启动客户端逻辑.
		}(i + 1)
	}
	wg.Wait() // 等待所有客户完成.
}

// startClient 客户端核心逻辑.
func startClient(clientID int) {
	// 控制通道.
	quit := make(chan struct{}) // 退出信号.
	done := make(chan struct{}) // 退出确认.

	// 建立 TCP 连接.
	conn, err := net.Dial("tcp", ":8888")
	if err != nil {
		fmt.Println("dial error:", err)
		return
	}
	defer conn.Close()
	fmt.Printf("[client %d]: dial ok\n", clientID)

	// 初始化组件.
	// 生成 payload.
	rng, err := codename.DefaultRNG() // 随机数生成器.
	if err != nil {
		panic(err)
	}

	frameCodec := frame.NewMyFrameCodec() // 创建帧编解码器.
	var counter int                       // 请求计数器

	// 响应处理 goroutine.
	go func() {
		// handle ack.
		for {
			// 处理退出信号.
			select {
			case <-quit: // 收到退出信号.
				done <- struct{}{}
				return
			default:
			}

			// 设置读超时时间 5s
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			// 解码帧数据.
			ackFramePayLoad, err := frameCodec.Decode(conn)
			if err != nil {
				// 处理超时错误.
				if e, ok := err.(net.Error); ok {
					if e.Timeout() {
						continue
					}
				}
				panic(err)
			}

			// 解码协议包.
			p, err := packet.Decode(ackFramePayLoad)
			if err != nil {
				panic(err)
			}
			submitAck, ok := p.(*packet.SubmitAck)
			if !ok {
				panic("not submitack")
			}
			fmt.Printf("[client %d]: the result of submit ack[%s] is %d\n", clientID, submitAck.ID, submitAck.Result)
		}
	}()

	// 请求发送循环
	for {

		// 发送 submit.
		counter++

		// 构造请求数据
		id := fmt.Sprintf("%08d", counter) // 8位数字 ID.
		payload := codename.Generate(rng, 4)
		s := &packet.Submit{
			ID:      id,
			Payload: []byte(payload),
		}

		// 编码协议包.
		framePayload, err := packet.Encode(s)
		if err != nil {
			panic(err)
		}

		// 打印发送日志（+4 包含帧头长度）.
		fmt.Printf("[client %d]: send submit id = %s,payload=%s,frame length = %d\n", clientID, s.ID, s.Payload, len(framePayload)+4)
		// 发送帧数据.
		if err := frameCodec.Encode(conn, framePayload); err != nil {
			panic(err)
		}
		// 控制发送节奏.
		time.Sleep(1 * time.Second)

		// 退出条件判断.
		if counter >= 10 {
			quit <- struct{}{} // 通知处理协程退出.
			<-done             // 等待处理协程确认.
			fmt.Printf("[client %d]: exit ok\n", clientID)
			return
		}
	}
}

package packet

import (
	"bytes"
	"fmt"
)

/* 协议定义
-----------------------------------------
基础包头：
+---------------+-------------------+
| commandID(1B) |     packet body   |
+---------------+-------------------+

具体包类型：
1. Submit包（commandID=0x02）:
+----------------+------------------+
|   ID(8B str)   |  payload(var)    |
+----------------+------------------+

2. SubmitAck包（commandID=0x82）:
+----------------+------------+
|   ID(8B str)   | result(1B) |
+----------------+------------+
*/

const (
	CommandConn   = iota + 0x01 // 0x01 连接请求.
	CommandSubmit               // 0x02 数据提交.
)

const (
	CommandConnAck   = iota + 0x80 // 0x81连接确认.
	CommandSubmitAck               // 0x82 提交确认.
)

// Packet 协议包统一接口.
type Packet interface {
	Decode([]byte) error     // []byte -> struct.
	Encode() ([]byte, error) // []struct -> []byte.
}

// Submit 提交数据包结构.
type Submit struct {
	ID      string // 固定 8 字节标识.
	Payload []byte // 可变长度载荷数据.
}

// Decode 实现 Submit 包的解码
func (s *Submit) Decode(pktBody []byte) error {
	// 前 8 字节作为 ID（确保输入长度 >= 8）.
	s.ID = string(pktBody[:8])
	// 剩余部分作为载荷.
	s.Payload = pktBody[8:]
	return nil
}

// Encode 实现 Submit 包的编码.
func (s *Submit) Encode() ([]byte, error) {
	return bytes.Join([][]byte{[]byte(s.ID[:8]), s.Payload}, nil), nil
}

// SubmitAck 提交确认包结构
type SubmitAck struct {
	ID     string // 固定 8 字节.
	Result uint8  // 1 字节处理结果
}

// Decode 实现SubmitAck包的解码.
func (s *SubmitAck) Decode(pktBody []byte) error {
	s.ID = string(pktBody[0:8])  // 前8字节为ID.
	s.Result = uint8(pktBody[8]) // 第9字节为结果.
	return nil
}

// Encode 实现 SubmitAck 包的编码.
func (s *SubmitAck) Encode() ([]byte, error) {
	return bytes.Join([][]byte{[]byte(s.ID[:8]), {s.Result}}, nil), nil
}

// Decode 对 Packet 进行解码.
func Decode(packet []byte) (Packet, error) {
	commandID := packet[0] // 1 字节的响应类型.
	pktBody := packet[1:]  // 后续字节为载荷体.
	switch commandID {
	case CommandConn:
		return nil, nil
	case CommandConnAck:
		return nil, nil
	case CommandSubmit:
		s := Submit{}
		err := s.Decode(pktBody)
		if err != nil {
			return nil, err
		}
		return &s, nil
	case CommandSubmitAck:
		s := SubmitAck{}
		err := s.Decode(pktBody)
		if err != nil {
			return nil, err
		}
		return &s, nil
	default:
		return nil, fmt.Errorf("unknown commandID [%d]", commandID)
	}
}

// Encode 对 Packet 包进行编码.
// p 为 Packet 的 Body，根据其类型给其加上 commandID.
func Encode(p Packet) ([]byte, error) {
	var commandID uint8 // 消息类型.
	var pktBody []byte  // 消息体.
	var err error

	// 类型断言确定包的类型.
	switch t := p.(type) {
	case *Submit:
		commandID = CommandSubmit
		pktBody, err = p.Encode()
		if err != nil {
			return nil, err
		}
	case *SubmitAck:
		commandID = CommandSubmitAck
		pktBody, err = p.Encode()
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown type [%s]", t)
	}
	// 拼接命令字节与包体.
	return bytes.Join([][]byte{{commandID}, pktBody}, nil), nil
}

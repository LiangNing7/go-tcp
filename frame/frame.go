package frame

import (
	"encoding/binary"
	"errors"
	"io"
)

/*
Frame 结构定义：
+----------------+-----------------------+
| frameHeader(4) |    framePayload(packet)  |
+----------------+-----------------------+
frameHeader: 4字节大端序整型，表示帧总长度（含头及 payload ）
framePayload: 实际数据载荷，对应 packet 内容
*/

// FramePayload 表示帧的有效载荷类型(字节切片).
type FramePayload []byte

type SteamFrameCodec interface {
	Encode(io.Writer, FramePayload) error   // 将数据编码为帧格式写入 io.Writer.
	Decode(io.Reader) (FramePayload, error) // 从 io.Reader 解码帧数据返回有效载荷.
}

// 错误定义.
var (
	ErrShortWrite = errors.New("short write") // 写入数据不足时返回.
	ErrShortRead  = errors.New("short read")  // 读取数据不足时返回.
)

// myFrameCodec 编解码器具体实现.
type myFrameCodec struct{}

// NewMyFrameCodec 创建帧编解码器实例.
func NewMyFrameCodec() SteamFrameCodec {
	return &myFrameCodec{}
}

// Encode 编码实现.
func (p *myFrameCodec) Encode(w io.Writer, framePayload FramePayload) error {
	f := framePayload
	// 计算总长度.
	totalLen := int32(len(framePayload)) + 4 // 4 字节的头部，即 totalLength.

	// 以大端序写入 4 字节帧头（包含头部信息）.
	if err := binary.Write(w, binary.BigEndian, &totalLen); err != nil {
		return err
	}

	// 写入有效载荷.
	n, err := w.Write([]byte(f))
	if err != nil {
		return err
	}

	// 验证实际写入字节数是否符合预期.
	if n != len(framePayload) {
		return ErrShortWrite
	}
	return nil
}

// Decode 解码方法实现.
func (p *myFrameCodec) Decode(r io.Reader) (FramePayload, error) {
	var totalLen int32

	// 读取 4 字节帧头获取总长度. 【因为 totalLen 是 int32 类型，占 4 字节】
	if err := binary.Read(r, binary.BigEndian, &totalLen); err != nil {
		return nil, err
	}

	// 创建缓冲区（总长度减去 4 字节头部长度）.
	buf := make([]byte, totalLen-4)

	// 读取完整的 payload 数据 （使用 ReadFull 确保读取指定字节数）.
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	// 验证读取字节数是否满足预期.
	if n != int(totalLen-4) {
		return nil, ErrShortRead
	}

	// 将字节切片转换为 FramePayload 类型进行返回.
	return FramePayload(buf), nil
}

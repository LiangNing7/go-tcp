Socket 中有足够数据的场景

先启动服务端：`go run base/4.readdata/server/main.go`

再启动客户端：`go run base/4.readdata/client/main.go`，发送的数据为：`hello world! Gopher`

客户端发送了19字节的数据，

服务端第一次读取数据将大小为 10 的 buf 填满，第一次读取的数据为 `hello worl`

服务端第二次读取数据，读取大小为 9 的剩余数据，数据为`d! Gopher`

服务器第三次读取数据，客户端在写入完成后，随后关闭连接（由于客户端在 `main` 中调用了 `defer conn.Close()`，且写操作完成后程序结束），此时服务端继续循环调用 `Read` 时，会遇到连接关闭，从而返回一个错误（通常是 `io.EOF`）。服务端会打印类似：`读取发送错误: EOF`

![image-20250408115230515](http://images.liangning7.cn/typora/202504081152341.png)
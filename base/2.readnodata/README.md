Socket 中无数据的场景【观察 server 的打印】

先启动服务端：`go run base/2.readnodata/server/main.go`

再启动客户端：`go run base/2.readnodata/client/main.go`

与 1 类似，在最后客户端连接完成后，没有数据连接，待睡眠完成后，客户端退出后，服务端收到 io.EOF 后，通过 defer 关掉和这个客户端的连接。然后继续等待客户端的连接

![image-20250408113653962](http://images.liangning7.cn/typora/202504081136022.png)
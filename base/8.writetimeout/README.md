**写入超时**的场景

先启动服务端：`go run base/8.writetimeout/server/main.go`

再启动客户端：`go run base/8.writetimeout/client/main.go`

逻辑：客户端在写满缓冲区之后，由于服务端要等待，客户端发现超过1s之后就直接退出

![image-20250408135438710](http://images.liangning7.cn/typora/202504081354819.png)

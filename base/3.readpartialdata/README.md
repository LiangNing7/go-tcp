Socket 中有部分数据的场景

先启动服务端：`go run base/3.readpartialdata/server/main.go`

再启动客户端：`go run base/3.readpartialdata/client/main.go`，发送的数据为`hello`，没有超过 `buf`

`c.Read()` 正常读取数据到 buf 中然后代码继续往下执行，而不是等 buf 满了才往下执行。

![image-20250408114230027](http://images.liangning7.cn/typora/202504081142079.png)
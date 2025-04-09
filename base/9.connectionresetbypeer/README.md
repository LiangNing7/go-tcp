模拟 `read: connection reset by peer` 异常

异常原因：client 的接收缓冲区还有未读数据，这时当 client 关闭，server 会报此异常。与 EOF 异常的区别是，当 client 把接收缓冲区的数据都读取完毕再关闭，就是 EOF 了
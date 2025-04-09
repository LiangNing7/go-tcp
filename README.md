# go-tcp 基础

> 实现一个基于 TCP 的自定义应用层协议的通信服务端。

## 问题描述

我们的输入，是一个基于传输层自定义的应用层协议规范。由于 TCP 是面向连接的流协议传输机制，数据流本身没有明显的边界，这样定义协议时，就需要自行定义确定边界的方法，因此，基于 TCP 的自定义应用层协议通常有两种常见的定义模式：

* **二进制模式：**采用长度字段标识独立数据包的边界。采用这种方式定义的常见协议包括 MQTT（物联网最常用的应用层协议之一）、SMPP（短信网关点对点接口协议）等；
* **文本模式**：采用特定分隔符标识流中的数据包的边界，常见的包括HTTP协议等。

相比之下，二进制模式要比文本模式编码更紧凑也更高效，所以我们这个问题中的自定义协议也采用了**二进制模式**，协议规范内容如下图：

![image-20250408084042873](http://images.liangning7.cn/typora/202504080840999.png)

关于协议内容的分析，我们放到设计与实现的那一讲中再细说，这里我们再看一下使用这个协议的通信两端的通信流程：

![image-20250408084123148](http://images.liangning7.cn/typora/202504080841210.png)

我们看到，这是一个典型的“请求/响应”通信模型。连接由客户端发起，建立连接后，客户端发起请求，服务端收到请求后处理并返回响应，就这样一个请求一个响应的进行下去，直到客户端主动断开连接为止。

**而我们的任务，就是实现支持这个协议通信的服务端。**

首先，前面说过 socket 是传输层给用户提供的编程接口，我们要进行的网络通信绕不开 socket，因此我们首先需要了解 socket 编程模型。

其次，一旦通过 socket 将双方的连接建立后，剩下的就是通过网络 I/O 操作在两端收发数据了，学习基本网络 I/O 操作的方法与注意事项也必不可少。

最后，任何一端准备发送数据或收到数据后都要对数据进行操作，由于 TCP 是流协议，我们需要了解针对字节的操作。

## TCP Socket 编程基础

TCP Socket 诞生以来，它的编程模型，也就是网络 I/O 模型已几经演化。网络 I/O 模型定义的是应用线程与操作系统内核之间的交互行为模式。我们通常用**阻塞（Blocking）**/**非阻塞（Non-Blocking）**来描述网络I/O模型。

阻塞/非阻塞，是以**内核**是否等数据全部就绪后，才返回（给发起系统调用的应用线程）来区分的。

* 如果内核一直等到全部数据就绪才返回，这种行为模式就称为**阻塞**。
* 如果内核查看数据就绪状态后，即便没有就绪也立即返回错误（给发起系统调用的应用线程），那么这种行为模式则称为**非阻塞**。

常用的网络 I/O 模型包括下面这几种：

* **阻塞 I/O (Blocking I/O)**

  阻塞I/O是最常用的模型，这个模型下应用线程与内核之间的交互行为模式是这样的：

  ![image-20250408091404561](http://images.liangning7.cn/typora/202504080914626.png)

  我们看到，在**阻塞I/O模型**下，当用户空间应用线程，向操作系统内核发起I/O请求后（一般为操作系统提供的I/O系列系统调用），内核会尝试执行这个I/O操作，并等所有数据就绪后，将数据从内核空间拷贝到用户空间，最后系统调用从内核空间返回。而在这个期间内，用户空间应用线程将阻塞在这个I/O系统调用上，无法进行后续处理，只能等待。

  因此，在这样的模型下，一个线程仅能处理一个网络连接上的数据通信。即便连接上没有数据，线程也只能阻塞在对Socket的读操作上（以等待对端的数据）。虽然这个模型对应用整体来说是低效的，但对开发人员来说，这个模型却是最容易实现和使用的，所以，各大平台在默认情况下都将Socket设置为阻塞的。

* **非阻塞I/O（Non-Blocking I/O）**

  非阻塞I/O模型下，应用线程与内核之间的交互行为模式是这样的：

  ![image-20250408092413883](http://images.liangning7.cn/typora/202504080924954.png)

  和阻塞I/O模型正相反，在**非阻塞模型**下，当用户空间线程向操作系统内核发起I/O请求后，内核会执行这个I/O操作，如果这个时候数据尚未就绪，就会立即将“未就绪”的状态以错误码形式（比如：EAGAIN/EWOULDBLOCK），返回给这次I/O系统调用的发起者。而后者就会根据系统调用的返回状态来决定下一步该怎么做。

  在非阻塞模型下，位于用户空间的I/O请求发起者通常会通过轮询的方式，去一次次发起I/O请求，直到读到所需的数据为止。不过，这样的轮询是对CPU计算资源的极大浪费，因此，非阻塞I/O模型单独应用于实际生产的比例并不高。

* **I/O多路复用（I/O Multiplexing）**

  为了避免非阻塞I/O模型轮询对计算资源的浪费，同时也考虑到阻塞I/O模型的低效，开发人员首选的网络I/O模型，逐渐变成了建立在内核提供的多路复用函数select/poll等（以及性能更好的epoll等函数）基础上的**I/O多路复用模型**。

  这个模型下，应用线程与内核之间的交互行为模式如下图：

  ![image-20250408092525087](http://images.liangning7.cn/typora/202504080925157.png)

  从图中我们看到，在这种模型下，应用线程首先将需要进行I/O操作的Socket，都添加到多路复用函数中（这里以select为例），然后阻塞，等待select系统调用返回。当内核发现有数据到达时，对应的Socket具备了通信条件，这时select函数返回。然后用户线程会针对这个Socket再次发起网络I/O请求，比如一个read操作。由于数据已就绪，这次网络I/O操作将得到预期的操作结果。

  我们看到，相比于阻塞模型一个线程只能处理一个Socket的低效，I/O多路复用模型中，一个应用线程可以同时处理多个Socket。同时，I/O多路复用模型由内核实现可读/可写事件的通知，避免了非阻塞模型中轮询，带来的CPU计算资源浪费的问题。

  目前，主流网络服务器采用的都是“I/O多路复用”模型，有的也结合了多线程。不过，**I/O多路复用**模型在支持更多连接、提升I/O操作效率的同时，也给使用者带来了不小的复杂度，以至于后面出现了许多高性能的I/O多路复用框架，比如：[libevent](http://libevent.org/)、[libev](http://software.schmorp.de/pkg/libev.html)、[libuv](https://github.com/libuv/libuv)等，以帮助开发者简化开发复杂性，降低心智负担。

  ## Go 语言 Socket 编程模型

  Go 语言设计者考虑得更多的是 Gopher 的开发体验。阻塞 I/O 模型是对开发人员最友好的，也是心智负担最低的模型，而**I/O多路复用**的这种**通过回调割裂执行流**的模型，对开发人员来说还是过于复杂了，于是 Go 选择了为开发人员提供**阻塞 I/O 模型**，Gopher 只需在Goroutine 中以最简单、最易用的**“阻塞I/O模型”**的方式，进行 Socket 操作就可以了。

  再加上，Go 没有使用基于线程的并发模型，而是使用了开销更小的 Goroutine 作为基本执行单元，这让每个Goroutine处理一个TCP连接成为可能，并且在高并发下依旧表现出色。

  不过，网络I/O操作都是系统调用，Goroutine执行I/O操作的话，一旦阻塞在系统调用上，就会导致M也被阻塞，为了解决这个问题，Go设计者将这个“复杂性”隐藏在Go运行时中，他们在运行时中实现了网络轮询器（netpoller)，netpoller的作用，就是只阻塞执行网络I/O操作的Goroutine，但不阻塞执行Goroutine的线程（也就是M）。

  > 由于 go 的 GPM 模型存在，实现多路复用简直就是降维打击。只挂起 G 而不挂起M，在G的视角里就是个阻塞模型而已

  这样一来，对于Go程序的用户层（相对于Go运行时层）来说，它眼中看到的goroutine采用了“阻塞I/O模型”进行网络I/O操作，Socket都是“阻塞”的。

  但实际上，这样的“假象”，是通过Go运行时中的netpoller **I/O多路复用机制**，“模拟”出来的，对应的、真实的底层操作系统Socket，实际上是非阻塞的。只是运行时拦截了针对底层Socket的系统调用返回的错误码，并通过**netpoller**和Goroutine调度，让Goroutine“阻塞”在用户层所看到的Socket描述符上。

  比如：当用户层针对某个Socket描述符发起`read`操作时，如果这个Socket对应的连接上还没有数据，运行时就会将这个Socket描述符加入到netpoller中监听，同时发起此次读操作的Goroutine会被挂起。

  直到Go运行时收到这个Socket数据可读的通知，Go运行时才会重新唤醒等待在这个Socket上准备读数据的那个Goroutine。而这个过程，从Goroutine的视角来看，就像是read操作一直阻塞在那个Socket描述符上一样。

  而且，Go语言在网络轮询器（netpoller）中采用了I/O多路复用的模型。考虑到最常见的多路复用系统调用select有比较多的限制，比如：监听Socket的数量有上限（1024）、时间复杂度高，等等，Go运行时选择了在不同操作系统上，使用操作系统各自实现的高性能多路复用函数，比如：Linux上的epoll、Windows上的iocp、FreeBSD/MacOS上的kqueue、Solaris上的event port等，这样可以最大程度提高netpoller的调度和执行性能。

  了解完 Go socket 编程模型后，接下来，我们就深入到几个常用的基于socket的网络I/O操作中，逐一了解一下这些操作的机制与注意事项。

## socket 监听 (listen) 与接受连接 (accpet)

socket编程的核心在于服务端，而服务端有着自己一套相对固定的套路：Listen+Accept。在这套固定套路的基础上，我们的服务端程序通常采用一个Goroutine处理一个连接，它的大致结构如下：

[Base Server](https://github.com/LiangNing7/go-tcp/blob/main/base/1.basic/server/main.go)

```go
package main

import (
	"fmt"
	"net"
)

func handleConn(c net.Conn) {
	defer c.Close()

	buf := make([]byte, 10)
	fmt.Println("准备读取...")
	n, err := c.Read(buf)
	if err != nil {
		fmt.Printf("读取发生错误：%+v", err)
		return
	}
	fmt.Printf("已读取 %d 字节", n)
}

func main() {
	l, err := net.Listen("tcp", ":8888")
	if err != nil {
		fmt.Print("listen error:", err)
		return
	}
	fmt.Println("已启动 tcp server")

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			break
		}

		fmt.Println("接受到一个新的连接")

		// start a new goroutine to handleConn
		// the new connection.
		go handleConn(c)
	}
}
```

在这个服务端程序中，我们在第12行使用了net包的Listen函数绑定（bind）服务器端口8888，并将它转换为监听状态，Listen返回成功后，这个服务会进入一个循环，并调用net.Listener的Accept方法接收新客户端连接。

在没有新连接的时候，这个服务会阻塞在Accept调用上，直到有客户端连接上来，Accept方法将返回一个net.Conn实例。通过这个net.Conn，我们可以和新连上的客户端进行通信。这个服务程序启动了一个新Goroutine，并将net.Conn传给这个Goroutine，这样这个Goroutine就专职负责处理与这个客户端的通信了。

而net.Listen函数很少报错，除非是监听的端口已经被占用，那样程序将输出类似这样的错误：

```plain
bind: address already in use
```

当服务程序启动成功后，我们可以通过netstat命令，查看端口的监听情况：

```bash
$ netstat -an|grep 8888    
tcp46       0      0  *.8888                 *.*                    LISTEN     
```

了解了服务端的“套路”后，我们再来看看客户端。

## 向服务端建立 TCP 连接

一旦服务端按照上面的`Listen + Accept`结构成功启动，客户端便可以使用`net.Dial`或`net.DialTimeout`向服务端发起连接建立的请求：

```go
conn, err := net.Dial("tcp", "localhost:8888")
conn, err := net.DialTimeout("tcp", "localhost:8888", 2 * time.Second)
```

Dial函数向服务端发起TCP连接，这个函数会一直阻塞，直到连接成功或失败后，才会返回。而DialTimeout带有超时机制，如果连接耗时大于超时时间，这个函数会返回超时错误。 对于客户端来说，连接的建立还可能会遇到几种特殊情形。

**第一种情况：网络不可达或对方服务未启动。**

如果传给`Dial`的服务端地址是网络不可达的，或者服务地址中端口对应的服务并没有启动，端口未被监听（Listen），`Dial`几乎会立即返回类似这样的错误：

```text
dial error: dial tcp :8888: getsockopt: connection refused
```

**第二种情况：对方服务的listen backlog队列满。**

当对方服务器很忙，瞬间有大量客户端尝试向服务端建立连接时，服务端可能会出现listen backlog队列满，接收连接（accept）不及时的情况，这就会导致客户端的`Dial`调用阻塞，直到服务端进行一次accept，从backlog队列中腾出一个槽位，客户端的Dial才会返回成功。

而且，不同操作系统下backlog队列的长度是不同的，在macOS下，这个默认值如下：

```bash
$ sysctl -a|grep kern.ipc.somaxconn
kern.ipc.somaxconn: 128
```

在Ubuntu Linux下，backlog队列的长度值与系统中`net.ipv4.tcp_max_syn_backlog`的设置有关。

那么，极端情况下，如果服务端一直不执行`accept`操作，那么客户端会一直阻塞吗？

答案是不会！我们看一个实测结果。如果服务端运行在macOS下，那么客户端会阻塞大约1分多钟，才会返回超时错误：

```text
dial error: dial tcp :8888: getsockopt: operation timed out
```

而如果服务端运行在Ubuntu上，客户端的`Dial`调用大约在2分多钟后提示超时错误，这个结果也和Linux的系统设置有关。

参考代码：

[Base Client]()

```go
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
```

## 全双工通信

一旦客户端调用Dial成功，我们就在客户端与服务端之间建立起了一条全双工的通信通道。通信双方通过各自获得的Socket，可以在向对方发送数据包的同时，接收来自对方的数据包。

下图展示了系统层面对这条全双工通信通道的实现原理：

![image-20250408102340816](http://images.liangning7.cn/typora/202504081023924.png)

任何一方的操作系统，都会为已建立的连接分配一个发送缓冲区和一个接收缓冲区。

以客户端为例，客户端会通过成功连接服务端后得到的conn（封装了底层的socket）向服务端发送数据包。这些数据包会先进入到己方的发送缓冲区中，之后，这些数据会被操作系统内核通过网络设备和链路，发到服务端的接收缓冲区中，服务端程序再通过代表客户端连接的conn读取服务端接收缓冲区中的数据，并处理。

反之，服务端发向客户端的数据包也是先后经过服务端的发送缓冲区、客户端的接收缓冲区，最终到达客户端的应用的。

## Socket 读操作

> 注意：完整代码参考后面总结！！！

连接建立起来后，我们就要在连接上进行读写以完成业务逻辑。我们前面说过，Go运行时隐藏了**I/O多路复用**的复杂性。Go语言使用者只需采用**Goroutine+阻塞I/O模型**，就可以满足大部分场景需求。Dial连接成功后，会返回一个net.Conn接口类型的变量值，这个接口变量的底层类型为一个`*TCPConn`：

```go
//$GOROOT/src/net/tcpsock.go
type TCPConn struct {
    conn
}
```

TCPConn内嵌了一个非导出类型：`conn`（封装了底层的socket），因此，TCPConn“继承”了`conn`类型的`Read`和`Write`方法，后续通过`Dial`函数返回值调用的`Read`和`Write`方法都是net.conn的方法，它们分别代表了对socket的读和写。

接下来，我们先来通过几个场景来总结一下Go中从socket读取数据的行为特点。

### 1. Socket 中无数据的场景

连接建立后，如果客户端未发送数据，服务端会阻塞在Socket的读操作上，这和前面提到的“阻塞I/O模型”的行为模式是一致的。执行该这个操作的Goroutine也会被挂起。Go运行时会监视这个Socket，直到它有数据读事件，才会重新调度这个Socket对应的Goroutine完成读操作。

### 2. Socket 中有部分数据

如果Socket中有部分数据就绪，且数据数量小于一次读操作期望读出的数据长度，那么读操作将会成功读出这部分数据，并返回，而不是等待期望长度数据全部读取后，再返回。

举个例子，服务端创建一个长度为10的切片作为接收数据的缓冲区，等待Read操作将读取的数据放入切片。当客户端在已经建立成功的连接上，成功写入两个字节的数据（比如：hi）后，服务端的Read方法将成功读取数据，并返回`n=2，err=nil`，而不是等收满10个字节后才返回。

### 3. Socket 中有足够数据

如果连接上有数据，且数据长度大于等于一次`Read`操作期望读出的数据长度，那么`Read`将会成功读出这部分数据，并返回。这个情景是最符合我们对`Read`的期待的了。

我们以上面的例子为例，当客户端在已经建立成功的连接上，成功写入15个字节的数据后，服务端进行第一次`Read`时，会用连接上的数据将我们传入的切片缓冲区（长度为10）填满后返回：`n = 10, err = nil`。这个时候，内核缓冲区中还剩5个字节数据，当服务端再次调用`Read`方法时，就会把剩余数据全部读出。

### 4. 设置读操作超时

有些场合，对socket的读操作的阻塞时间有严格限制的，但由于Go使用的是阻塞I/O模型，如果没有可读数据，Read操作会一直阻塞在对Socket的读操作上。

这时，我们可以通过net.Conn提供的SetReadDeadline方法，设置读操作的超时时间，当超时后仍然没有数据可读的情况下，Read操作会解除阻塞并返回超时错误，这就给Read方法的调用者提供了进行其他业务处理逻辑的机会。

SetReadDeadline方法接受一个绝对时间作为超时的deadline。一旦通过这个方法设置了某个socket的Read deadline，当发生超时后，如果我们不重新设置Deadline，那么后面与这个socket有关的所有读操作，都会返回超时失败错误。

下面是结合SetReadDeadline设置的服务端一般处理逻辑：

```go
func handleConn(c net.Conn) {
	defer c.Close()

	fmt.Println("准备读取...")

	for {
		var buf = make([]byte, 10)
		c.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, err := c.Read(buf)
		if err != nil {
			fmt.Println(fmt.Sprintf("读取发生错误：%+v", err))
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				fmt.Println(fmt.Sprintf("%s ", time.Now().Format("2006-01-02 15:04:05.000")), "发生读超时")
				continue
			}
			return
		}
		fmt.Println(fmt.Sprintf("%s 已读取 %d 字节", time.Now().Format("2006-01-02 15:04:05.000"), n))
	}

}
```

如果我们要取消超时设置，可以使用`SetReadDeadline(time.Time{})`实现。

> `time.Time{}`是个零值，也就是一个未设置的时间点

## Socket 写操作

通过 net.Conn 实例的 Write 方法，我们可以将数据写入 Socket。当 Write 调用的返回值 n 的值，与预期要写入的数据长度相等，且 err = nil 时，我们就执行了一次成功的 Socket 写操作，这是我们在调用 Write 时遇到的最常见的情形。

和Socket的读操作一些特殊情形相比，Socket 写操作遇到的特殊情形同样不少，我们也逐一看一下。

### 1. 写阻塞

TCP协议通信两方的操作系统内核，都会为这个连接保留数据缓冲区，调用Write向Socket写入数据，实际上是将数据写入到操作系统协议栈的数据缓冲区中。TCP是全双工通信，因此每个方向都有独立的数据缓冲。当发送方将对方的接收缓冲区，以及自身的发送缓冲区都写满后，再调用Write方法就会出现阻塞的情况。

客户端代码如下：

```go
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
		log.Println("dial error:", err)
		return
	}
	defer conn.Close()
	fmt.Println("dial ok")

	data := make([]byte, 65536)
	var total int
	for {
		n, err := conn.Write(data)
		if err != nil {
			total += n
			fmt.Printf("write %d bytes, error: %+v\n", n, err)
			break
		}
		total += n
		fmt.Printf("write %d bytes this time, %d bytes in total\n", n, total)
	}
	fmt.Printf("write %d bytes in total\n", total)
}
```

客户端每次调用Write方法向服务端写入65536个字节，并在Write方法返回后，输出此次Write的写入字节数和程序启动后写入的总字节数量。

服务端代码：

```go
package main

import (
	"fmt"
	"net"
	"time"
)

func handleConn(c net.Conn) {
	defer c.Close()
	time.Sleep(time.Second * 10)
	fmt.Println("准备读取...")

	for {
		time.Sleep(5 * time.Second)
		buf := make([]byte, 60000)
		n, err := c.Read(buf)
		if err != nil {
			fmt.Printf("读取发生错误: %+v\n", err)
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			}
			return
		}
		fmt.Printf("%s 已读取 %d 字节\n", time.Now().Format("2006-01-02 15:04:05.000"), n)
	}
}

func main() {
	l, err := net.Listen("tcp", ":8888")
	if err != nil {
		fmt.Println("listen error:", err)
		return
	}
	fmt.Println("已启动 tcp server")

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			break
		}
		fmt.Println("接收到一个新的连接")
		// start a new goroutine to handle
		// the new connection.
		go handleConn(c)
	}
}
```

我们可以看到，服务端在前10秒中并不读取数据，因此当客户端一直调用Write方法写入数据时，写到一定量后就会发生阻塞。你可以看一下客户端的执行输出：

```text
2025-04-08 16:06:03.123 begin dial...
2025-04-08 16:06:03.123 dial ok
2025-04-08 16:06:03.123 write 65536 bytes this time, 65536 bytes in total
... ...
2025-04-08 16:06:03.123 write 65536 bytes this time, 589824 bytes in total
2025-04-08 16:06:03.123 write 65536 bytes this time, 655360 bytes in total  <-- 之后，写操作将阻塞
```

后续当服务端每隔5秒进行一次读操作后，内核socket缓冲区腾出了空间，客户端就又可以写入了

### 2. 写入部分数据

Write操作存在写入部分数据的情况，比如上面例子中，当客户端输出日志停留在“write 65536 bytes this time, 655360 bytes in total”时，我们杀掉服务端，这时我们就会看到客户端输出以下日志：

```text
...
2025-04-08 16:06:05.293 write 65536 bytes this time, 655360 bytes in total
2025-04-08 16:06:05.293 write 24108 bytes, error:write tcp 127.0.0.1:62245->127.0.0.1:8888: write: broken pipe
2025-04-08 16:06:05.293 write 679468 bytes in total
```

显然，`Write`并不是在655360这个地方阻塞的，而是后续又写入24108个字节后发生了阻塞，服务端Socket关闭后，我们看到客户端又写入24108字节后，才返回的`broken pipe`错误。由于这24108字节数据并未真正被服务端接收到，程序需要考虑妥善处理这些数据，以防数据丢失。

> 注意，这里不是每个平台都是 `write: broken pipe`，看每个平台的底层实现。有可能还有`connection reset by perr`

### 3. 写入超时

如果我们非要给Write操作增加一个期限，可以调用SetWriteDeadline方法。比如，我们可以将上面例子中的客户端源码拷贝一份，然后在新客户端源码中的Write调用之前，增加一行超时时间设置代码：

```go
conn.SetWriteDeadline(time.Now().Add(time.Microsecond * 10))
```

然后先后启动服务端与新客户端，我们可以看到写入超时的情况下，Write方法的返回结果：

![image-20250408135438710](http://images.liangning7.cn/typora/202504081354819.png)

我们可以看到，在Write方法写入超时时，依旧存在**数据部分写入（仅写入24108个字节）**的情况。另外，和SetReadDeadline一样，只要我们通过SetWriteDeadline设置了写超时，那无论后续Write方法是否成功，如果不重新设置写超时或取消写超时，后续对Socket的写操作都将以超时失败告终。

综合上面这些例子，虽然Go给我们提供了阻塞I/O的便利，但在调用`Read`和`Write`时，依旧要综合函数返回的`n`和`err`的结果以做出正确处理。

不过，前面说的Socket读与写都是限于单Goroutine下的操作，如果多个Goroutine并发读或写一个socket会发生什么呢？我们继续往下看。

## 并发 Socket 读写

Goroutine的网络编程模型，决定了存在着不同Goroutine间共享`conn`的情况，那么`conn`的读写是否是Goroutine并发安全的呢？不过，在深入这个问题之前，我们先从应用的角度上，看看并发read操作和write操作的Goroutine安全的必要性。

对于Read操作而言，由于TCP是面向字节流，`conn.Read`无法正确区分数据的业务边界，因此，多个Goroutine对同一个conn进行read的意义不大，Goroutine读到不完整的业务包，反倒增加了业务处理的难度。

但对于Write操作而言，倒是有多个Goroutine并发写的情况。不过conn读写是否是Goroutine安全的测试并不是很好做，我们先深入一下运行时代码，从理论上给这个问题定个性。

首先，`net.conn`只是`*netFD` 的外层包裹结构，最终Write和Read都会落在其中的`fd`字段上：

```go
//$GOROOT/src/net/net.go
type conn struct {
    fd *netFD
}
```

另外，netFD在不同平台上有着不同的实现，我们以`net/fd_posix.go`中的`netFD`为例看看：

```go
// $GOROOT/src/net/fd_posix.go

// Network file descriptor.
type netFD struct {
    pfd poll.FD 
    
    // immutable until Close
    family      int
    sotype      int
    isConnected bool // handshake completed or use of association with peer
    net         string
    laddr       Addr
    raddr       Addr
}  
```

netFD中最重要的字段是poll.FD类型的pfd，它用于表示一个网络连接。我也把它的结构摘录了一部分：

```go
// $GOROOT/src/internal/poll/fd_unix.go

// FD is a file descriptor. The net and os packages use this type as a
// field of a larger type representing a network connection or OS file.
type FD struct {
    // Lock sysfd and serialize access to Read and Write methods.
    fdmu fdMutex
    
    // System file descriptor. Immutable until Close.
    Sysfd int
    
    // I/O poller.
    pd pollDesc 

    // Writev cache.
    iovecs *[]syscall.Iovec
    ... ...    
}
```

我们看到，`FD`类型中包含了一个运行时实现的`fdMutex`类型字段。从它的注释来看，这个`fdMutex`用来串行化对字段`Sysfd`的Write和Read操作。也就是说，所有对这个FD所代表的连接的Read和Write操作，都是由`fdMutex`来同步的。从`FD`的Read和Write方法的实现，也证实了这一点：

```go
// $GOROOT/src/internal/poll/fd_unix.go

func (fd *FD) Read(p []byte) (int, error) {
    if err := fd.readLock(); err != nil {
        return 0, err
    }
    defer fd.readUnlock()
    if len(p) == 0 {
        // If the caller wanted a zero byte read, return immediately
        // without trying (but after acquiring the readLock).
        // Otherwise syscall.Read returns 0, nil which looks like
        // io.EOF.
        // TODO(bradfitz): make it wait for readability? (Issue 15735)
        return 0, nil
    }
    if err := fd.pd.prepareRead(fd.isFile); err != nil {
        return 0, err
    }
    if fd.IsStream && len(p) > maxRW {
        p = p[:maxRW]
    }
    for {
        n, err := ignoringEINTRIO(syscall.Read, fd.Sysfd, p)
        if err != nil {
            n = 0
            if err == syscall.EAGAIN && fd.pd.pollable() {
                if err = fd.pd.waitRead(fd.isFile); err == nil {
                    continue
                }
            }
        }
        err = fd.eofError(n, err)
        return n, err
    }
}

func (fd *FD) Write(p []byte) (int, error) {
    if err := fd.writeLock(); err != nil {
        return 0, err
    }
    defer fd.writeUnlock()
    if err := fd.pd.prepareWrite(fd.isFile); err != nil {
        return 0, err
    }
    var nn int
    for {
        max := len(p)
        if fd.IsStream && max-nn > maxRW {
            max = nn + maxRW
        }
        n, err := ignoringEINTRIO(syscall.Write, fd.Sysfd, p[nn:max])
        if n > 0 {
            nn += n
        }
        if nn == len(p) {
            return nn, err
        }
        if err == syscall.EAGAIN && fd.pd.pollable() {
            if err = fd.pd.waitWrite(fd.isFile); err == nil {
                continue
            }
        }
        if err != nil {
            return nn, err
        }
        if n == 0 {
            return nn, io.ErrUnexpectedEOF
        }
    }
}
```

你看，每次Write操作都是受lock保护，直到这次数据全部写完才会解锁。因此，在应用层面，要想保证多个Goroutine在一个`conn`上write操作是安全的，需要一次write操作完整地写入一个“业务包”。一旦将业务包的写入拆分为多次write，那也无法保证某个Goroutine的某“业务包”数据在`conn`发送的连续性。

同时，我们也可以看出即便是Read操作，也是有lock保护的。多个Goroutine对同一`conn`的并发读，不会出现读出内容重叠的情况，但就像前面讲并发读的必要性时说的那样，一旦采用了不恰当长度的切片作为buf，很可能读出不完整的业务包，这反倒会带来业务上的处理难度。

比如一个完整数据包：`world`，当Goroutine的读缓冲区长度 < 5时，就存在这样一种可能：一个Goroutine读出了“worl”，而另外一个Goroutine读出了”d”。

最后我们再来看看Socket关闭。

## Socket 关闭

通常情况下，当客户端需要断开与服务端的连接时，客户端会调用net.Conn的Close方法关闭与服务端通信的Socket。如果客户端主动关闭了Socket，那么服务端的`Read`调用将会读到什么呢？这里要分“有数据关闭”和“无数据关闭”两种情况。

“有数据关闭”是指在客户端关闭连接（Socket）时，Socket中还有服务端尚未读取的数据。在这种情况下，服务端的Read会成功将剩余数据读取出来，最后一次Read操作将得到`io.EOF`错误码，表示客户端已经断开了连接。如果是在“无数据关闭”情形下，服务端调用的Read方法将直接返回`io.EOF`。

不过因为Socket是全双工的，客户端关闭Socket后，如果服务端Socket尚未关闭，这个时候服务端向Socket的写入操作依然可能会成功，因为数据会成功写入己方的内核socket缓冲区中，即便最终发不到对方socket缓冲区也会这样。【报错：`read: connection reset by peer`】

### connection reset by peer 的产生

假设有如下情况，每次客户端给服务端发送消息，服务端接收消息后，直接写回给客户端，客户端在写入数据后没有调用相应的读取操作，而直接关闭了连接，这时可能已经有部分数据还在服务器的发送缓冲区中。客户端关闭连接时如果发送 RST 包，服务端后续的写操作就会检测到对端已经重置了连接，从而触发“connection reset by peer”错误。

> **正常的连接关闭（FIN 包）：**
>  如果双方按照 TCP 协议的正常流程关闭连接，一般会使用 FIN 包来通知对端关闭写通道。此时，对端在读操作时会得到 EOF（读取返回 0），表明连接已经优雅地关闭，不会再继续尝试写入数据。
>
> **异常关闭（RST 包）：**
>  在你的场景中，客户端发送数据后没有进行数据读取，而是在短暂等待后直接调用 `Close()` 关闭连接。这样，在客户端仍然存在未读取的数据时，对应的操作系统可能会立即发送 RST 包给服务器，表示连接中断。当服务端随后试图写入数据时，因发现对端已重置连接，便会报出“connection reset by peer”错误

## 总结

实现各个操作里的示例代码：

1. [最基本的 TCP server 和 client 写法【观察 server 的打印】](https://github.com/LiangNing7/go-tcp/tree/main/base/1.basic)
2. [Socket 中无数据的场景【观察 server 的打印】](https://github.com/LiangNing7/go-tcp/tree/main/base/2.readnodata)
3. [Socket 中有部分数据的场景](https://github.com/LiangNing7/go-tcp/tree/main/base/3.readpartialdata)
4. [Socket 中有足够数据的场景](https://github.com/LiangNing7/go-tcp/tree/main/base/4.readdata)
5. [设置读操作超时场景](https://github.com/LiangNing7/go-tcp/tree/main/base/5.readtimeout)
6. [设置读操作超时场景——多次连接](https://github.com/LiangNing7/go-tcp/tree/main/base/6.readtimeoutmore)
7. [写阻塞和写入部分数据的场景](https://github.com/LiangNing7/go-tcp/tree/main/base/7.writeblock)
8. [**写入超时**的场景](https://github.com/LiangNing7/go-tcp/tree/main/base/8.writetimeout)
9. [模拟 `read: connection reset by peer` 异常](https://github.com/LiangNing7/go-tcp/tree/main/base/9.connectionresetbypeer)

# go-tcp 实现

## 建立对协议的抽象

程序是对现实世界的抽象。对于现实世界的自定义应用协议规范，我们需要在程序世界建立起对这份协议的抽象。

![image-20250408190743327](http://images.liangning7.cn/typora/202504081907622.png)

## 深入协议字段

这是一个高度简化的、基于二进制模式定义的协议。二进制模式定义的特点，就是采用长度字段标识独立数据包的边界。

在这个协议规范中，请求包和应答包前三个字段都一样，而请求消息包中的第四个字段为载荷【payload】字段，而响应消息包中的第四个字段为响应状态【result】。

1. `totalLength`，类型`uint42`，长度`4`字节，消息总长度；

2. `commandID`，类型`uint8`，长度`1`字节，消息或响应的类型；

   ```go
   const (
       CommandConn   = iota + 0x01 // 0x01，连接请求包
       CommandSubmit               // 0x02，消息请求包
   )
   
   const (
       CommandConnAck   = iota + 0x81 // 0x81，连接请求的响应包
       CommandSubmitAck               // 0x82，消息请求的响应包
   )
   ```

3. `ID`，数字类型`string`，长度`8`字节，消息流水号【顺序累加，步长为 1，循环使用，一对请求和应答消息的流水号必须相同】；

4. 请求消息包中的`payload`，类型字节序列，长度为任意长度，消息的有效载荷，应用层需要的有效数据；

5. 响应消息包中的`result`，类型`uint8`，长度`1`字节，响应状态【`0`：正常，`1`：错误】。

## 建立 Frame 和 Packet 抽象

首先我们要知道，TCP连接上的数据是一个没有边界的字节流，但在业务层眼中，没有字节流，只有各种协议消息。因此，无论是从客户端到服务端，还是从服务端到客户端，业务层在连接上看到的都应该是一个挨着一个的协议消息流。

现在我们建立第一个抽象：**Frame**。每个Frame表示一个协议消息，这样在业务层眼中，连接上的字节流就是由一个接着一个Frame组成的，如下图所示：

![image-20250408195139418](http://images.liangning7.cn/typora/202504081951619.png)

我们的自定义协议就封装在这一个个的Frame中。协议规定了将Frame分割开来的方法，那就是利用每个Frame开始处的totalLength，每个Frame由一个totalLength和Frame的负载（payload）构成，比如你可以看看下图中左侧的Frame结构：

![image-20250408195215420](http://images.liangning7.cn/typora/202504081952602.png)

这样，我们通过Frame header: totalLength就可以将Frame之间隔离开来。

在这个基础上，我们建立协议的第二个抽象：**Packet**。我们将Frame payload定义为一个Packet。上图右侧展示的就是Packet的结构。

Packet就是业务层真正需要的消息，每个Packet由Packet头和Packet Body部分组成。Packet头就是commandID，用于标识这个消息的类型；而ID和payload（packet payload）或result字段组成了Packet的Body部分，对业务层有价值的数据都包含在Packet Body部分。

那么到这里，我们就通过Frame和Packet两个类型结构，完成了程序世界对我们私有协议规范的抽象。接下来，我们要做的就是基于Frame和Packet这两个概念，实现对我们私有协议的解包与打包操作。

## 协议的解包与打包

所谓协议的**解包（decode）**，就是指识别TCP连接上的字节流，将一组字节“转换”成一个特定类型的协议消息结构，然后这个消息结构会被业务处理逻辑使用。

而**打包（encode）**刚刚好相反，是指将一个特定类型的消息结构转换为一组字节，然后这组字节数据会被放在连接上发送出去。

具体到我们这个自定义协议上，解包就是指`字节流 -> Frame`，打包是指`Frame -> 字节流`。你可以看一下针对这个协议的服务端解包与打包的流程图：

![image-20250408200028715](http://images.liangning7.cn/typora/202504082000903.png)

我们看到，TCP流数据先后经过frame decode和packet decode，得到应用层所需的packet数据，而业务层回复的响应，则先后经过packet的encode与frame的encode，写入TCP数据流中。

到这里，我们实际上已经完成了协议抽象的设计与解包打包原理的设计过程了。接下来，我们先来看看私有协议部分的相关代码实现。

## Frame 的实现

[参考](https://github.com/LiangNing7/go-tcp/blob/main/frame/frame.go)

frame 包的职责是提供识别TCP流边界的编解码器。其结构为：

```go
/*
Frame 结构定义：
+----------------+-----------------------+
| frameHeader(4) |    framePayload(packet)  |
+----------------+-----------------------+
frameHeader: 4字节大端序整型，表示帧总长度（含头及 payload ）
framePayload: 实际数据载荷，对应 packet 内容
*/
```

我们可以很容易为这样的编解码器，定义出一个统一的接口类型 `StreamFrameCodec` ：

```go
// FramePayload 表示帧的有效载荷类型(字节切片).
type FramePayload []byte

type SteamFrameCodec interface {
	Encode(io.Writer, FramePayload) error   // 将数据编码为帧格式写入 io.Writer.
	Decode(io.Reader) (FramePayload, error) // 从 io.Reader 解码帧数据返回有效载荷.
}
```

`StreamFrameCodec` 接口类型有两个方法`Encode`与`Decode`。`Encode`方法用于将输入的`Frame payload`编码为一个`Frame`，然后写入`io.Writer`所代表的输出（outbound）TCP流中。而Decode方法正好相反，它从代表输入（inbound）TCP流的`io.Reader`中读取一个完整`Frame`，并将得到的`Frame payload`解析出来并返回。

这里我们我们要先有对 `SteamFrameCodec`接口的具体实现：

```go
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
```

然后分别完成 Encode 和 Decode。

* `Encode(w io.Writer, framePayload FramePayload) error`

  通过调用 `Encode`可以完成对 `FramePayload` 的封装形成 `Frame`并写入`io.Writer`，即给`framePayload`加上`frameHeader(totalLength)`字段。

  ```go
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
  ```

* `Decode(r io.Reader) (FramePayload, error)`

  通过调用 `Decode`，可以完成从`io.Reader`中接收到 `Frame`，先处理 `Frame`的前`4`字节的长度，然后将剩余部分作为`FramePayload`进行返回。

  ```go
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
  ```

  在这段实现中，有三点事项需要我们注意：

  * 网络字节序使用大端字节序（BigEndian），因此无论是Encode还是Decode，我们都是用`binary.BigEndian`；
  * `binary.Read`或`Write`会根据参数的宽度，读取或写入对应的字节个数的字节，这里`totalLen`使用`int32`，那么`Read`或`Write`只会操作数据流中的`4`个字节；
  * 这里没有设置网络`I/O`操作的`Deadline`，`io.ReadFull`一般会读满你所需的字节数，除非遇到`EOF`或`ErrUnexpectedEOF`。

## Packet 的实现

[参考](https://github.com/LiangNing7/go-tcp/blob/main/packet/packet.go)

Packet 的本质也就是 `FramePayload`，Packet有多种类型（这里只定义了Conn、submit、connack、submit ack)。所以我们要先抽象一下这些类型需要遵循的共同接口：

```go
type Packet interface {
    Decode([]byte) error     // []byte -> struct
    Encode() ([]byte, error) //  struct -> []byte
}
```

其中，Decode 是将一段字节流数据解码为一个 Packet 类型，可能是 conn，可能是submit等，具体我们要根据解码出来的 commandID 判断。而 Encode 则是将一个 Packet 类型编码为一段字节流数据。

为了简化，这里只完成了 submit 和 submitack 类型的 Packet 接口实现，其协议大致如下：

协议定义：

```go
/* 
协议定义
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
```

再来看对应的 Decode 和 Encode 的实现

```go
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
```

代码中已经解释的很详细了，这里就不过多赘述了。

这里各种类型的编解码被调用的前提，是明确数据流是什么类型的，因此我们需要在包级提供一个导出的函数Decode，这个函数负责从字节流中解析出对应的类型（根据commandID），并调用对应类型的Decode方法：

```go
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
```

同样，我们也需要包级的Encode函数，根据传入的packet类型调用对应的Encode方法实现对象的编码：

```go
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
```

## 服务端的组装

我们按照每个连接一个Goroutine的模型，给出了典型Go网络服务端程序的结构，这里我们就以这个结构为基础，将Frame、Packet加进来。

服务器的代码调用结构：

```go
// handleConn的调用结构

read frame from conn
    ->frame decode
	    -> handle packet
		    -> packet decode
		    -> packet(ack) encode
    ->frame(ack) encode
write ack frame to conn
```

其服务端代码如下：

```go
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
```

这个程序的逻辑非常清晰，服务端程序监听8888端口，并在每次调用Accept方法后得到一个新连接，服务端程序将这个新连接交到一个新的Goroutine中处理。

## 客户端实现

客户端的`main`函数：

```go
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
```

我们看到，客户端启动了5个Goroutine，模拟5个并发连接。startClient函数是每个连接的主处理函数，我们来看一下：

```go
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
```

* `startClient` 函数启动了两个`Goroutine`，一个负责向服务端发送`submit`消息请求，另外一个`Goroutine`则负责读取服务端返回的响应；
* 负责读取服务端返回响应的`Goroutine`，使用`SetReadDeadline`方法设置了读超时，这主要是考虑该`Goroutine`可以在收到退出通知时，能及时从`Read`阻塞中跳出来。

使用 `Makefile` 进行编译：

```makefile
all: server client

server: cmd/server/main.go
	go build -o _output/ github.com/LiangNing7/go-tcp/cmd/server
client: cmd/client/main.go 
	go build -o _output/ github.com/LiangNing7/go-tcp/cmd/client

clean:
	rm -rf ./_output
```

构建成功后，我们先来启动`server`程序

```bash
$ ./_output/server
server start ok(on *.8888)
```

然后，我们启动client程序，启动后client程序便会向服务端建立5条连接，并发送submit请求，client端的部分日志如下：

```bash
$ ./_output/client
[client 5]: dial ok
[client 1]: dial ok
[client 5]: send submit id = 00000001, payload=credible-deathstrike-33e1, frame length = 38
[client 3]: dial ok
[client 1]: send submit id = 00000001, payload=helped-lester-8f15, frame length = 31
[client 4]: dial ok
[client 4]: send submit id = 00000001, payload=strong-timeslip-07fa, frame length = 33
[client 3]: send submit id = 00000001, payload=wondrous-expediter-136e, frame length = 36
[client 5]: the result of submit ack[00000001] is 0
[client 1]: the result of submit ack[00000001] is 0
[client 3]: the result of submit ack[00000001] is 0
[client 2]: dial ok
... ...
[client 3]: send submit id = 00000010, payload=bright-monster-badoon-5719, frame length = 39
[client 4]: send submit id = 00000010, payload=crucial-wallop-ec2d, frame length = 32
[client 2]: send submit id = 00000010, payload=pro-caliban-c803, frame length = 29
[client 1]: send submit id = 00000010, payload=legible-shredder-3d81, frame length = 34
[client 5]: send submit id = 00000010, payload=settled-iron-monger-bf78, frame length = 37
[client 3]: the result of submit ack[00000010] is 0
[client 4]: the result of submit ack[00000010] is 0
[client 1]: the result of submit ack[00000010] is 0
[client 2]: the result of submit ack[00000010] is 0
[client 5]: the result of submit ack[00000010] is 0
[client 4]: exit ok
[client 1]: exit ok
[client 3]: exit ok
[client 5]: exit ok
[client 2]: exit ok
```

client 在每条连接上发送10个submit请求后退出。这期间服务端会输出如下日志：

```bash
recv submit: id = 00000001, payload=credible-deathstrike-33e1
recv submit: id = 00000001, payload=helped-lester-8f15
recv submit: id = 00000001, payload=wondrous-expediter-136e
recv submit: id = 00000001, payload=strong-timeslip-07fa
recv submit: id = 00000001, payload=delicate-leatherneck-4b12
recv submit: id = 00000002, payload=certain-deadpool-779d
recv submit: id = 00000002, payload=clever-vapor-25ce
recv submit: id = 00000002, payload=causal-guardian-4f84
recv submit: id = 00000002, payload=noted-tombstone-1b3e
... ...
recv submit: id = 00000010, payload=settled-iron-monger-bf78
recv submit: id = 00000010, payload=pro-caliban-c803
recv submit: id = 00000010, payload=legible-shredder-3d81
handleConn: frame decode error: EOF
handleConn: frame decode error: EOF
handleConn: frame decode error: EOF
handleConn: frame decode error: EOF
handleConn: frame decode error: EOF
```

从结果来看，我们实现的服务端运行正常！

# go-tcp 扩展

## 面试题：Golang中服务端建立 TCP 的上线

有三点需要进行考虑：在 OS 中，每个 TCP 连接都需要占用一个文件描述符；在golang中，每个TCP连接会占用一个 `goroutine`；考虑内存空间，每个连接至少占用数 KB 内存（读写缓冲区 + Goroutine 开销）。

1. 文件描述符【FD】：每个 TCP 连接占用一个 FD，操作系统默认 FD 数量较少，需要手动调整以支持大规模连接。大多数系统允许通过 `ulimit -n` 命令或者修改内核参数来提高这一限制。
2. `goroutine`：虽然 goroutine 轻量，但成千上万的 goroutine 累加起来也会消耗大量内存，调度器的负载也需关注。
3. 内存：一个 goroutine 的栈内存（启动时小、按需增长）；与连接相关的网络缓冲区（读写缓冲区通常是按需分配的，并且可以通过合理的调优减少浪费）。


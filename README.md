## Disruptor

### 1. 简介
#### 1.1. 为什么要写Disruptor
在go语言中一般都是使用chan作为消息传递的队列，但在实际高并发的环境下使用发现chan存在严重的性能问题，其直接表现就是将对象放入到chan中时会特别耗时，
即使chan的容量尚未打满，在严重时甚至会产生几百ms还无法放入到chan中的情况。

经过源码的查看，chan实际对应的结构是`runtime.hchan`，其结构中包含了一个lock字段：`lock mutex`。
这个lock看名字就知道是一个锁，当然它不是我们业务中经常使用的sync.Mutex，而是一个`runtime.mutex`。
这个锁是一个互斥锁，在linux系统中它的实现是futex，在没有竞争的情况下，会退化成为一个自旋操作，速度非常快，但是当竞争比较大时，它就会在内核中休眠。
注意，此处是在内核中休眠，而与runtime.Mutex不同，后者其实是通过gopark()方式将当前g调度出去了，从P中换了一个其他g执行。
因此，当竞争比较大时，chan的性能是比较低的，很难支持对性能要求比较高的业务。

#### 1.2. go-disruptor
众所众知，在java中有一个比较出名的高性能无锁队列：Disruptor。虽然在go中也有一个，但是经过实际测试，发现其性能很一般，并且不支持并发写入。
因此就萌生了go语言版本的Disruptor想法。

在实际编写时参考了Disruptor的很多想法，整体而言，该库有以下几个特点：
##### 1）整个库实现了绝对意义上的无锁，所有的操作都是通过原子变量操作完成，性能上有很大提升；
##### 2）消费者只有一个g操作，屏蔽掉读操作竞争带来的性能损耗；
##### 3）基于"写优先"原则，在往队列中写入时尽可能更快的写进去；
##### 4）available数组用于标识ringbuffer中元素的可用状态，为提高性能使用unsafe.Pointer操作，降低切片寻址的损耗；
##### 5）使用预分配ringbuffer环形数组来填充对象，防止频繁的内存申请，并限制gc回收；
##### 6）使用缓存行填充模型，防止由于cpu多级缓存带来的伪共享问题；
##### 7）buffer大小必须是2的指数倍，这么做主要是为了使用 &运算代替 %运算，提高性能；

### 2. 核心概念
整体的disruptor模型如下所示：
<img src="images/total.png">

##### ringBuffer
具体对象的存放区域，通过数组（定长切片）实现环状数据结构，其中的数据对象是具体的结构体而非指针，这样可以一次性进行内存申请。

##### available
切片实现的map，通过index（或pos）标识每个位置为0或1，当长时间无法读取时会通过blockC进行阻塞，写线程完成时可释放该blockC。
其内部buf实际是[]uint8，但由于[]uint8切片在寻址时会进行游标是否越界的判断，造成性能下降，因此通过使用unsafe.Pointer直接对对应的值进行操作，从而避免越界判断，提升性能。
之所以使用uint8数组而不是使用的bitmap，主要是考虑到写并发的行为，防止bit操作导致数据异常（或靠锁解决）。

##### sequencer
序号产生器，维护读和写两个状态，写状态具体由内部游标（cursor）维护，读取状态由自身维护，一个uint64变量维护。它的核心方法是next()，用于获取下个可以写入的游标。

##### Producer
生产者，核心方法是Write，通过调用Write方法可以将对象写入到队列中。支持多个g并发操作，保证加入时处理的效率。

##### consumer
消费者，这个消费者只会有一个g操作，这样处理的好处是可以不涉及并发操作，其内部不会涉及到任何锁，对于实际的并发操作由该g进行分配。

##### waitStrategy
等待策略，该策略用于获取写入可用的sequence时进行的等待。默认提供了两个实现，SchedWaitStrategy和SleepWaitStrategy，前者使用runtime.Gosched()，后者使用time.Sleep()实现。
推荐使用SchedWaitStrategy，也可以自己实现。

##### EventHandler
事件处理器接口，整个项目中唯一需要用户实现的接口，该接口描述消费端收到消息时该如何处理，它使用泛型，通过编译阶段确定事件类型，提高性能。

### 3. 使用方式

#### 3.1. 导入模块
可使用 `go get github.com/bruceshao/lockfree` 获取最新版本

#### 3.2. 代码调用
为了提升性能，Disruptor支持go版本1.18及以上，以便于支持泛型，Disruptor使用非常简单：
```go
func main() {
    var (
        goSize    = 10
        sizePerGo = 10
        counter   = uint64(0)
    )
    // 创建事件处理器
    eh := &longEventHandler[uint64]{}
	// 创建消费端串行处理的Disruptor
    disruptor := lockfree.NewSerialDisruptor[uint64](1024*1024, eh, &lockfree.SchedWaitStrategy{})
    // 启动Disruptor
	if err := disruptor.Start(); err != nil {
        panic(err)
    }
	// 获取生产者对象
    producer := disruptor.Producer()
    var wg sync.WaitGroup
    wg.Add(goSize)
    for i := 0; i < goSize; i++ {
        go func() {
            for j := 0; j < sizePerGo; j++ {
                x := atomic.AddUint64(&counter, 1)
				// 写入数据
                err := producer.Write(x)
                if err != nil {
                    panic(err)
				}
            }
            wg.Done()
        }()
    }
    wg.Wait()
    fmt.Println("----- write complete -----")
    time.Sleep(time.Second * 1)
	// 关闭Disruptor
    disruptor.Close()
}

type longEventHandler[T uint64] struct {
}

func (h *longEventHandler[T]) OnEvent(v uint64) {
    fmt.Printf("value = %v\n", v)
}
```

### 4. 性能对比

整体上来看，Disruptor在写入和读取上的性能大概都在channel的7倍以上，数据写入的越多，性能提升越明显。
下面是buffer=1024*1024时，写入数据的耗时对比：

<img src="images/time.png">


整体测试的对比情况如下表所示：

写入=500,000：

| 对比类型   | channel | disruptor |
| ---------- | ------- | --------- |
| 总写入耗时 | 97ms    | 39ms      |
| 总读取耗时 | 99ms    | 42ms      |
| <1us       | 426198  | 490307    |
| 1-10us     | 48630   | 8340      |
| 10-100us   | 24835   | 1255      |
| 100-1000us | 327     | 94        |
| 1-10ms     | 6       | 4         |
| 10-100ms   | 4       | 0         |
| >100ms     | 0       | 0         |


写入=1,000,000：

| 对比类型   | channel | disruptor |
| ---------- | ------ | ------ |
| 总写入耗时 | 187ms  | 57ms   |
| 总读取耗时 | 192ms  | 64ms   |
| <1us       | 843858 | 980004 |
| 1-10us     | 104287 | 17513  |
| 10-100us   | 51598  | 2343   |
| 100-1000us | 217    | 131    |
| 1-10ms     | 20     | 9      |
| 10-100ms   | 20     | 0      |
| >100ms     | 0      | 0      |


写入=10,000,000：

| 对比类型   | channel | disruptor |
| ---------- | ------- | ------- |
| 总写入耗时 | 3868ms  | 974ms   |
| 总读取耗时 | 3906ms  | 997ms   |
| <1us       | 1007273 | 6405519 |
| 1-10us     | 117192  | 23298   |
| 10-100us   | 50303   | 47347   |
| 100-1000us | 8822466 | 3519377 |
| 1-10ms     | 2714    | 3083    |
| 10-100ms   | 39      | 1376    |
| >100ms     | 13      | 0       |



写入=50,000,000：

| 对比类型   | channel | disruptor |
| ---------- | -------- | -------- |
| 总写入耗时 | 24237ms  | 3700ms   |
| 总读取耗时 | 24274ms  | 3716ms   |
| <1us       | 990905   | 40485785 |
| 1-10us     | 119376   | 30654    |
| 10-100us   | 48902    | 19052    |
| 100-1000us | 530      | 466781   |
| 1-10ms     | 48835376 | 8987742  |
| 10-100ms   | 4889     | 9986     |
| >100ms     | 22       | 0        |



写入=100,000,000：

| 对比类型   | channel | disruptor |
| ---------- | -------- | -------- |
| 总写入耗时 | 54145ms  | 7335ms   |
| 总读取耗时 | 54186ms  | 7357ms   |
| <1us       | 1117019  | 88333884 |
| 1-10us     | 76828    | 46109    |
| 10-100us   | 33322    | 43460    |
| 100-1000us | 1504     | 630901   |
| 1-10ms     | 98746320 | 9701375  |
| 10-100ms   | 24960    | 1244271  |
| >100ms     | 47       | 0        |


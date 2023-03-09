# VM-FT -- 容错虚拟机

### VM-FT 核心：主备份

实现主备份的两种方法：

- **State transfer（状态转移）**【传输数据量多，所需带宽大】
- **State Machine（备份状态机）**【不确定操作干扰，复杂，但传输数据少】

备份状态机=>***确定性重放replay***：允许记录主服务器的执行，并确保备份以相同方式执行。容错(FT)是基于确定性重放的

![](https://github.com/yanyanran/pictures/blob/main/VM-FT.png?raw=true)

Primary接收输入信息=>通过Logging channel传输给Backup来保持同步



#### 容错FT设计

VM的虚拟磁盘位于共享存储上【主/备VM都可以访问其输入和输出】

但在网络层面只有主VM通知其在网络上的存在【所有网络输出都进入主VM】

主VM收到输入=>日志通道的网络连接=>发到备VM【此时主备VM的执行相同】

但备VM的输出会被删除=>只有主VM返回给客户机实际输出



#### FT容错协议

主机失败【比如说主机在输出后，还没等到备机执行到同样的操作就立刻故障了，此时就会出现主备状态不一致的情况】=> 使用Output Rule =>

![](https://github.com/yanyanran/pictures/blob/main/Output%20Rule.png?raw=true)

primary把所有输出信息都发送给backup后，backup会发送一个**ACK**表示自己已经收到，此时primary才会把输出发送给外界。这步是**异步**执行的【指延迟发送】，不会影响后续操作的进度和时间。



#### 关于日志

日志很重要，但primary和backup在生产和消费日志的时候不是直接写到磁盘里面的。

它们各自有一个log buffer缓存，操作日志的时候主要靠这个log buffer通过logging channel传输**=>**backup再从自己的缓存中读日志并replay ：

![](https://github.com/yanyanran/pictures/blob/main/Logging.png?raw=true)

系统会通过**提高/降低CPU资源**从而实现这俩log buffer的负载均衡



#### 脑裂问题（split-brain）

primary和backup间有UDP的心跳检测来监测对方是不是挂了

脑裂问题就是：backup长时间没有接收到来自primary的心跳包【但此时primary可能只是网络延迟了】，此时backup上前顶替primary，primary又突然恢复了，相当于此时有两个master去操作共享内存，后果难搞

VM-TF的解决方法是：创建一个类似于锁的flag原子操作，谁拿到这个锁谁才能去操作磁盘数据：

```go
func test-and-set() {
	acquire lock()
	if flag == true:
	    release lock()
	    return false
	 else:
	    flag = true
	    release lock()
        return true
}  // 就是锁嘛
```


### NSQ源码阅读

- **执行主逻辑**

在nsq/apps/nsqd/main.go中启动service。

通过第三方svc包进行优雅后台进程管理。svc.Run-> svc.Init-> svc.Start，完成初始化配置（`opts，cfg`）-> 加载历史数据（`nsqd.LoadMetadata`）-> 持久化最新数据（`nsqd.PerisistMetadata`）后开启协程进入主逻辑**nsqd.Main**函数。
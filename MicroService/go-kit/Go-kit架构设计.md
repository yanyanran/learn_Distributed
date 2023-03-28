# Go-kit架构设计

go-kit是go包的集合，帮助构建微服务，可以理解为一个工具包。

> Go kit 服务分为三层：
>
> - **Transport 传输层**
>
>   1、接收用户请求，把数据转换为endpoint数据格式；（decodeHttpRequest
>
>   2、把endpoint返回值封装返回给用户。（encodeHttpRespone
>
> - **Endpoint 端点层**
>
>   类似于handler，用于接收传输层请求
>
> - **Service服务层**
>
>   通常将多个endpoint组合在一起，通过实现接口的方式实现业务逻辑
>
>   (可通过编写中间件来添加额外的功能


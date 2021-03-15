# v2ray-heal

> v2ray-heal 是一个v2ray节点检测并提供订阅的服务

## 如何使用

### 配置文件

启动程序需要 `config.yaml` 文件。你可以参考 `.config.yaml` 模板。

### 启动

```go
# 直接启动
go run .
# 或者编译后启动
go build -o vh .
./vh
```

### 维护订阅
首先你需要手动导入从网络中获得的订阅地址，提交到 `ip:port/pub` 中即可。

一个示例: 

```curl
curl --location --request POST 'https://x.xuthus.cc/pub' \
--header 'Content-Type: application/json' \
--data-raw '{
    "remark": "xxx",
    "sub_link": "https://xxx.xxx/link/haiFQxIarPQMAG2Y?sub=3&extend=1"
}'
```


### 获得可用订阅
而后在v2ray客户端订阅地址中填写 `ip:port/sub` 即可获取订阅链接。
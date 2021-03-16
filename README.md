# v2ray-heal

> v2ray-heal 是一个v2ray节点检测并提供订阅的服务

`v2ray-heal` 不能直接为你提供v2ray订阅，需要你运行程序后，手动从网上获取订阅地址，添加到pub接口中，pub接口通过检测节点信息，获取可用的优质节点后，通过sub接口订阅。

过程: 网上获取订阅地址 --> 通过pub接口推送到程序中 --> 通过sub接口订阅可用节点

## 如何使用

### 配置文件

启动程序需要 `config.yaml` 文件。你可以参考 `.config.yaml` 模板。

### 启动

```bash
# 直接启动
go run .
# 或者编译后启动
go build -o vh .
./vh
```

### 维护订阅(pub)

首先你需要手动导入从网络中获得的订阅地址，提交到 `ip:port/pub` 中，`v2ray-heal` 会按照配置文件中指定的频率定时刷新检测可用节点。

每一次提交需要使用POST请求，且指定两个参数 `remark 标记名称` 和 `sub_link 订阅地址`

```json
{
    "remark": "xxx",
    "sub_link": "https://xxx.xxx/xxx"
}
```

一个提交示例: 

```curl
curl --location --request POST 'https://x.xuthus.cc/pub' \
--header 'Content-Type: application/json' \
--data-raw '{
    "remark": "xxx",
    "sub_link": "https://xxx.xxx/link/haiFQxIarPQMAG2Y?sub=3&extend=1"
}'
```

你的每一次提交，都会触发 `v2ray-heal` 的更新操作。如此一来，只要你能从网上找到免费且稳定的订阅地址，都可以将其推送到该程序中，

### 获得可用订阅(sub)

在v2ray客户端订阅配置中填写 `ip:port/sub` 即可获取可用订阅链接。

其中，你可以通过指定 `best` 参数来获取最优节点，即 `ip:port/sub?best=true`
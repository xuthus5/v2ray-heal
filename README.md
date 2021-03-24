# v2ray-heal

> v2ray-heal 是一个v2ray节点检测并提供订阅的服务

`v2ray-heal` 不能直接为你提供v2ray订阅，需要你运行程序后，手动从网上获取订阅地址，添加到pub接口中，pub接口通过检测节点信息，获取可用的优质节点后，通过sub接口订阅。

过程: 网上获取订阅地址 --> 通过pub接口推送到程序中 --> 通过sub接口订阅可用节点

**注:** 需要说明的是，即使服务端返回了订阅节点列表，由于测试的网络环境不同，他依然有可能在你的机器上无法连通。

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

如果你期望使用docker进行工作，这完全有可能。
```shell
docker run -d \
	--network=host \
	-p 2131:2131 \
	--name v2ray-heal \
	-e "time_interval=30" \ 
	-e "token=abcdefg" \
	xuthus5/vh
```

**docker环境变量:**

`time_interval`: 检测时间间隔 默认不填写时 60分钟检测一次

`token`: 订阅节点是否需要token(请提防订阅地址泄露) 默认不填

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

如果你配置了token访问，请在获取订阅时指定 `token` 参数，即 `ip:port/sub?token=xxxxx`

### 下期更新

> 订阅防ban (由于批量ping操作, 部分付费节点出现被运营商ban的情况)
# 初代部署工具 Emu

Emu (/ˈiːmjuː/; Dromaius novaehollandiae) 是鸸鹋的意思。好喜欢鸸鹋啊！！！！


# 编译

运行 go build 即可。

# 配置

运行编译后的可执行文件，首次将生成一个默认配置 `config.yaml`

```
./emu
```

示例如下：

```yaml
// 管理页面会有 basic 验证，这里是用户名和密码
accounts: 
  - username: admin
    password: admin
port: 8080
// 环境名，staging/release，改变会影响日志文件名
mode: staging 
// 服务列表
services:
  - name: 通知系统 lophorina
    exec: lophorina
    env: []
    args:
      - "-conf"
      - "local.staging.yaml"
```

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
  - username: letjoy
    password: letjoy
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

上面这个配置有一个 lophorina 服务，且服务可执行文件名叫 lophorina。你需要保证 service/ 目录下有一个 loporina 文件。在 emu 启动时，lophorina 会自动启动。

# 使用

运行成功后，打开 8080 端口。可以看到服务列表页面，可上传新服务可执行文件进行更新服务。

# 博客API服务

这是一个使用Go语言开发的博客API服务。

## 项目结构

```
blog-api
  ├── cmd/                 # 命令行入口
  ├── config/              # 配置文件
  ├── internal/            # 内部包
  │   ├── config/          # 配置处理
  │   ├── controller/      # 控制器
  │   ├── database/        # 数据库操作
  │   ├── dto/             # 数据传输对象
  │   ├── logger/          # 日志处理
  │   ├── middleware/      # 中间件
  │   ├── model/           # 数据模型
  │   ├── router/          # 路由
  │   └── service/         # 业务逻辑
  ├── logs/                # 日志文件
  ├── pkg/                 # 公共包
  │   ├── auth/            # 认证相关
  │   ├── cache/           # 缓存相关
  │   └── response/        # 响应处理
  ├── .air.toml            # Air热加载配置
  ├── go.mod               # Go模块定义
  ├── main.go              # 应用入口
  └── Makefile             # 构建脚本
```

## 开发指南

### 环境要求

- Go 1.18+
- MySQL
- Redis
- Elasticsearch (可选)

### 使用Air热加载

这个项目使用[Air](https://github.com/cosmtrek/air)进行热加载，大大提高开发效率。

1. 安装Air:

```bash
go install github.com/cosmtrek/air@latest
```

2. 使用Makefile启动开发模式（自动安装Air）:

```bash
make dev
```

这将启动应用程序并监视文件更改。当您修改Go代码时，应用程序将自动重新编译和重启。

### 其他常用命令

构建项目:
```bash
make build
```

运行测试:
```bash
make test
```

清理临时文件:
```bash
make clean
```

## API文档

待补充...

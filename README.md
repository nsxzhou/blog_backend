# Blog API

博客 API 后端服务

## 配置管理

### 敏感信息配置

本项目使用配置模板 + 环境变量的方式管理敏感信息：

1. **复制配置模板**：

   ```bash
   cp config/config.yaml.template config/config.yaml
   ```

2. **编辑环境变量**：
   在 `config.yaml` 文件中填入真实的配置信息：

   ```bash
   # 腾讯云COS配置
   COS_SECRET_ID=AKIDxxxxxxxxxxxx
   COS_SECRET_KEY=xxxxxxxxxxxxxxxx
   COS_BUCKET_URL=https://your-bucket.cos.region.myqcloud.com
   COS_REGION=ap-guangzhou
   COS_BUCKET=your-bucket
   COS_URL_PREFIX=https://your-bucket.cos.region.myqcloud.com

   # 其他敏感配置
   MYSQL_PASSWORD=your_mysql_password
   REDIS_PASSWORD=your_redis_password
   JWT_SECRET_KEY=your_jwt_secret_key
   ```

### 生产环境部署

在生产环境中，建议：

1. 使用环境变量而非配置文件
2. 使用云服务商的密钥管理服务
3. 不要在代码中硬编码任何敏感信息

## 快速开始

### 使用案例

以下是从零开始运行博客 API 的完整流程：

```bash
# 1. 克隆项目
git clone <repository-url>
cd blog_api

# 2. 复制配置模板
cp config/config.yaml.template config/config.yaml

# 3. 编辑配置文件，填入真实的数据库和存储配置
vim config/config.yaml

# 4. 初始化数据库表结构
go run main.go db init-tables

# 5. 创建管理员用户
go run main.go user create-admin
# 按提示输入用户名、邮箱和密码

# 6. 启动HTTP服务
go run main.go serve
# 服务将在 http://localhost:8080 启动
```

### 常用命令

可查看 makefile 文件

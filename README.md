# community-forum-backend
社区服务后台（Go + Gin + GORM + MySQL + Redis + JWT）

## 运行方式

### 1. 本地直接运行

需要本机已启动 MySQL（库名 `community_forum`）与 Redis。

1. 首次使用请先创建本地配置：`cp configs/config.example.yaml configs/config.yaml`
   （`configs/config.yaml` 已被 .gitignore 排除，不会入库，请填入本机真实数据库账号密码与 JWT 密钥）
2. 启动服务：`go run main.go`，默认监听 `:8080`。

若 `configs/config.yaml` 不存在，程序会自动回退加载 `configs/config.example.yaml`（占位值，仅供打通流程），并打印提示。

### 2. Docker Compose（容器化运行，推荐）

前置条件：已安装 Docker Desktop 并启动。

```bash
# 构建镜像并后台启动 app + mysql + redis 三个容器
docker compose up --build -d

# 查看容器状态（三个服务均应为 healthy）
docker compose ps

# 查看 app 日志
docker compose logs -f app

# 验证健康检查接口
curl http://localhost:8080/api/v1/posts/search
```

停止与清理：

```bash
docker compose down          # 停止并删除容器（保留数据卷）
docker compose down -v       # 连同 mysql/redis 数据卷一并删除
```

配置说明：

- `configs/config.yaml` 在构建镜像时被打包进镜像，但其值只是"默认值"；
- 运行期由 docker-compose.yml 注入的环境变量**覆盖** YAML 值，优先级：**环境变量 > YAML**；
- 生产/联调时请在项目根目录创建 `.env` 文件显式设置 `JWT_SECRET`（参考 `.env.example`），
  compose 会自动读取，未设置时使用内置弱默认值 `your-secret-key-change-in-production`。

### 环境变量覆盖表

| 环境变量 | 对应配置项 | 默认来源 |
|---|---|---|
| SERVER_PORT | server.port | YAML |
| DB_HOST / DB_PORT / DB_USER / DB_PASSWORD / DB_NAME | mysql.host / port / username / password / dbname | YAML |
| REDIS_HOST / REDIS_PORT / REDIS_PASSWORD | redis.host / port / password | YAML |
| JWT_SECRET / JWT_ACCESS_EXP_HOUR / JWT_REFRESH_EXP_DAY | jwt.secret / access_exp_hour / refresh_exp_day | YAML |
| LOG_LEVEL | log.level | YAML |

规则：变量未设置、值为空或（对数字项）解析失败时，保持 YAML 中的原值。

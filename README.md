# MicroShop - 微服务电商平台示例项目

一个完整的 Go 微服务项目示例，包含以下技术栈：

## 技术栈

| 组件 | 技术 | 用途 |
|------|------|------|
| **HTTP 框架** | Gin | API Gateway |
| **RPC 框架** | gRPC | 服务间通信 |
| **Actor 模型** | ProtoActor | 分布式消息处理 |
| **服务发现** | etcd | 服务注册与发现 |
| **缓存** | Redis | 分布式缓存 |
| **关系数据库** | MySQL | 主要数据存储 |
| **文档数据库** | MongoDB | 审计日志 |
| **API 文档** | Swagger | API 文档生成 |
| **配置管理** | Viper | 配置文件管理 |
| **链路追踪** | Jaeger | 分布式追踪 |
| **监控** | Prometheus + Grafana | 监控和可视化 |

## 项目结构

```
microshop/
├── cmd/                    # 服务入口
│   ├── user/              # 用户服务
│   ├── order/             # 订单服务
│   ├── gateway/           # API Gateway
│   └── actor/             # Actor 服务
├── pkg/                   # 公共包
│   ├── config/           # 配置管理
│   ├── discovery/        # 服务发现
│   ├── grpc/             # gRPC 服务器
│   ├── models/           # 数据模型
│   └── repository/       # 数据访问层
├── gateway/              # Gateway 实现
├── protos/               # Protobuf 定义
├── config/               # 配置文件
├── scripts/              # 脚本文件
├── deployments/          # 部署配置
├── docs/                 # Swagger 文档
├── docker-compose.yml    # Docker Compose
├── Makefile             # 构建脚本
└── go.mod               # Go 模块
```

## 快速开始

### 1. 启动基础设施

```bash
make docker-up
```

这将启动：
- etcd (2379)
- Redis (6379)
- MySQL (3306)
- MongoDB (27017)
- Jaeger (16686)
- Prometheus (9090)
- Grafana (3000)

### 2. 安装依赖

```bash
make deps
```

### 3. 生成 Protobuf 文件

```bash
make proto
```

### 4. 构建服务

```bash
make build
```

### 5. 运行服务

```bash
# 运行用户服务
make run-user

# 运行订单服务
make run-order

# 运行 Actor 服务
make run-actor

# 运行 API Gateway
make run-gateway
```

## API 端点

### Gateway (http://localhost:8080)

#### 用户服务

```
POST   /api/v1/users          - 创建用户
GET    /api/v1/users/:id      - 获取用户
GET    /api/v1/users          - 列出用户
PUT    /api/v1/users/:id      - 更新用户
DELETE /api/v1/users/:id      - 删除用户
```

#### 订单服务

```
POST   /api/v1/orders         - 创建订单
GET    /api/v1/orders/:id     - 获取订单
GET    /api/v1/orders         - 列出订单
PUT    /api/v1/orders/:id/status - 更新订单状态
```

#### Swagger 文档

```
http://localhost:8080/swagger/index.html
```

## 服务发现

服务使用 etcd 进行服务注册和发现：

```
/microshop/services/
├── user-service/
│   └── 0.0.0.0:50051
└── order-service/
    └── 0.0.0.0:50052
```

## Actor 服务

ProtoActor 服务提供分布式消息处理能力：

- **OrderActor**: 处理订单相关消息
- **NotificationActor**: 处理通知发送
- **OrderClusterActor**: 分布式订单处理

## 数据库初始化

```bash
# 连接到 MySQL
docker exec -it microshop-mysql mysql -uroot -p123456

# 执行初始化脚本
source /docker-entrypoint-initdb.d/init.sql
```

## 监控

- **Jaeger UI**: http://localhost:16686
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)

## 配置

配置文件位于 `config/config.yaml`，支持：

- 服务器配置
- 数据库连接
- Redis 配置
- etcd 配置
- 日志配置

## 开发

```bash
# 运行测试
make test

# 运行测试并生成覆盖率报告
make test-coverage

# 代码格式化
make fmt

# 代码检查
make lint
```

## 清理

```bash
# 停止所有服务
make docker-down

# 清理构建产物
make clean
```

## License

MIT License

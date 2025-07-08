# Go Gin 项目架构文档

## 文档概览

本文档提供了 Go Gin 项目的完整架构说明，包括系统设计、技术选型、部署方案等详细信息。

## 文档结构

### 📋 [架构说明文档](架构说明.md)

详细的项目架构说明，包含：

- 项目概述和技术栈
- 分层架构设计
- 核心组件分析
- 配置管理
- 部署方式
- 扩展性分析

### 📊 架构图表

#### 1. 整体架构图

展示了项目的完整架构，包括客户端、负载均衡、应用服务器、中间件、数据存储、监控系统等各个层次的组件及其关系。

**主要组件：**

- 应用服务器集群 (端口8080)
- 管理服务器 (端口8081)
- 中间件层 (认证、限流、熔断器、安全、日志)
- 多种数据存储 (MySQL、PostgreSQL、Redis、MongoDB、ClickHouse、TDengine、Elasticsearch、ManticoreSearch)
- 消息队列 (Kafka、NSQ)
- 监控系统 (Prometheus、Grafana、ELK Stack)

#### 2. 应用程序启动流程图

详细展示了应用程序从启动到运行的完整流程：

- Bootstrap初始化过程
- 外部服务初始化
- 服务器启动过程
- 后台任务启动
- 优雅关闭设置

#### 3. HTTP请求处理流程图

展示了HTTP请求从客户端到服务器的完整处理流程：

- 负载均衡和路由
- 中间件处理链
- 业务逻辑处理
- 数据访问和缓存
- 响应返回

#### 4. 数据库集成架构图

展示了项目对多种数据库的集成方案：

- 连接池管理
- 多数据库支持
- ORM框架集成
- 缓存策略
- 监控告警

#### 5. 部署架构图

展示了生产环境的完整部署架构：

- Kubernetes集群部署
- 数据库集群配置
- 监控系统部署
- CI/CD流水线

## 核心特性

### 🚀 高性能

- Gin框架提供高性能HTTP服务
- 连接池优化数据库访问
- Redis缓存提升响应速度
- 熔断器防止级联故障

### 🔒 高安全性

- JWT/Casbin双重认证机制
- 多层限流保护
- 安全头部和CORS配置
- 输入验证和SQL注入防护

### 📈 高可扩展性

- 微服务架构设计
- 水平扩展支持
- 多数据库支持
- 插件化中间件

### 🔍 高可观测性

- Prometheus监控指标
- 结构化日志记录
- 分布式链路追踪
- 性能分析工具

### 🛡️ 高可靠性

- 优雅关闭机制
- 健康检查
- 数据备份和恢复
- 故障转移支持

## 技术栈总览

### 核心框架

- **Go 1.x**: 主要编程语言
- **Gin**: HTTP Web 框架
- **GORM**: ORM 框架

### 数据存储

- **关系型**: MySQL, PostgreSQL
- **缓存**: Redis
- **文档**: MongoDB
- **分析**: ClickHouse, TDengine
- **搜索**: Elasticsearch, ManticoreSearch

### 消息队列

- **Kafka**: 高吞吐量分布式消息队列
- **NSQ**: 轻量级实时消息队列

### 监控运维

- **Prometheus**: 监控指标收集
- **Grafana**: 可视化面板
- **ELK Stack**: 日志分析

### 部署工具

- **Docker**: 容器化
- **Kubernetes**: 容器编排
- **CI/CD**: 自动化部署

## 快速开始

### 本地开发

```bash
# 克隆项目
git clone <repository-url>
cd go-gin-project

# 安装依赖
go mod download

# 启动开发环境
make dev

# 或使用Docker
docker-compose up -d
```

### 生产部署

```bash
# 构建镜像
make build

# 部署到Kubernetes
kubectl apply -f k8s/
```

## 配置说明

### 主要配置文件

- `conf/server.toml`: 服务器配置
- `conf/service/`: 各种外部服务配置
- `conf/log/`: 日志配置

### 环境变量

支持通过环境变量覆盖配置文件设置：

- `GIN_MODE`: Gin运行模式
- `LOG_LEVEL`: 日志级别
- `DB_HOST`: 数据库主机
- `REDIS_ADDR`: Redis地址

## 监控和运维

### 监控端点

- `http://localhost:8081/metrics`: Prometheus指标
- `http://localhost:8081/debug/pprof/`: 性能分析

### 日志查看

```bash
# 查看应用日志
tail -f log/info.log

# 查看错误日志
tail -f log/error.log

# 使用Docker查看日志
docker logs -f <container-id>
```

### 性能分析

```bash
# CPU分析
go tool pprof http://localhost:8081/debug/pprof/profile

# 内存分析
go tool pprof http://localhost:8081/debug/pprof/heap

# Goroutine分析
go tool pprof http://localhost:8081/debug/pprof/goroutine
```

## 开发指南

### 代码结构

- `main.go`: 应用入口
- `bootstrap/`: 初始化逻辑
- `servers/`: 服务器配置
- `library/`: 公共库
- `pkg/`: 第三方包装
- `model/`: 数据模型

### 开发规范

- 遵循Go代码规范
- 完整的错误处理
- 详细的代码注释
- 单元测试覆盖

### 扩展开发

- 添加新的中间件
- 集成新的数据库
- 扩展监控指标
- 自定义业务逻辑

## 故障排查

### 常见问题

1. **服务启动失败**: 检查配置文件和依赖服务
2. **数据库连接失败**: 验证连接参数和网络
3. **性能问题**: 使用pprof分析性能瓶颈
4. **内存泄漏**: 监控内存使用和Goroutine数量

### 调试工具

- pprof性能分析
- Prometheus监控指标
- 结构化日志分析
- 分布式链路追踪

## 贡献指南

### 开发流程

1. Fork项目
2. 创建功能分支
3. 编写代码和测试
4. 提交Pull Request

### 代码质量

- 通过所有测试
- 代码覆盖率 > 80%
- 通过静态分析
- 符合代码规范

📚 **更多文档**: 请查看 [架构说明文档](架构说明.md) 获取详细的技术架构信息。
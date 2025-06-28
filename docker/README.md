# Docker Compose 开发环境

这个 Docker Compose 配置提供了一个完整的开发环境，包含多种数据库和中间件服务。

## 🚀 快速开始

### 1. 环境准备

```bash
# 复制环境变量文件
cp .env.example .env

# 编辑环境变量文件，设置安全的密码
vim .env
```

### 2. 启动服务

```bash
# 启动所有服务
docker-compose up -d

# 启动特定服务
docker-compose up -d mysql redis

# 查看服务状态
docker-compose ps

# 查看服务日志
docker-compose logs -f mysql
```

### 3. 停止服务

```bash
# 停止所有服务
docker-compose down

# 停止并删除数据卷（谨慎使用）
docker-compose down -v
```

## 📋 服务列表

| 服务 | 端口 | 用户名 | 密码 | 说明 |
|------|------|--------|------|------|
| MySQL | 3306 | root | 123456 | 关系型数据库 |
| PostgreSQL | 5432 | postgres | 123456 | 关系型数据库 |
| MongoDB | 27017 | root | 123456 | 文档数据库 |
| Redis | 6379 | - | 123456 | 缓存数据库 |
| ClickHouse | 8123,9000 | root | 123456 | 列式数据库 |
| Elasticsearch | 9200,9300 | - | - | 搜索引擎 |
| Kibana | 5601 | - | - | ES 可视化工具 |
| Kafka | 9092 | - | - | 消息队列 |
| Zookeeper | 2181 | - | - | 协调服务 |
| etcd | 2379,2380 | - | - | 键值存储 |
| NSQ | 4150,4151,4161,4171 | - | - | 消息队列 |
| Manticore | 9306,9308 | - | - | 搜索引擎 |
| TDengine | 6041 | - | - | 时序数据库 |

## 🔧 高级配置

### 数据持久化

所有服务的数据都会持久化到 Docker 卷中，容器重启不会丢失数据。

### 资源限制

每个服务都配置了合理的内存和 CPU 限制，适合开发环境使用。

### 健康检查

所有服务都配置了健康检查，可以通过以下命令查看：

```bash
docker-compose ps
```

### 网络配置

所有服务都在同一个自定义网络中，可以通过服务名相互访问。

## 🛠️ 故障排除

### 查看日志
```bash
docker-compose logs -f [service_name]
```

### 重启服务
```bash
docker-compose restart [service_name]
```

### 清理资源
```bash
# 清理未使用的镜像
docker image prune

# 清理未使用的卷
docker volume prune
```

## ⚠️ 注意事项

1. **生产环境使用**：请修改所有默认密码
2. **资源要求**：确保系统有足够的内存（建议 8GB+）
3. **端口冲突**：确保所有端口未被占用
4. **数据备份**：重要数据请定期备份

## 📝 自定义配置

可以在对应的配置目录中添加自定义配置文件：

- `mysql/init/` - MySQL 初始化脚本
- `postgres/init/` - PostgreSQL 初始化脚本
- `mongodb/init/` - MongoDB 初始化脚本
- `redis/config/` - Redis 配置文件
- `elasticsearch/config/` - Elasticsearch 配置
- `kibana/config/` - Kibana 配置

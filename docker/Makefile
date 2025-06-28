# Docker Compose 管理工具

.PHONY: help up down restart logs ps clean setup

# 默认目标
help:
	@echo "可用命令："
	@echo "  setup     - 初始化环境（复制 .env 文件）"
	@echo "  up        - 启动所有服务"
	@echo "  down      - 停止所有服务"
	@echo "  restart   - 重启所有服务"
	@echo "  logs      - 查看所有服务日志"
	@echo "  ps        - 查看服务状态"
	@echo "  clean     - 清理未使用的资源"
	@echo ""
	@echo "单个服务操作："
	@echo "  make up-mysql     - 启动 MySQL"
	@echo "  make logs-mysql   - 查看 MySQL 日志"
	@echo "  make restart-mysql - 重启 MySQL"

# 初始化环境
setup:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "已创建 .env 文件，请编辑其中的密码配置"; \
	else \
		echo ".env 文件已存在"; \
	fi

# 启动所有服务
up:
	docker-compose up -d

# 停止所有服务
down:
	docker-compose down

# 重启所有服务
restart:
	docker-compose restart

# 查看日志
logs:
	docker-compose logs -f

# 查看服务状态
ps:
	docker-compose ps

# 清理资源
clean:
	docker system prune -f
	docker volume prune -f

# 单个服务操作
up-%:
	docker-compose up -d $*

down-%:
	docker-compose stop $*

restart-%:
	docker-compose restart $*

logs-%:
	docker-compose logs -f $*

# 数据库服务快捷方式
mysql: up-mysql
postgres: up-postgresql
mongo: up-mongodb
redis: up-redis
clickhouse: up-clickhouse
elasticsearch: up-elasticsearch

# 消息队列服务
kafka: up-zookeeper up-kafka
nsq: up-nsqlookupd up-nsqd up-nsqadmin

# 健康检查
health:
	@echo "检查服务健康状态..."
	@docker-compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"

# 备份数据
backup:
	@echo "创建数据备份..."
	@mkdir -p backups/$(shell date +%Y%m%d_%H%M%S)
	@docker-compose exec mysql mysqldump -u root -p123456 --all-databases > backups/$(shell date +%Y%m%d_%H%M%S)/mysql_backup.sql
	@echo "MySQL 备份完成"

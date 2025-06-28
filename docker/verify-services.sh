#!/bin/bash

# 服务验证脚本
# 用法: ./verify-services.sh

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_status() {
    local status=$1
    local message=$2
    case $status in
        "success")
            echo -e "${GREEN}✓${NC} $message"
            ;;
        "error")
            echo -e "${RED}✗${NC} $message"
            ;;
        "warning")
            echo -e "${YELLOW}⚠${NC} $message"
            ;;
        "info")
            echo -e "${BLUE}ℹ${NC} $message"
            ;;
    esac
}

# 验证数据库连接
verify_databases() {
    print_status "info" "验证数据库连接..."
    
    # MySQL
    if docker exec mysql mysql -u root -p123456 -e "SELECT 'MySQL连接成功' as status;" >/dev/null 2>&1; then
        print_status "success" "MySQL 连接成功"
    else
        print_status "error" "MySQL 连接失败"
    fi
    
    # PostgreSQL
    if docker exec -e PGPASSWORD=123456 postgresql psql -U postgres -d test -c "SELECT 'PostgreSQL连接成功' as status;" >/dev/null 2>&1; then
        print_status "success" "PostgreSQL 连接成功"
    else
        print_status "error" "PostgreSQL 连接失败"
    fi
    
    # MongoDB
    if docker exec mongodb mongosh --username root --password 123456 --authenticationDatabase admin --eval "db.adminCommand('ping')" >/dev/null 2>&1; then
        print_status "success" "MongoDB 连接成功"
    else
        print_status "error" "MongoDB 连接失败"
    fi
    
    # Redis
    if docker exec redis redis-cli -a 123456 ping >/dev/null 2>&1; then
        print_status "success" "Redis 连接成功"
    else
        print_status "error" "Redis 连接失败"
    fi
}

# 验证端口可访问性
verify_ports() {
    print_status "info" "验证端口可访问性..."
    
    local ports=("3306:MySQL" "5432:PostgreSQL" "27017:MongoDB" "6379:Redis" "4161:NSQ_Lookup" "4151:NSQ_Daemon" "4171:NSQ_Admin" "2379:etcd")
    
    for port_info in "${ports[@]}"; do
        IFS=':' read -r port service <<< "$port_info"
        if nc -z localhost $port 2>/dev/null; then
            print_status "success" "$service (端口 $port) 可访问"
        else
            print_status "error" "$service (端口 $port) 不可访问"
        fi
    done
}

# 验证 NSQ 服务
verify_nsq() {
    print_status "info" "验证 NSQ 服务..."

    # 检查 NSQ Lookup
    if curl -s http://localhost:4161/ping >/dev/null 2>&1; then
        print_status "success" "NSQ Lookup 服务正常"
    else
        print_status "error" "NSQ Lookup 服务异常"
    fi

    # 检查 NSQ Admin
    if curl -s http://localhost:4171/ >/dev/null 2>&1; then
        print_status "success" "NSQ Admin 界面可访问"
    else
        print_status "error" "NSQ Admin 界面不可访问"
    fi
}

# 验证 Zookeeper 服务
verify_zookeeper() {
    print_status "info" "验证 Zookeeper 服务..."

    # 检查 Zookeeper 状态
    if docker exec zookeeper zkServer.sh status >/dev/null 2>&1; then
        print_status "success" "Zookeeper 服务正常"
    else
        print_status "error" "Zookeeper 服务异常"
    fi

    # 检查 Zookeeper 连接
    if docker exec zookeeper zkCli.sh -server localhost:2181 ls / >/dev/null 2>&1; then
        print_status "success" "Zookeeper 连接正常"
    else
        print_status "error" "Zookeeper 连接失败"
    fi
}

# 验证 Kafka 服务
verify_kafka() {
    print_status "info" "验证 Kafka 服务..."

    # 检查 Kafka 容器是否运行
    if docker ps --format "{{.Names}}" | grep -q "^kafka$"; then
        print_status "success" "Kafka 容器正在运行"

        # 检查 Kafka 端口
        if nc -z localhost 9092 2>/dev/null; then
            print_status "success" "Kafka 端口 9092 可访问"
        else
            print_status "error" "Kafka 端口 9092 不可访问"
        fi

        # 尝试检查 Kafka broker
        if docker exec kafka kafka-broker-api-versions.sh --bootstrap-server localhost:9092 >/dev/null 2>&1; then
            print_status "success" "Kafka broker 响应正常"
        else
            print_status "warning" "Kafka broker 可能还在启动中"
        fi
    else
        print_status "error" "Kafka 容器未运行"
    fi
}

# 显示服务状态
show_service_status() {
    print_status "info" "当前服务状态:"
    docker-compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"
}

# 显示资源使用情况
show_resource_usage() {
    print_status "info" "资源使用情况:"
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"
}

# 主函数
main() {
    echo "=== Docker Compose 服务验证 ==="
    echo "验证时间: $(date)"
    echo

    verify_databases
    echo

    verify_ports
    echo

    verify_nsq
    echo

    verify_zookeeper
    echo

    verify_kafka
    echo

    show_service_status
    echo

    show_resource_usage

    echo
    print_status "info" "验证完成！"
}

# 检查依赖
if ! command -v docker &> /dev/null; then
    print_status "error" "Docker 未安装或不在 PATH 中"
    exit 1
fi

if ! command -v nc &> /dev/null; then
    print_status "warning" "netcat (nc) 未安装，跳过端口检查"
fi

if ! command -v curl &> /dev/null; then
    print_status "warning" "curl 未安装，跳过 HTTP 检查"
fi

# 运行主函数
main

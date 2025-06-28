#!/bin/bash

# 服务监控脚本
# 用法: ./monitor.sh [service_name]

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
        "healthy")
            echo -e "${GREEN}✓${NC} $message"
            ;;
        "unhealthy")
            echo -e "${RED}✗${NC} $message"
            ;;
        "starting")
            echo -e "${YELLOW}⚠${NC} $message"
            ;;
        "info")
            echo -e "${BLUE}ℹ${NC} $message"
            ;;
    esac
}

# 检查单个服务
check_service() {
    local service=$1
    local container_name=$2
    local port=$3
    
    # 检查容器是否运行
    if docker ps --format "table {{.Names}}" | grep -q "^${container_name}$"; then
        # 检查健康状态
        health_status=$(docker inspect --format='{{.State.Health.Status}}' $container_name 2>/dev/null || echo "no-health-check")
        
        case $health_status in
            "healthy")
                print_status "healthy" "$service ($container_name) - 运行正常"
                ;;
            "unhealthy")
                print_status "unhealthy" "$service ($container_name) - 健康检查失败"
                ;;
            "starting")
                print_status "starting" "$service ($container_name) - 正在启动"
                ;;
            "no-health-check")
                print_status "info" "$service ($container_name) - 运行中（无健康检查）"
                ;;
        esac
        
        # 检查端口
        if [ ! -z "$port" ]; then
            if nc -z localhost $port 2>/dev/null; then
                print_status "healthy" "  端口 $port 可访问"
            else
                print_status "unhealthy" "  端口 $port 不可访问"
            fi
        fi
    else
        print_status "unhealthy" "$service ($container_name) - 未运行"
    fi
    echo
}

# 主函数
main() {
    echo "=== Docker Compose 服务监控 ==="
    echo "检查时间: $(date)"
    echo

    # 如果指定了服务名，只检查该服务
    if [ ! -z "$1" ]; then
        case $1 in
            "mysql")
                check_service "MySQL" "mysql" "3306"
                ;;
            "postgresql")
                check_service "PostgreSQL" "postgresql" "5432"
                ;;
            "mongodb")
                check_service "MongoDB" "mongodb" "27017"
                ;;
            "redis")
                check_service "Redis" "redis" "6379"
                ;;
            "clickhouse")
                check_service "ClickHouse" "clickhouse" "8123"
                ;;
            "elasticsearch")
                check_service "Elasticsearch" "elasticsearch" "9200"
                ;;
            "kibana")
                check_service "Kibana" "kibana" "5601"
                ;;
            *)
                echo "未知服务: $1"
                echo "支持的服务: mysql, postgresql, mongodb, redis, clickhouse, elasticsearch, kibana"
                exit 1
                ;;
        esac
        return
    fi

    # 检查所有服务
    print_status "info" "检查数据库服务..."
    check_service "MySQL" "mysql" "3306"
    check_service "PostgreSQL" "postgresql" "5432"
    check_service "MongoDB" "mongodb" "27017"
    check_service "Redis" "redis" "6379"
    check_service "ClickHouse" "clickhouse" "8123"
    
    print_status "info" "检查搜索引擎..."
    check_service "Elasticsearch" "elasticsearch" "9200"
    check_service "Kibana" "kibana" "5601"
    check_service "Manticore" "manticore" "9306"
    
    print_status "info" "检查消息队列..."
    check_service "Kafka" "kafka" "9092"
    check_service "Zookeeper" "zookeeper" "2181"
    check_service "NSQ Lookup" "nsqlookupd" "4161"
    check_service "NSQ Daemon" "nsqd" "4151"
    check_service "NSQ Admin" "nsqadmin" "4171"
    
    print_status "info" "检查其他服务..."
    check_service "etcd" "etcd" "2379"
    check_service "TDengine" "tdengine" "6041"
    
    # 显示资源使用情况
    echo "=== 资源使用情况 ==="
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"
}

# 检查依赖
if ! command -v docker &> /dev/null; then
    print_status "unhealthy" "Docker 未安装或不在 PATH 中"
    exit 1
fi

if ! command -v nc &> /dev/null; then
    print_status "unhealthy" "netcat (nc) 未安装，无法检查端口"
fi

# 运行主函数
main "$@"

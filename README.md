# go-gin-project
基于 Gin 进行模块化设计的 API 框架，封装了常用的功能，包括消息队列，数据库等，使用简单，致力于进行快速的业务研发。

仅供参考学习，线上请谨慎使用！！！

### 集成组件：
1. 支持 rate 接口限流
2. 支持 JWT 鉴权管理
3. 支持 Casbin 鉴权管理
4. 支持 zap 日志收集
5. 支持 toml 配置文件解析
6. 支持 gorm 数据库组件
7. 支持 go-redis 组件
8. 支持 MySQL，Postgresql 关系型数据库
9. 支持 ElasticSearch 非关系型数据库
10. 支持 Redis，Etcd 缓存数据库
11. 支持 Kafka，NSQ 消息队列
12. 支持 RESTful API 返回值规范

### 待添加完善组件：
1. 支持 TDengine 时序数据库
2. 支持 MongoDB 非关系型数据库
3. 支持 Clickhouse 列式数据库
4. 支持 Prometheus 指标记录
5. 支持 pprof 性能剖析

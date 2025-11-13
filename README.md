# Review Service

一个基于 Go 与 Kratos 的评审（Review）服务，提供 HTTP/gRPC API，集成 MySQL、Redis、Kafka、Elasticsearch，并配套容器化与本地开发环境。

## 特性
- HTTP/gRPC 双栈服务，自动路由与 OpenAPI 文档
- 分层架构：`biz`（领域）、`data`（资源适配）、`service`（接口）、`server`（装配）、`conf`（配置）
- 持久化与缓存：MySQL、Redis
- 消息与检索：Kafka、Elasticsearch
- 容器与依赖编排：Docker/Docker Compose

## 目录结构
- `api/` Proto 定义与生成的 HTTP/GRPC 绑定
- `cmd/` 服务入口：`review-service`（主服务）、`review-task`（后台任务）
- `configs/` 配置文件（默认 `config.yaml`）
- `internal/` 分层代码（`biz`/`data`/`service`/`server`/`conf`）
- `third_party/` 依赖的 proto（google/openapi/validate 等）
- `Dockerfile`、`Makefile`、`openapi.yaml` 等

## 快速开始

### 环境要求
- Go 1.23+
- Docker 与 Docker Compose（可选，用于本地依赖）

### 方式一：本地运行
1. 生成代码与依赖注入：
   ```bash
   make generate
   ```
2. 构建二进制：
   ```bash
   make build
   ```
3. 启动服务：
   ```bash
   ./bin/server -conf=./configs
   ```

### 方式二：容器与依赖编排
1. 启动依赖（按需选择）：
   ```bash
   docker compose -f ../docker-compose.mysql.yml up -d
   docker compose -f ../docker-compose.redis.yml up -d
   docker compose -f ../docker-compose.kafka.yml up -d
   docker compose -f ../docker-compose.es.yml up -d
   ```
2. 构建与运行服务容器：
   ```bash
   docker build -t review-service:local .
   docker run --rm -p 8000:8000 -p 9000:9000 \
     -v %cd%\configs:/data/conf \
     --name review-service review-service:local \
     ./server -conf /data/conf
   ```

## 配置说明
- 配置文件：`configs/config.yaml`
- 主要项（示例，实际以 `internal/conf/conf.proto` 为准）：
  - `server.http`：端口、超时
  - `server.grpc`：端口、超时
  - `data.database`：MySQL 连接（避免提交明文凭据）
  - `data.redis`：Redis 连接
  - `data.kafka`：Kafka broker、topic
  - `data.elasticsearch`：ES 地址与索引

## API 概览
- 资源：`Review`
  - `POST /v1/reviews` 创建评审
  - `PUT /v1/reviews/{id}` 更新评审
  - `DELETE /v1/reviews/{id}` 删除评审
  - `GET /v1/reviews/{id}` 查询详情
  - `GET /v1/reviews` 分页列表
- OpenAPI：`openapi.yaml`

## 构建与开发
- 代码生成：
  ```bash
  make generate  # 包含 protoc 与 wire 生成
  ```
- 常用任务：
  ```bash
  make build     # 构建到 bin/
  make all       # 依赖安装、生成、构建一体化
  ```
- 入口程序：
  - `cmd/review-service/main.go` 主服务
  - `cmd/review-task/main.go` 后台任务（如 Kafka 消费与 ES 同步）

## 测试与质量（建议）
- 补充 `internal/biz`、`internal/data` 单元测试与 Review API 集成测试
- 在 CI 中配置覆盖率阈值

## 可观测性（建议）
- 集成 Prometheus 指标与 OpenTelemetry Trace 导出
- 统一错误响应与结构化日志字段（trace_id/span_id）

## 安全与配置（建议）
- 引入鉴权（JWT/OAuth2）与输入校验（`protoc-validate` 或结构体校验）
- 将敏感信息迁移到环境变量或 Secret 管理
- 关闭依赖服务的默认不安全配置（例如 ES 安全）

## 部署
- 构建镜像：
  ```bash
  docker build -t a1den777/review-service:latest .
  ```
- CI/CD：建议使用 GitHub Actions 执行 `lint/test/build/push` 并统一 Go 版本至 1.23

## 许可
- 见 `LICENSE`

## 致谢
- 基于 [Kratos](https://go-kratos.dev/) 框架与相关生态


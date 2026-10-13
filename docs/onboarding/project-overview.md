# 项目概览

> 适用对象：第一次接触本仓库、希望理解一条真实请求如何落到数据层的 Go 开发者。本文依据当前工作区源码；当前目录不含 Git 元数据，无法标注提交版本。

## 一句话模型

**Riff** 是一个面向社区用户的单进程 Go Web 应用：访客可注册、登录和分页浏览帖子，登录用户可查看社区、发布帖子、查看帖子详情并投票；Gin 提供 HTTP 接口与静态页面，GORM/MySQL 保存用户、社区和帖子，Redis 有序集合保存帖子的发布时间、热度分数和逐用户投票状态，JWT 承载登录身份。这个模型由 `router/router.go: SetupRouter`、`logic/*` 和 `dao/*` 的实际调用共同验证，而不仅来自 Swagger 描述。

## 用户与主要场景

- 访客：打开根页面，访问健康检查，注册、登录，浏览帖子列表。
- 已登录用户：携带 `Authorization: Bearer <token>` 查看社区与帖子详情、发布帖子和投票。
- 运维/开发者：通过 `/ping` 检查进程，通过 `/swagger/*any` 查看当前生成的接口文档；非 release 模式还可通过 `/debug/pprof/*` 分析运行状态。

路由事实以 `router/router.go: SetupRouter` 为准。`GET /api/v1/posts` 与增强版 `GET /api/v1/posts2` 都在认证中间件注册前，因此是公开接口；Swagger 已与这一认证边界对齐。

## 已实现能力

| 能力 | HTTP 入口 | 主要实现 |
| --- | --- | --- |
| 注册 | `POST /api/v1/signup` | `controller/user.go: SignUpHandler` → `logic/user.go: SignUp` → `dao/mysql/user.go` |
| 登录 | `POST /api/v1/login` | `controller/user.go: LoginHandler` → `logic/user.go: Login` → MySQL → `pkg/jwt/jwt.go: GenToken` |
| 基础帖子列表 | `GET /api/v1/posts` | `controller/post.go: PostList` → `logic/post.go: PostList` → MySQL |
| 增强帖子列表 | `GET /api/v1/posts2` | `GetPostListHandler2` → `logic.GetPostList2` → Redis 排行 + MySQL 批量补齐 + Redis 净票数 |
| 社区查询 | `GET /api/v1/community[/:id]` | `controller/community.go` → `logic/community.go` → `dao/mysql/community.go` |
| 发帖 | `POST /api/v1/post` | JWT → `controller/post.go: CreatePost` → MySQL → Redis |
| 帖子详情 | `GET /api/v1/post/:id` | `controller/post.go: GetPostByID` → MySQL |
| 投票 | `POST /api/v1/vote` | JWT → `controller/vote.go: VoteHandler` → `logic/vote.go` → `dao/redis/vote.go` |

## 系统边界与依赖

- 进程内：配置加载、日志、HTTP 路由、中间件、控制器、业务逻辑、DAO、JWT 与 Snowflake ID。
- 持久化边界：MySQL（`users`、`communities`、`posts`，由 `dao/mysql/mysql.go: InitMySQL` 自动迁移）和 Redis。
- 前端边界：`templates/index.html` 加载 `static/` 中已经构建好的前端产物；前端源码不在本仓库中。
- 运行时依赖：Go 1.25.4（`go.mod`）、MySQL、Redis；镜像构建见 `Dockerfile`。
- 未发现消息队列、后台任务或定时任务。关于“投票到期后回写 MySQL 并删除 Redis key”的内容只出现在 `logic/vote.go` 和 `dao/redis/vote.go` 的注释中，没有实现入口。

## 五分钟架构

```text
HTTP / JSON / 静态资源
          |
      Gin Router
          |
 日志/恢复 -> JWT（部分 /api/v1 路由）
          |
      controller        参数绑定、上下文身份、统一响应
          |
        logic           ID 生成、用例编排、投票规则入口
       /     \
 MySQL DAO   Redis DAO  用户/社区/帖子 | 时间/分数/投票
```

典型请求的控制流是 `router → middleware → controller → logic → dao → controller.Response*`。结构体既承担 GORM 模型又承担部分 API DTO，集中在 `models/`。

## 可安全验证的最小体验

在不准备数据库时，只做编译级验证：

```powershell
go test ./...
go vet ./...
```

当前快照两项均通过；`controller`、`dao/mysql`、`dao/redis` 和 `pkg/jwt` 已有针对分页边界、密码兼容升级、投票输入和 token 往返的单元测试。数据库与 Redis 的跨存储行为仍应通过容器级 smoke test 验证。

准备好 MySQL、Redis 和安全的本地配置后，可启动进程并先访问：

```text
GET http://127.0.0.1:8888/ping
预期：{"message":"pong"}
```

直接本地启动前必须设置至少 32 字节的 `AUTH_JWT_SECRET`。`docker compose up --build` 会使用仅供本地开发的容器密钥、等待 MySQL/Redis 健康后启动应用，并把服务暴露在 `127.0.0.1:8887`。

## 已知缺口与易误导点

以下均为当前源码可直接观察到的事实：

- 仓库没有根 README、贡献指南或版本化迁移脚本；数据表靠 `AutoMigrate`，社区种子数据来源未提供。
- `logic/post.go: CreatePost` 仍是 MySQL 后 Redis 的跨存储写入，但 Redis 失败时会调用 `mysql.DeletePost` 补偿。若补偿本身失败，错误会通过 `errors.Join` 保留，仍需运维介入。
- `dao/redis/vote.go: VoteForPost` 已改为 Lua 原子状态转换并修复取消投票；个人投票 key 会在该帖投票窗口结束时过期。注释描述的票数汇总回 MySQL 仍未实现。
- `/posts2` 已接入 `page`、`size`、`community_id` 和 `order=time|score`；基础 `/posts` 仍保留为纯 MySQL 兼容接口。
- 新密码使用 bcrypt；旧版密码在成功登录时兼容校验并自动升级。JWT 密钥必须从配置/环境注入并至少 32 字节，解析仅接受 HS256。`conf/config.yaml` 中的数据库口令仍是本地开发值，不应照搬到真实环境。
- Swagger 已覆盖当前九个业务路由并区分公开/受保护接口；后续改路由时仍需重新执行 `swag init`，避免再次漂移。
- 自动化测试尚未覆盖真实 MySQL/Redis、Lua 并发和补偿失败；本轮使用隔离 Compose 环境完成了成功链、撤销投票和 Redis 故障补偿 smoke test。
- 当前 Windows 环境缺少 C 编译器，因此 `go test -race ./...` 无法执行；上线前应在支持 CGO 的 CI/Linux 环境补跑竞态检测。
- `riff_log` 是运行时产物，不是入门必读文件；`.gitignore` 已排除日志。恢复日志会去掉查询字符串并脱敏 Authorization、Cookie 等敏感头。

## 建议阅读顺序

1. 本文：建立产品和边界模型。
2. [架构说明](architecture.md)：理解组件责任、依赖方向和状态归属。
3. [核心调用链](core-call-chains.md)：沿启动、登录、发帖和投票纵向追踪。
4. [学习路线](learning-roadmap.md)：按依赖顺序练习并完成可验证检查点。

# 学习路线

## 总原则

按“外部行为 → 系统边界 → 可运行基线 → 身份与数据基础 → 纵向主链 → 状态机与失败 → 运维质量”学习，而不是逐目录扫文件。每一阶段都以能解释、预测或验证某个行为为完成标准。

## 材料分级

### 必读

1. `docs/onboarding/project-overview.md`
2. `router/router.go: SetupRouter`
3. `main.go: main`
4. `models/post.go`、`models/user.go`、`models/community.go`、`models/params.go`
5. `middlerwares/auth.go: JWTAuthMiddleware` 与 `controller/request.go: getCurrentUser`
6. 发帖纵向链：`controller/post.go: CreatePost` → `logic/post.go: CreatePost` → `dao/mysql/post.go: CreatePost` / `dao/redis/vote.go: CreatePost`
7. 投票纵向链：`controller/vote.go: VoteHandler` → `logic/vote.go: VoteForPost` → `dao/redis/vote.go: VoteForPost`
8. `docs/onboarding/core-call-chains.md`

### 按需参考

- `conf/config.go`、`setting/setting.go`：排查配置来源与热更新。
- `logger/logger.go`：排查日志、panic 和请求生命周期。
- `controller/response.go`、`controller/code.go`、`controller/validator.go`：理解 API 约定。
- `dao/mysql/mysql.go`、`dao/redis/redis.go`：连接初始化与关闭。
- `docs/swagger.yaml`：当前九个业务路由的生成契约；仍以 `router.SetupRouter` 验证实际注册顺序。
- `Dockerfile`、`docker-compose.yml`：学习镜像边界、健康检查和环境变量覆盖。

### 首轮可跳过

- `static/` 下的压缩 JS/CSS、source map 和图片：它们是前端构建产物，前端源码缺失。
- `docs/docs.go`：Swagger 生成代码；优先读注释源和 YAML。
- `controller/doc_response_models.go`：只服务 Swagger 的响应类型。
- `riff_log`：运行时日志，不应作为常规学习材料；请求转储已有敏感信息脱敏保护。
- `wait-for.sh`：当前 Compose 已改用服务健康条件，此脚本未被调用，只作为历史辅助文件保留。

## 阶段 1：建立产品与路由模型

| 字段 | 内容 |
| --- | --- |
| 目标 | 说清谁在用、有哪些外部行为、哪些路由需要登录 |
| 阅读 | `project-overview.md` → `router/router.go: SetupRouter` → `controller/code.go` |
| 追踪 | 从每个 `/api/v1` 路由只追到对应 handler，暂不深入实现 |
| 练习 | 从源码手写“公开/需 JWT”两列路由表，再与架构说明核对 |
| 检查点 | 不看笔记解释为何 `/posts` 是公开的、`/post/:id` 却受保护，并说明响应 envelope |
| 可选 | 浏览 `templates/index.html`，确认根页面只是加载已构建前端 |

## 阶段 2：掌握启动与系统边界

| 字段 | 内容 |
| --- | --- |
| 目标 | 能从内存画出进程、MySQL、Redis、客户端四个边界 |
| 阅读 | `architecture.md` → `main.go: main` → `setting/setting.go: Init` → `dao/mysql/mysql.go: InitMySQL` → `dao/redis/redis.go: InitRedis` |
| 追踪 | 启动初始化顺序和收到 SIGTERM 后的关闭路径 |
| 练习 | 预测 MySQL 不可用、Redis 不可用、start_time 格式错误时进程分别停在哪一步 |
| 检查点 | 画出组件与依赖箭头，并指出配置热更新不会重建哪些资源 |
| 可选 | `logger/logger.go`、`pkg/snowflake/snowflake.go` |

## 阶段 3：获得可重复的黑盒基线

| 字段 | 内容 |
| --- | --- |
| 目标 | 区分“能编译”和“业务可运行” |
| 阅读 | `go.mod` → `Dockerfile` → `conf/config.yaml` → `docker-compose.yml` |
| 追踪 | Docker 构建产物如何进入运行镜像；应用究竟从哪里读取配置 |
| 练习 | 运行 `go test ./...`、`go vet ./...` 和 `docker compose config --quiet`；再以 `docker compose up --build` 验证 `/ping`。在带 C 编译器的环境补跑 `go test -race ./...` |
| 检查点 | 能解释本地配置为何要求 `AUTH_JWT_SECRET`，以及 Compose 如何等待 MySQL/Redis 健康后启动应用 |
| 可选 | 为 Compose 增加社区种子数据，但不要把生产秘密写入文件 |

## 阶段 4：理解身份和核心数据形状

| 字段 | 内容 |
| --- | --- |
| 目标 | 理解 user/post/community 的标识、关系和 JWT 身份传播 |
| 阅读 | `models/*.go` → `logic/user.go` → `dao/mysql/user.go` → `pkg/jwt/jwt.go` → `middlerwares/auth.go` → `controller/request.go` |
| 追踪 | 注册生成 user ID；登录生成 token；下一次请求把 claims.UserID 写入 Gin context |
| 练习 | 写出登录成功 token 中的自定义 claims 和有效期；解释 bcrypt 新密码与旧摘要自动升级路径 |
| 检查点 | 不看代码复述“用户名密码如何变成 CreatePost 可用的 AuthorID” |
| 可选 | 校验翻译器的中英文注册逻辑 |

## 阶段 5：纵向追踪发布帖子主链

| 字段 | 内容 |
| --- | --- |
| 目标 | 掌握最具代表性的跨层、跨数据源用例 |
| 阅读 | `core-call-chains.md` 的发布帖子部分 → `controller/post.go: CreatePost` → `logic/post.go: CreatePost` → 两个 DAO 的 `CreatePost` |
| 追踪 | Bearer token → AuthorID/PostID → MySQL INSERT → Redis 两次 ZADD → 统一响应 |
| 练习 | 画出 Redis 失败、MySQL 补偿成功/失败两种状态；比较当前补偿与 outbox 方案的取舍 |
| 检查点 | 用精确文件和符号复述每一跳的输入、输出与副作用 |
| 可选 | 比较“数据库事务 + outbox”与“只重试 Redis”的失败语义 |

## 阶段 6：学习投票状态机

| 字段 | 内容 |
| --- | --- |
| 目标 | 能预测任意旧票/新票组合的 Redis 变化与终止条件 |
| 阅读 | `models.ParamVote` → `logic/vote.go: VoteForPost` → `dao/redis/keys.go` → `dao/redis/vote.go: VoteForPost` |
| 追踪 | 输入校验 → Lua 时间窗口 → 旧票 → `(new-old)*432` → 分数榜/个人投票榜/TTL |
| 练习 | 手算六种状态转换；设计并发投票和 `1 → 0` 撤销的 Redis 集成测试 |
| 检查点 | 解释 Lua 为何消除了“分数已变但投票记录未变”的竞争窗口，以及个人 key 何时过期 |
| 可选 | 设计到期归档 worker，但明确它当前不存在 |

## 阶段 7：补齐查询、错误与可观测性

| 字段 | 内容 |
| --- | --- |
| 目标 | 识别 API 契约、实现、日志与失败处理之间的差异 |
| 阅读 | `controller/post.go: PostList/GetPostByID` → `dao/mysql/post.go` → `controller/response.go` → `logger/logger.go` → `docs/swagger.yaml` |
| 追踪 | 列表分页默认值和固定排序；详情返回形状；panic 的恢复与日志路径 |
| 练习 | 对比 `/posts` 与 `/posts2`，验证增强接口的社区过滤、时间排序、热度排序和页大小上限 |
| 检查点 | 能解释业务错误为什么常是 HTTP 200，以及监控会因此漏掉什么 |
| 可选 | 验证 release 模式不注册 pprof，并分析限流为何尚未启用 |

## 阶段 8：形成维护计划

| 字段 | 内容 |
| --- | --- |
| 目标 | 把理解转化为有顺序、可验证的改进项 |
| 阅读 | 四份 onboarding 文档中的“缺口/待确认” → 对应源码 |
| 追踪 | 为每个缺口追到一个具体拥有者符号，而不是停在目录层面 |
| 练习 | 按“数据正确性 → 认证/秘密 → 可运行环境 → 错误契约 → 文档/测试”给改进排序 |
| 检查点 | 提交一页维护提案：每项包含证据路径、风险、最小验收测试和不做的后果 |
| 可选 | 评估是否引入显式迁移、repository 接口和依赖注入；小项目未必都需要 |

## 推荐代码阅读顺序（精简版）

```text
router/router.go
  -> main.go
  -> models/*.go
  -> middlerwares/auth.go + controller/request.go
  -> controller/post.go:CreatePost
  -> logic/post.go:CreatePost
  -> dao/mysql/post.go:CreatePost
  -> dao/redis/vote.go:CreatePost
  -> controller/vote.go
  -> logic/vote.go
  -> dao/redis/vote.go:VoteForPost
  -> 查询链、响应码、日志、部署文件
```

这个顺序先建立外部契约和状态模型，再进入最重要的垂直切片，最后处理横切与运维问题；它与实际依赖关系一致，而不是文件夹顺序。

## 完成学习后的自测

如果能在不打开源码的情况下完成以下内容，就已具备安全修改该项目的基础：

1. 画出一条受保护请求从路由到 MySQL/Redis 再返回的完整链。
2. 说出三个 Redis key 的 member/score 含义。
3. 预测发帖第二阶段失败时补偿成功/失败的最终数据状态，以及六种投票转换。
4. 解释 Swagger 生成、Compose 健康依赖、JWT 环境配置与配置热更新的实际行为。
5. 给出最先应补的测试，并说明每个测试保护哪条业务不变量。

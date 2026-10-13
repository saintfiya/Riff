# Threadnest（Riff）功能式面试高频题库

> 按功能学习：先讲清请求入口和数据流，再回答实现细节、失败场景与取舍。答案依据当前仓库源码；“可延伸”问题用于面试追问，不代表项目已经实现该方案。

## 怎么用这份题库

每一组对应一个可独立讲述的功能。建议先遮住答案，按“入口 → 参数/身份 → 业务规则 → 存储副作用 → 响应”口述，再对照文件和符号查漏补缺。项目是 Go + Gin 的分层单体，核心依赖为 MySQL、Redis、JWT 和 Snowflake。

## 1. 项目架构与请求入口

### Q1：介绍一下这个项目的架构，以及一次典型请求经过哪些层？

**答：** 这是一个 Go 社区讨论应用，采用分层单体结构。Gin 路由收到 HTTP 请求后，先执行日志/恢复等中间件，受保护路由还会经过 JWT 中间件；然后进入 controller 做绑定、校验和响应，logic 编排用例，DAO 访问 MySQL 或 Redis。MySQL 存用户、社区、帖子实体；Redis 存帖子时间榜、热度榜和逐用户投票状态。主要装配点在 `router.SetupRouter`，启动入口在 `main`。

**代码定位：** `main.go:main`、`router/router.go:SetupRouter`、`controller/`、`logic/`、`dao/`。

### Q2：哪些接口公开，哪些要求登录？依据是什么？

**答：** 注册、登录、`GET /posts`、`GET /posts2` 在 `v1.Use(JWTAuthMiddleware())` 之前注册，所以公开；社区查询、发帖、帖子详情、投票在中间件之后注册，所以受 JWT 保护。判断要看 Gin 路由的注册顺序，而不只看 Swagger 注释。

**代码定位：** `router/router.go:SetupRouter`。

### Q3：项目统一响应格式是什么？业务失败一定返回非 200 HTTP 状态吗？

**答：** `ResponseSuccess` 和 `ResponseError` 都返回 `{code,msg,data}`，而业务错误通常仍使用 HTTP 200；绑定失败有些 handler 返回 400，panic 恢复路径返回 500。因此客户端需要解析业务 `code`，监控也不能只依赖 HTTP 状态码。

**代码定位：** `controller/response.go`、`controller/code.go`、各 handler。

## 2. 注册与密码存储

### Q4：用户注册从请求到数据库经历什么？

**答：** `SignUpHandler` 用 `ShouldBindJSON` 绑定并按 `ParamSignUp` 的标签校验用户名、密码和重复密码；`logic.SignUp` 生成 Snowflake `UserID` 并组装 `User`；`mysql.InsertUser` 对新密码做 bcrypt，再插入 MySQL。MySQL 唯一约束冲突被映射为 `ErrorUserExist`，controller 再映射成用户已存在的业务码。

**代码定位：** `models/params.go:ParamSignUp`、`controller/user.go:SignUpHandler`、`logic/user.go:SignUp`、`dao/mysql/user.go:InsertUser`。

### Q5：密码为什么不能明文保存？当前用什么算法？

**答：** 明文数据库泄露会直接暴露用户凭据。当前新密码用 bcrypt 的 `GenerateFromPassword`，验证时用 `CompareHashAndPassword`；bcrypt 自带盐并可调计算成本，适合密码哈希。代码还保留了旧版 MD5 派生格式，仅用于兼容老用户登录并迁移，不能把它当成新密码方案。

**可延伸：** 讨论密码哈希和普通快速哈希的区别、成本参数、密码重置和泄露后的处置。

**代码定位：** `dao/mysql/user.go:hashPassword`、`verifyPassword`。

### Q6：用户名唯一性只靠应用层检查够不够？

**答：** 不够。并发请求可能同时通过“先查后写”的应用检查；数据库唯一索引才是最终并发约束。当前 `users.username` 有唯一索引，插入冲突时将 MySQL 1062 映射到 `ErrorUserExist`。

**代码定位：** `models/user.go:User`、`dao/mysql/user.go:InsertUser`。

## 3. 登录与 JWT 认证

### Q7：登录成功后 token 是怎么产生并传到后续请求的？

**答：** handler 绑定用户名和密码，logic 调 MySQL DAO 查询用户并验证密码；成功后调用 `jwt.GenToken`，把用户 ID、用户名和标准 claims 签成 HS256 token，过期时间为 7 天。客户端后续在 `Authorization: Bearer <token>` 发送它。

**代码定位：** `controller/user.go:LoginHandler`、`logic/user.go:Login`、`dao/mysql/user.go:Login`、`pkg/jwt/jwt.go:GenToken`。

### Q8：JWT 中间件如何认证请求，身份如何交给 handler？

**答：** `JWTAuthMiddleware` 读取 `Authorization`，要求格式为 `Bearer token`，调用 `jwt.ParseToken`；解析通过后，把 claims 中的 `UserID` 写进 Gin context 的 `userID` 键，再 `Next()`。handler 通过 `getCurrentUser` 取回并断言为 `int64`。缺头或 token 无效时写业务错误并 `Abort()`，后续 handler 不执行。

**代码定位：** `middlerwares/auth.go`、`controller/request.go:getCurrentUser`、`pkg/jwt/jwt.go:ParseToken`。

### Q9：为什么启动时检查 JWT 密钥长度，解析时限制签名算法？

**答：** `jwt.Init` 拒绝短于 32 字节的密钥，减少弱密钥风险；`ParseWithClaims` 使用 `WithValidMethods` 只接受 HS256，避免接受意外算法。密钥由配置注入，未初始化时签发和解析都会报错。

**代码定位：** `pkg/jwt/jwt.go:Init`、`signingSecret`、`ParseToken`；`main.go:main`。

### Q10：JWT 的优点和限制是什么？当前项目如何处理注销或密钥轮换？

**答：** JWT 验签不需要每次查询会话库，适合无状态认证；但已签发 token 通常在过期前仍有效，单纯改用户状态不会自动撤销它。当前代码设置 7 天过期，没有 token 黑名单、refresh token 或密钥版本轮换流程。若需要即时注销，可引入短期 access token 加 refresh token、版本号或 Redis 撤销列表，并权衡存储和性能。

## 4. 发帖与 MySQL/Redis 一致性

### Q11：完整描述发帖的调用链和数据写入顺序。

**答：** `POST /api/v1/post` 先经过 JWT；controller 绑定标题、正文、社区 ID，从 context 取用户 ID 覆盖请求中的 `AuthorID`；logic 生成 Snowflake `PostID`，先写 MySQL，再调用 Redis 把帖子加入时间榜和分数榜。成功后 controller 返回统一成功响应。

**代码定位：** `router/router.go`、`controller/post.go:CreatePost`、`logic/post.go:CreatePost`、`dao/mysql/post.go:CreatePost`、`dao/redis/vote.go:CreatePost`。

### Q12：为什么作者 ID 不能信任客户端传来的值？

**答：** 客户端可以篡改 JSON 字段来冒充他人。controller 从已验签 token 对应的 Gin context 读取用户 ID，并覆盖 `p.AuthorID`，让身份来源是认证结果而不是请求体。

### Q13：发帖写 MySQL 成功、写 Redis 失败会怎样？这是不是分布式事务？

**答：** 当前 logic 尝试补偿：Redis 写入失败时调用 `mysql.DeletePost` 删除刚插入的帖子；若补偿也失败，用 `errors.Join` 同时保留两个错误。它不是跨 MySQL/Redis 的原子事务，进程崩溃、网络超时或补偿失败仍可能留下不一致状态。更强方案可用 MySQL outbox + 后台投递/重试，或持久化状态机并保证幂等。

**代码定位：** `logic/post.go:CreatePost`、`dao/mysql/post.go:DeletePost`。

### Q14：为什么生成 Snowflake ID？它和数据库自增 ID 有什么区别？

**答：** 用户和帖子对外使用 Snowflake ID，能在应用侧生成、便于分布式场景避免集中取号；表内仍有自增主键作为数据库键。Snowflake 依赖机器号和起始时间配置，机器号冲突或时钟问题会影响唯一性，所以要正确配置并监控。

**代码定位：** `pkg/snowflake/snowflake.go`、`models/user.go`、`models/post.go`、`main.go`。

## 5. 帖子列表与详情查询

### Q15：`/posts` 和 `/posts2` 有什么区别？

**答：** `/posts` 是旧版纯 MySQL 分页，按 `create_time DESC`，返回帖子基础字段；`/posts2` 支持时间或热度排序、社区过滤，并组装作者、社区和净票数。增强列表先从 Redis 取排序用的帖子 ID（有社区过滤时先从 MySQL 取候选 ID，再从 Redis 取分数排序），再批量从 MySQL 补帖子/作者/社区，最后从 Redis 算净票数。

**代码定位：** `controller/post.go:PostList`、`GetPostListHandler2`、`logic/post.go:GetPostList2`、`dao/redis/post.go`、`dao/mysql/post.go:GetPostListByIDs`。

### Q16：增强列表为什么先拿 ID 排序，再回 MySQL 查详情？

**答：** 排序分数和时间榜在 Redis，帖子正文及作者/社区关系在 MySQL。先分页拿有序 ID，再批量加载详情，可以让 Redis 管排序、MySQL 管实体数据，并减少逐帖查库造成的 N+1 查询；DAO 用 ID 到对象的 map 重建 Redis 的 ID 顺序。

### Q17：分页参数如何处理？两个接口的策略完全一样吗？

**答：** `/posts2` 初始化默认 `page=1,size=10,order=time`，query binding 限制 page 至少 1、size 在 1 到 100、order 为 `time|score`；`/posts` 由 `getPageInfo` 解析，非法或小于 1 的 page/size 回退默认值，并把 size 上限截到 100。因此旧接口是宽松回退，增强接口会对绑定失败返回参数错误。

**代码定位：** `controller/post.go`、`controller/request.go:getPageInfo`、`models/params.go:ParamPostList`。

### Q18：社区过滤 + 热度排序如何工作？有什么性能边界？

**答：** 当前先从 MySQL 取该社区所有可见帖子的 ID，再 pipeline 查询 Redis 对应的分数，在应用侧排序和分页。这样实现直接，但候选集合越大，Redis 查询和内存排序成本越高；可以考虑维护社区维度的 Redis 排行榜、限制候选窗口、或把排序能力放进专用索引系统。当前排序同分时按 ID 降序。

**代码定位：** `logic/post.go:rankAndPagePostIDs`、`dao/mysql/post.go:GetPostIDsByCommunity`、`dao/redis/post.go:GetPostOrderScores`。

### Q19：代码定义了帖子详情 DTO，详情接口真的返回了作者、社区和票数吗？

**答：** 当前 `GetPostByID` 通过 MySQL 读取后直接返回 `*models.Post`，不会组装 `ApiPostDetail`，所以不能仅凭 DTO 定义或 Swagger 预期断言接口已经返回作者名、社区和票数。应以 handler 实际调用链为准。

**代码定位：** `controller/post.go:GetPostByID`、`logic/post.go:GetPostByID`、`models/post.go:ApiPostDetail`。

## 6. 社区查询

### Q20：社区列表和详情的调用链是什么？不存在的社区如何处理？

**答：** controller 调用 logic，logic 再调用 MySQL DAO。列表返回社区集合；详情先解析 path 中的 ID，再查询 MySQL。DAO 的 not found 错误映射为 logic 层的 `ErrCommunityNotExist`，controller 将它映射为 `CodeCommunityNotExist`，其他错误归为服务繁忙。

**代码定位：** `controller/community.go`、`logic/community.go`、`dao/mysql/community.go`、`logic/errors.go`。

### Q21：发帖时有没有验证社区一定存在？

**答：** `Post` 的绑定规则要求 `community_id > 0`，但这只是格式/范围校验；是否存在需要数据库约束或显式查询。应检查 `CreatePost` 写库路径和数据库外键/约束，不能把正数校验当成存在性校验。面试中可进一步讨论在 DAO 事务内验证社区状态以避免竞态。

## 7. 投票状态机与 Redis 原子性

### Q22：投票有哪些合法值，各自代表什么？

**答：** `direction=1` 赞成，`-1` 反对，`0` 取消。HTTP 参数先由 `ParamVote` 校验，Redis DAO 也二次校验，只接受这三个值。

**代码定位：** `models/params.go:ParamVote`、`controller/vote.go`、`dao/redis/vote.go:VoteForPost`。

### Q23：`(newVote-oldVote)*432` 如何保证改票分数正确？

**答：** ZSET 中用户分数代表其当前票值：赞成是 `1`，反对是 `-1`，未投/取消是 `0`。帖子分数只增加新旧状态差：例如 `1→-1` 是 `(-1-1)*432=-864`；`-1→1` 是 `864`；`-1→0` 是 `432`。这样无需先减旧票再单独加新票两次。

**代码定位：** `dao/redis/vote.go:voteScript`。

### Q24：为什么把投票逻辑放进 Lua 脚本？

**答：** Lua 脚本在 Redis 内原子执行“确认帖子存在、检查七天窗口、读取旧票、更新总分、保存或删除个人投票状态”。如果这些步骤由多条客户端命令分别执行，并发改票可能让总分和个人状态不一致。原子脚本减少了 Redis 内部的竞争窗口，但不等于 MySQL 与 Redis 之间有分布式事务。

### Q25：投票时间窗口和 key 过期怎样实现？

**答：** 帖子创建时 Unix 时间写入 `post:time`。投票脚本用当前时间减发布时间，超过 7 天就拒绝；个人投票 ZSET 在写入新非零投票后设置剩余窗口 TTL。注释提到期满归档回写 MySQL，但仓库没有对应 worker/归档实现。

**代码定位：** `dao/redis/vote.go:voteScript`、`CreatePost`。

### Q26：取消投票如何保证把个人状态移除？

**答：** Lua 对新票 `0` 执行 `ZREM`，member 是用户 ID；同时总分按 `(0-oldVote)*432` 调整。这个路径不应只更新总分，否则重复请求或再次投票时读取到旧票会算错。

### Q27：接口返回的 `vote_num` 是什么，是否等于热度分？

**答：** `vote_num` 是赞成票数量减反对票数量，由 Redis 个人投票 ZSET 中 score 为 `1` 和 `-1` 的成员数相减；它不是用于排序的热度分。热度榜分数初始值为发帖 Unix 时间，再按投票增减 432。

**代码定位：** `dao/redis/post.go:GetPostVoteData`、`dao/redis/vote.go:voteScript`。

## 8. 配置、启动与服务关闭

### Q28：服务启动有哪些关键步骤？为什么基础依赖失败时不继续服务？

**答：** `main` 依次加载配置、初始化日志和 JWT、连接并迁移 MySQL、连接 Redis、初始化 Snowflake 与校验翻译器，再装配路由并启动 HTTP server。配置、数据库或 Redis 不可用时直接返回，属于快速失败，避免服务在依赖不完整的状态下接收请求。

**代码定位：** `main.go:main`、`setting/setting.go:Init`、`dao/mysql/mysql.go:InitMySQL`、`dao/redis/redis.go:InitRedis`。

### Q29：配置热更新是否会自动重建所有资源？

**答：** 不会。Viper 文件监听回调只重新反序列化到全局 `conf.Conf`；已有 MySQL/Redis client、logger、HTTP server 监听端口和 JWT 全局密钥不会因此自动重建或轮换。若要动态生效，需要明确实现各资源的重载和并发安全切换。

**代码定位：** `setting/setting.go:Init`、`main.go`、各基础设施初始化函数。

### Q30：优雅退出是如何实现的？5 秒超时限制什么？

**答：** `ListenAndServe` 在 goroutine 运行；主 goroutine 用 `signal.Notify` 接收 SIGINT/SIGTERM，并阻塞在 `<-quit`。收到信号后创建 5 秒 context 调 `srv.Shutdown`，等待 HTTP 请求排空；这个 timeout 约束 `Shutdown` 等待时间。随后 defer 关闭 Redis、MySQL 并同步日志。`log.Fatal` 会调用 `os.Exit`，因此如果走到它，普通 defer 不会执行。

**代码定位：** `main.go:main`。

## 9. 日志、恢复、限流与安全追问

### Q31：Gin 为什么用 `gin.New()`，项目又如何补上日志和 panic 恢复？

**答：** `gin.New()` 不自动附加 Gin 默认 logger/recovery；项目显式注册 `logger.GinLogger()` 和 `logger.GinRecovery(true)`，统一控制请求日志与 panic 恢复行为。恢复中间件应避免把敏感请求头或查询参数原样写入日志。

**代码定位：** `router/router.go:SetupRouter`、`logger/logger.go`。

### Q32：项目限流中间件已经生效了吗？

**答：** 中间件代码存在，但 `SetupRouter` 中调用它的 `r.Use(...)` 被注释，因此当前路由没有启用该限流。实现使用进程内 token bucket，单实例内生效；多实例部署时每个实例额度独立，若需要全局额度要用共享存储或网关限流。

**代码定位：** `middlerwares/ratelimit.go`、`router/router.go:SetupRouter`。

### Q33：健康检查、Swagger 和 pprof 的入口是什么？暴露 pprof 有什么注意点？

**答：** `/ping` 返回 pong；Swagger 在 `/swagger/*any`；非 release 模式注册 pprof。pprof 可暴露运行时和性能信息，开发模式端口不应无意暴露到不可信网络，生产环境应关闭或加访问控制。

**代码定位：** `router/router.go:SetupRouter`。

## 10. 综合设计题（适合面试收尾）

### Q34：如果 Redis 暂时不可用，哪些功能受影响？如何恢复？

**答：** 登录和基础 MySQL 列表等路径可能仍可用；增强列表依赖 Redis 排序/投票数据，发帖需要写 Redis，投票完全依赖 Redis。当前发帖遇到 Redis 失败会尝试删除 MySQL 行，但没有持久化重试队列。可讨论超时、熔断、降级到 MySQL 时间排序、outbox 重试和数据修复任务，并明确哪些行为允许降级。

### Q35：如何验证投票在并发下不会重复加分或丢状态？

**答：** 基于真实 Redis 做并发集成测试：同一用户同时重复投同一方向，最终净分只应体现一次状态；同时竞争相反方向，最终个人状态和帖子总分必须对应脚本序列化后的最终状态；覆盖撤票、过期和不存在帖子。当前有 DAO 单元测试，但应以真实 Redis 验证 Lua、TTL 和并发语义。

### Q36：如果要让投票结果最终持久化到 MySQL，你会如何设计？

**答：** 当前 Redis 保存可变投票状态，注释中的回写尚未实现。可以在窗口关闭时用可靠的定时/队列任务汇总并幂等写 MySQL，再标记完成；考虑任务重复执行、进程崩溃、Redis key 到期顺序、写库失败重试、对账及数据保留。另一种方案是投票明细持久化到数据库、Redis 只做缓存/榜单。先定义数据正确性与恢复目标，再选择架构。

### Q37：你会优先补哪些测试？

**答：** 优先覆盖核心不变量和错误边界：密码 bcrypt 注册/登录与旧密码迁移；JWT 缺失、格式错误、过期和错误算法；发帖 MySQL/Redis 失败及补偿失败；投票六种旧新状态组合、七天边界、撤票和并发；`/posts2` 社区过滤、排序顺序和分页。对 MySQL/Redis 交互使用隔离集成环境，纯规则则用单元测试。

## 一分钟项目介绍参考

“这是一个 Go + Gin 的社区讨论应用。请求按 router、中间件、controller、logic、DAO 分层；MySQL 保存用户、社区和帖子，Redis 保存帖子时间/热度排行榜以及用户投票状态。用户注册后用 JWT 认证，发帖会先落 MySQL 再写 Redis，Redis 失败时做 MySQL 补偿；投票通过 Lua 在 Redis 内原子更新个人状态和热度分数。帖子有纯 MySQL 的基础列表，也有结合 Redis 排序并批量回 MySQL 补详情的增强列表。项目里我重点关注跨存储一致性、JWT 生命周期、分页排序和投票状态机。”

## 面试复述自检

- 不看代码讲清注册、登录、发帖、投票四条链路。
- 解释路由认证边界、Gin context 的身份传播和统一响应码。
- 手算 `-1→1`、`1→0` 两种投票转换，说明 Lua 的原子范围。
- 区分 Redis 热度分、净票数、帖子发布时间和个人投票状态。
- 说清楚补偿事务的边界、配置热更新的边界，以及当前未实现的投票归档。

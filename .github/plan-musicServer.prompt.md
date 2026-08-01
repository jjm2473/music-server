## Plan: 音乐服务与播放器一次性交付

基于 Go 实现 server（serve+scan）并交付纯静态前端 player，一次性覆盖需求中的 WebDAV 只读访问、扫描索引、封面提取去重、分类播放与状态恢复。实现上优先复用成熟库与标准协议，重点把路径安全、软链接策略、隐藏文件过滤与本地状态恢复做成可测试的明确规则。

**Steps**
1. Phase 1 - 工程骨架与配置模型
2. 在 /Volumes/data/src/music-server/server 初始化 Go 模块与 CLI 入口，定义 serve/scan 两个子命令，默认加载工作目录 music-server.yaml。
3. 设计配置结构体并实现读取+默认值：common.root、common.path、serve.port、scan.ext（默认 mp3/ogg/m4a）；将配置校验前置（root 存在、path 以 / 开头、ext 去重小写化）。
4. 定义统一路径转换函数：文件系统绝对路径 <-> 对外 URL 路径（以 common.path 为前缀），供 serve 与 scan 复用。*后续步骤依赖本步*
5. Phase 2 - serve 命令（静态只读 + WebDAV PROPFIND）
6. 实现只读 HTTP 文件服务（GET/HEAD），将 common.path 映射至 common.root，禁止写方法（PUT/DELETE/MKCOL/COPY/MOVE/LOCK/UNLOCK/PATCH/POST）。
7. 实现路径安全链路：URL 解码与 clean、拒绝 .. 穿越；允许跟随软链接。
8. 实现 WebDAV 最小只读协议：OPTIONS、PROPFIND（Depth: 0/1）并返回 multistatus XML，字段至少包含 href、resourcetype、getcontentlength、getlastmodified、getcontenttype；目录枚举时隐藏点开头文件/目录。
9. 为 serve 增加统一错误语义：路径非法 403、不存在 404、方法不允许 405，避免泄露主机文件系统细节。*可与步骤 8 并行后集成*
10. Phase 3 - scan 命令（扫描、元数据、歌词、封面）
11. 递归扫描 common.root，跳过点开头目录与文件，按 scan.ext 过滤音频文件；输出顺序稳定化（按 URL 排序）以减小差异噪声。
12. 提取元数据（title/artist/album/length）与内嵌封面：使用成熟库读取音频标签；length 统一秒级数值。
13. 构造输出对象：url 为 common.path 映射路径；同名 .lrc 存在时写入 lrc；name=title，若 title 缺失则回退为 url 文件名；空字段从对象删除。
14. 封面提取与去重：写入 common.root/.cover/<md5>.<ext>，若同名已存在则复用不重复写；输出 cover 为 common.path/.cover/...。
15. 原子写入 music.json（先临时文件后 rename），避免中断导致损坏。*依赖步骤 11-14*
16. Phase 4 - player 静态站点（APlayer + 分类 + 收藏 + 恢复）
17. 在 /Volumes/data/src/music-server/player 构建纯静态页面，启动时读取 window 全局可配置 music.json URL（默认 /data/music.json）。
18. 基于音乐列表生成播放源，并自动构建全部、收藏、按专辑、按艺术家四类列表；每个列表可切换为当前播放列表。
19. 接入 APlayer 列表播放：同步当前列表、当前曲目、循环模式到 localStorage；首次加载时恢复并校验索引有效性。
20. 实现收藏：以 name-artist 作为 key 保存集合；列表加载后恢复收藏状态并生成收藏列表。
21. 为列表切换与恢复失败提供降级策略：目标歌曲缺失时回退到全部列表第一个可播项。*依赖步骤 18-20*
22. Phase 5 - 验证与文档收口
23. 编写 server 单元测试：路径穿越、软链接目标校验、隐藏文件过滤、URL 映射、封面 md5 去重、music.json 字段裁剪。
24. 编写集成验证脚本：serve + PROPFIND + scan 联调；覆盖正常与攻击路径（../、%2e%2e）。
25. 编写使用文档：配置示例、scan/serve 命令、常见错误与排查、player 全局变量配置说明。

**Relevant files**
- /Volumes/data/src/music-server/server/go.mod — Go 模块与依赖版本锁定。
- /Volumes/data/src/music-server/server/main.go — CLI 入口与子命令分发。
- /Volumes/data/src/music-server/server/internal/config/config.go — 配置结构、加载与校验。
- /Volumes/data/src/music-server/server/internal/pathmap/pathmap.go — 文件路径与 URL 路径双向转换。
- /Volumes/data/src/music-server/server/internal/security/path_guard.go — 路径 clean、穿越拦截、软链接解析与边界校验。
- /Volumes/data/src/music-server/server/internal/serve/http_handler.go — 静态只读 GET/HEAD 处理与方法拒绝。
- /Volumes/data/src/music-server/server/internal/serve/webdav_propfind.go — PROPFIND XML 响应与 Depth 处理。
- /Volumes/data/src/music-server/server/internal/scan/walker.go — 递归扫描、隐藏项过滤、扩展名过滤。
- /Volumes/data/src/music-server/server/internal/scan/metadata.go — title/artist/album/length 提取适配层。
- /Volumes/data/src/music-server/server/internal/scan/cover.go — 内嵌封面抽取、md5 命名、去重写入。
- /Volumes/data/src/music-server/server/internal/scan/index_writer.go — music.json 组装、空字段删除、原子落盘。
- /Volumes/data/src/music-server/player/index.html — 页面骨架、APlayer 挂载点、全局配置读取。
- /Volumes/data/src/music-server/player/app.js — 数据加载、分类构建、播放器初始化与状态恢复主流程。
- /Volumes/data/src/music-server/player/storage.js — 收藏与播放状态 localStorage 读写。
- /Volumes/data/src/music-server/player/style.css — 基础布局与响应式样式。
- /Volumes/data/src/music-server/README.md — 运行说明与验收步骤。

**Verification**
1. 单元测试：执行 server 包测试，重点断言 path_guard 在 ../ 与编码绕过场景返回拒绝，并验证软链接合法目标可访问。
2. 协议验证：curl OPTIONS 与 PROPFIND（Depth 0/1）检查 XML 结构与只读语义；写方法应返回 405。
3. 扫描验证：准备含 mp3/ogg/m4a、lrc、隐藏目录、嵌套目录、内嵌封面的样例库，运行 scan 后检查 music.json 字段规则（name 回退、空字段删除、length 秒）。
4. 封面验证：重复扫描同一批文件，确认 .cover 无重复写入且 cover 路径稳定。
5. 前端验证：打开 player 页面，检查分类列表、收藏恢复、当前播放与循环模式恢复；删除某首歌后再次打开应自动降级到可播项。
6. 端到端验证：scan 生成索引后启动 serve，player 从 /data/music.json 正常加载并可播放音频、歌词、封面。

**Decisions**
- 实现语言：Go（已对齐）。
- 交付策略：一次性全量交付（已对齐），不拆 MVP。
- WebDAV 范围：只读最小集（OPTIONS + PROPFIND），明确不实现写操作。
- 隐藏项策略：扫描与目录枚举均跳过点开头文件/目录；但 .cover 中被引用封面文件可按 URL 访问。
- 本次范围包含：server 与 player 全功能、测试与文档；不包含用户系统、上传接口、转码与权限体系。

**Further Considerations**
1. 元数据库选择优先级：先选对 mp3/ogg/m4a 覆盖完整且可提取封面的库；若某格式缺失，保留该格式文件但仅输出 name/url。
2. music.json 体积优化：若曲目规模很大，可在后续增加 gzip 与缓存头；当前阶段先保证正确性与稳定性。
3. 未来增量扫描：本计划先全量扫描，后续可基于 mtime/哈希加入增量能力。
# Music Server

静态音乐分享站点，包含两个部分：

- server：提供只读 HTTP 文件服务和 WebDAV PROPFIND 读取能力，支持扫描音乐并生成索引。
- player：纯静态前端播放器，读取 music.json 自动构建列表并播放。

## 目录结构

- server：Go 实现的命令行程序，支持 serve/scan。
- player：静态 HTML/CSS/JS 页面，基于 APlayer。

## 配置文件

默认读取当前工作目录下的 music-server.yaml。

```yaml
common:
	root: /root/music-server/data
	path: /data
serve:
	port: 8000
	webui: /root/music-server/player
scan:
	ext:
		- mp3
		- ogg
		- m4a
```

字段说明：

- common.root：音乐根目录，也是 music.json 与 .cover 输出目录。
- common.path：HTTP 映射前缀，前端按该前缀访问资源。
- serve.port：服务监听端口。
- serve.webui：静态页面根目录（可选），用于托管 player 等前端文件。
- scan.ext：扫描扩展名白名单，不区分大小写。

## Server 使用

### 1. 安装依赖

在 server 目录执行：

```bash
go mod tidy
```

### 2. 扫描音乐并生成索引

```bash
cd server
go run . scan -config ../music-server.yaml
```

会在 common.root 下输出：

- music.json
- .cover/<md5>.<ext>

扫描规则：

- 递归扫描子目录。
- 跳过点开头目录与文件。
- 提取 title/artist/album/length（length 当前对 mp3/ogg 可用）。
- 同名 .lrc 文件存在时写入 lrc。
- 内嵌封面提取到 .cover，按内容 md5 去重。

### 3. 启动只读服务

```bash
cd server
go run . serve -config ../music-server.yaml
```

能力：

- GET/HEAD 只读文件访问。
- OPTIONS。
- PROPFIND（Depth 0/1）用于 WebDAV 读取。
- 若请求路径不匹配 common.path 且配置了 serve.webui，则回退到 serve.webui 目录查找静态文件。
- 拒绝写方法（405）。

安全策略：

- URL 路径会做 clean 和映射校验。
- 防止路径穿越。
- 允许跟随软链接，软链接最终目标可在 common.root 外。

## Player 使用

player 是纯静态站点，可由 server 一并托管，或独立静态服务托管。

### 默认行为

- 启动时读取 /data/music.json。
- 自动构建：全部、收藏、按专辑、按艺术家 列表。
- 每个列表可切换为当前播放列表。

### 可配置项

在加载 app.js 前设置全局变量：

```html
<script>
	window.MUSIC_JSON_URL = "/data/music.json";
</script>
```

### 本地状态

使用 localStorage 保存并恢复：

- 收藏集合（key: name-artist）。
- 当前播放列表。
- 当前曲目。
- 循环模式。

## 验证

在 server 目录执行：

```bash
go test ./...
```

## 已知限制

- m4a 的时长字段可能为空（会按规则省略该字段）。
- 当前实现只覆盖 WebDAV 的只读最小子集（OPTIONS + PROPFIND）。
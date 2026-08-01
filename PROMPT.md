## overview
这是一个静态的音乐分享网站，包含两个部分：
1. server：提供http静态只读文件服务，提供webdav读取服务，提供一个命令扫描音乐文件并生成索引
2. player：纯前端静态html/js实现的音乐播放器


### server 技术参数
1. server支持两个命令，分别是serve和scan。默认读取工作目录下的 music-server.yaml 配置文件
2. 配置文件格式：
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
3. serve命令创建http服务监听端口（serve.port参数），同时支持 webdav 读取（也就是实现 PROPFIND），将根路径（common.path参数）的访问映射到根目录（common.root参数），注意防止路径穿越漏洞（但允许跟随软链接）。server.webui作为web根目录，仅提供http服务，通常是一个播放器。http请求资源时优先判断common.path，如果不匹配再查找server.webui路径下的文件或目录
4. scan命令以根目录（common.root参数）为起点扫描所有音乐文件（默认限定`.mp3,.ogg,.m4a`，可通过配置文件配置(scan.ext参数)），包括子目录（点开头目录和文件除外），提取音乐文件元数据，如果有相同文件名的`.lrc`歌词文件也收集路径。保存路径时需要按根路径（common.path参数）转换，方便前端 player 使用。提取的信息保存到根目录（common.root参数）下的 `music.json`，格式如下：
	```json
	[
		{
			"name": "Jungle P",
			"title": "Jungle P",
			"artist": "5050",
			"album": "One Piece",
			"length": 300,
			"url": "/data/One Piece/5050 - Jungle P.mp3",
			"lrc": "/data/One Piece/5050 - Jungle P.lrc",
			"cover": "/data/.cover/0c2379335f229390514e22901a2788fd.jpg"
		},
		...
	]
	```
	其中`length`表示音频长度，单位是秒。若某些元数据不存在或为空则删除对应字段。`name`字段等于`title`，如果`title`不存在则按`url`截取文件名；
	如果音乐文件内嵌了封面，则提取到根目录下的`.cover`目录，文件名是文件内容的md5，如果同名文件已存在则跳过提取。路径写入`cover`字段
5. 使用成熟方案实现，避免重复造轮子

### player 技术参数
1. 无后端接口，打开时读取`/data/music.json`（可以通过window上的全局变量配置）获取音乐列表
2. 根据音乐列表自动创建分类列表（专辑、艺术家），以及“全部”、“收藏”列表。每个列表都可以作为播放列表
3. “收藏”功能将`name-artist`作为 key 保存到 localstorage，不考虑冲突。在音乐列表加载以后根据 key 恢复收藏列表
4. 基于 [APlayer.js](https://github.com/DIYgod/APlayer) 实现播放功能和列表播放功能
5. 当前列表和当前播放曲目，以及当前播放列表循环模式保存到 localstorage，在下次打开时恢复状态

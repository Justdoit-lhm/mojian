<p align="center">
  <img src="logos/png/mojian-logo-horizontal-1600.png" alt="墨间" width="400">
</p>

<h1 align="center">墨间 · Mojian</h1>

<p align="center"><em>字落于纸，工具退于幕后。</em></p>

<p align="center">
  macOS 13+ &nbsp;·&nbsp; Linux amd64/arm64 &nbsp;·&nbsp; MyGo 原生界面 &nbsp;·&nbsp; 无 WebView &nbsp;·&nbsp; 无 cgo 构建
</p>

<p align="center">
  <a href="README.md">English</a> &nbsp;·&nbsp; <a href="README.zh-CN.md">简体中文</a>
</p>

<p align="center">
  <img src="docs/screenshot.png" alt="墨间中打开的一篇 Markdown 笔记" width="960">
</p>
<p align="center"><sub>左边是项目，眼前是文字，别无他物打扰。</sub></p>

## 墨间

墨间是一款为纯文本而生的 macOS 编辑器。它打开普通文件夹，编辑普通文件：不导入、不建库、不索引，也不需要账号。

窗口由 [MyGo](https://mygo.egoist.dev) 以原生界面绘制——在 macOS 上经 Metal 呈现——因此没有 WebView，没有 Electron，也没有 JavaScript 构建。一个标签页由若干可平铺的窗格组成；窗格可以编辑文件、预览图片或 PDF，也可以运行终端。终端进程活在独立的会话服务器里，退出应用也不会结束；项目还可以放在通过 SSH 连接的服务器上。

**墨**是文字，**间**是留白。工具退到纸后，文件留在眼前；编辑的节奏，由你掌握。

## 安装

从 [Releases](https://github.com/ox0pine/mojian/releases) 下载最新构建。

macOS：Apple Silicon 选 `mojian-<版本>-arm64.dmg`，Intel 选 `mojian-<版本>-amd64.dmg`；打开镜像，把墨间拖进「应用程序」。墨间需要 macOS 13 或更高版本。发布包采用 ad-hoc 签名、未经公证；如果系统拦截首次启动，请在 Finder 中右键墨间并选择「打开」。

Linux：安装 `mojian_<版本>_<架构>.deb`（amd64 或 arm64），或解压 `mojian-<版本>-linux-<架构>.tar.gz` 后运行其中的 `sh install.sh`，再从应用菜单启动。压缩包安装到用户目录，不需要 root。

## 功能

### 项目即文件夹

`⌘O` 打开任意文件夹，左上角的两个徽标分别切换最近的项目与服务器。每个项目保留自己的标签页与窗格布局。未指定项目时，首次启动会打开 `~/Documents/Mojian`，文件夹为空时留下一张欢迎页。

### 先写字，其余退后

`⌘S` 原子写入。编辑先留在内存，关窗时统一写入尚未保存的内容。

语法高亮覆盖 MyGo 所知的多种语言——Python、JavaScript、TypeScript、Go、Rust、C 与 C++、Swift、Kotlin、Ruby、JSON、YAML、TOML、SQL、HTML、CSS、Markdown 等。代码保留行号，散文自然换行。编辑器沿用 Zed 的键位：查找替换、多光标、整行移动、注释与折叠都在。

Markdown 窗格可在标题栏切换源码与预览；设置 › Editor 可让新开的 `.md` 直接以排版后的样子打开。图片（含 SVG）在窗格内等比呈现；安装 `pdftoppm` 时 PDF 逐页渲染（至多 60 页），否则以 Quick Look 生成首页。

### 标签页、窗格与 Explorer

一个标签页是一间工作台，由若干窗格组成。分屏会打开一个空窗格，并就地搜索文件；拖动窗格把手到另一窗格的边缘即可插入，也可用 `⌥⇧` 与方向键移动。`⌥` 加方向键切换焦点，`⇧⌘M` 或双击标题放大窗格。

`⇧⌘B` 展开 Explorer。搜索框筛选文件，`⇧⌘O` 直接聚焦搜索。顶部可新建文件与文件夹；右键菜单提供「在当前窗口打开」「在新标签页打开」、重命名（`F2`）、复制、剪切、复制路径、复制相对路径、在 Finder 中显示与移到废纸篓（`⌘⌫`）；文件夹与空白处还可粘贴。窗口重获焦点时树会重新读取磁盘，`.git`、`node_modules`、`dist`、`build` 等生成目录不会出现。

### 终端比窗口活得更久

`⇧⌘T` 在项目根目录打开一个真正的终端。它像文件窗格一样分屏、放大、关闭。Shell 运行在独立的会话服务器中：退出墨间只是断开，重新打开即可接回。`⇧⌘L` 通过 `~/.ssh/config` 中的主机开启 SSH 终端。

### 项目在服务器上

在设置 › Hosts 添加主机，再从服务器徽标中选择；地址可写 `ssh` 接受的任何形式（`buildbox`、`me@devbox` 或 `ssh://me@devbox:2222`）。连接时，墨间经 `ssh` 在主机上准备一个轻量的用户级会话服务器，转发它的 Unix socket，并把项目镜像到本地缓存；保存时写回远端。无需 root，不开放端口，也不注册为系统服务。Explorer、Git 页与终端在远端副本上照常工作。交互方式参考 [Kero](https://kero.sh)。

### 工作区一目了然

`⇧⌘G` 以 `HEAD` 为基准呈现工作区：左侧是改动文件与状态，右侧是带新旧行号的差异，增删用克制的色块区分。已暂存、未暂存与未跟踪的改动都在其中。双击文件即可打开，刷新按钮随时重读。

### 设置、命令面板与明暗

`⌘,` 在独立窗口中打开设置：左侧是可搜索的分类——General、Appearance、Editor、Fonts、Terminal、Shell、Notifications、Keyboard、Hosts、Git、Advanced——右侧是设置本身。每项改动即时生效并保存。

`⇧⌘P` 以模糊匹配搜索命令、文件与标签页。两套调色板皆经手工校准；默认随系统，也可由 View › Appearance 指定。

## 快捷键

### 文件与项目

| macOS | 动作 |
|---|---|
| `⌘O` | 打开项目文件夹 |
| `⇧⌘O` | 查找文件 |
| `⌘N` | 新建文件 |
| `⇧⌘N` | 新建文件夹 |
| `F2` | 重命名所选条目 |
| `⌘⌫` | 将所选条目移到废纸篓 |
| `⌘S` | 立即保存 |

### 标签页与窗格

| macOS | 动作 |
|---|---|
| `⌘T` | 新建标签页 |
| `⌘W` / `⇧⌘W` | 关闭窗格 / 标签页 |
| `⇧⌘R` | 重命名标签页 |
| `⌃⇥` / `⌃⇧⇥` | 下一个 / 上一个标签页 |
| `⌘1…⌘9` | 显示标签页 |
| `⇧⌘D` / `⇧⌘E` | 向右 / 向下分屏 |
| `⌥⌘←` / `⌥⌘↑` | 向左 / 向上分屏 |
| `⌥←` / `⌥→` / `⌥↑` / `⌥↓` | 切换窗格焦点 |
| `⌥⇧←` / `⌥⇧→` / `⌥⇧↑` / `⌥⇧↓` | 移动所选窗格 |
| `⇧⌘M` | 放大窗格 |

### 面板与命令

| macOS | 动作 |
|---|---|
| `⇧⌘B` | 显示或隐藏 Explorer |
| `⇧⌘T` | 打开或关闭终端窗格 |
| `⇧⌘L` | 打开 SSH 终端 |
| `⇧⌘G` | 显示 Git 差异页 |
| `⇧⌘P` | 命令面板 |
| `⌘,` | 设置 |

双击标签页可重命名，双击窗格标题可放大；右键标签页可重置名称、关闭其他标签页，以及执行其余操作。

## 从源码构建

需要 Go 1.27.3 或更高版本。

```sh
git clone https://github.com/ox0pine/mojian.git
cd mojian

go tool mygo dev        # 热重载的开发版
go test ./...           # 单元与集成测试
go tool mygo vet .      # go vet 与 MyGo 的界面生命周期检查
go tool mygo build      # 生成 build/<os>-<arch>/Mojian.app 与 .dmg
```

MyGo 以 Go tool 的形式固定在 `go.mod` 中，首次运行 `go tool mygo` 会下载对应版本。`go tool mygo dev -web` 在浏览器中打开界面，文件保存在内存里。改动会话协议或 `cmd/mojian-server` 后，用 `sh tools/build-remote.sh` 重新生成内嵌的远端二进制。

## 形色

墨 `#1F1B16`，纸 `#FBF7EF`，朱砂 `#D24A2B`。在墨间的品牌里，几乎一切都是墨与纸，只有光标保留一点朱色——它是整页唯一愿意提高的声音。品牌文件与导出见 [`logos/README.md`](logos/README.md)。

## 致谢

动作图标来自 [Lucide](https://lucide.dev)（ISC），文件图标来自 [Material Icon Theme](https://github.com/PKief/vscode-material-icon-theme)（MIT），字体为 [JetBrains Mono](https://www.jetbrains.com/lp/mono/)（OFL），终端由 libghostty-vt 驱动，伪终端来自 [creack/pty](https://github.com/creack/pty)（MIT）。Logo 以墨、纸与一点朱砂构成，字标取自 Literata（SIL OFL）。窗口形制参考 [GoRex](https://github.com/egoist/gorex)，远程主机与设置参考 [Kero](https://kero.sh)。全部由 [MyGo](https://mygo.egoist.dev) 绘制。

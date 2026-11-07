<p align="center">
  <img src="logos/png/mojian-logo-horizontal-1600.png" alt="墨间" width="400">
</p>

<h1 align="center">墨间 · Mojian</h1>

<p align="center"><em>字落于纸，工具退于幕后。</em></p>

<p align="center">
  macOS 13+ &nbsp;·&nbsp; MyGo 原生界面 &nbsp;·&nbsp; 无 WebView &nbsp;·&nbsp; 零 cgo
</p>

<p align="center">
  <a href="README.md">English</a> &nbsp;·&nbsp; <a href="README.zh-CN.md">简体中文</a>
</p>

<p align="center">
  <img src="docs/screenshot.png" alt="墨间的编辑窗口" width="960">
</p>
<p align="center"><sub>阅读与书写</sub></p>

## 关于

墨间是一款为纯文本而生的编辑器。它以 [MyGo](https://mygo.egoist.dev) 的原生界面绘制每一帧：没有 WebView，没有 cgo，没有 JavaScript 构建——一枚 Go 二进制，在 macOS 上由 Metal 呈现。

**墨**是文字，**间**是留白。工具退到纸后，文件留在眼前；编辑的节奏，由你掌握。

## 书写

- **项目即文件夹。** 打开任意文件夹即可开始，不导入、不建库、不索引。`⌘O` 在窗内选择目录，左上角的徽标在项目与服务器之间切换；每个项目保留自己的标签页与窗格布局。未指定项目时，首次启动会打开 `~/Documents/Mojian`，为空时备下一张欢迎页。
- **纯文本，落笔由你。** `⌘S` 原子写入；编辑先留在内存，关窗时写入。语法高亮覆盖 MyGo 编辑器所知的语言——Python、JavaScript、TypeScript、Go、Rust、C、C++、Swift、Kotlin、JSON、YAML、TOML、SQL、HTML、CSS、Markdown 等；代码保留行号，散文自然换行。
- **Explorer，项目树而非文件列表。** `⇧⌘B` 展开项目；搜索框筛选文件，`⇧⌘O` 直接抵达。资源树顶部可新建文件与文件夹，窗口重获焦点时树会刷新。右键文件可在当前窗口打开或开入新标签页；每个条目都可重命名（`F2`）、复制、剪切、复制路径、复制相对路径、在 Finder 中显示、移到废纸篓（`⌘⌫`）；文件夹与空白处还可粘贴。`.git`、`node_modules`、`dist`、`build` 等目录不进入视野。
- **标签页与分屏。** 一个标签页是一间工作台，由若干窗格组成。文件分屏先给出一个空窗格，以就地搜索选择文件；不断分屏，不设四格之限。拖动窗格把手到另一窗格的边缘即可插入，也可用 `⌥⇧` 与方向键移动；`⌥` 加方向键切换焦点，`⇧⌘M` 或双击窗格标题放大。
- **Markdown，两种面貌。** Markdown 窗格右上角可在源码与预览间切换；设置中可选择新开的 `.md` 直接呈现为排版后的页面。预览读取实时缓冲区，并保留自己的滚动位置。
- **图像与 PDF。** 图片（含 SVG）在窗格内等比呈现，附尺寸与「打开」；PDF 在安装了 `pdftoppm` 时逐页渲染（至多 60 页），否则以 Quick Look 生成首页预览。

## 空间

- **终端在窗内。** `⇧⌘T` 在项目根目录打开一个真正的终端窗格；它与文件窗格一样分屏、放大、关闭。终端会话比窗口活得更久：退出墨间只是断开，重新打开即可接回。`⇧⌘L` 通过 `~/.ssh/config` 中的主机开启 SSH 终端。
- **项目在服务器上。** 服务器徽标在「本地」与设置中登记的主机之间切换。连接时，墨间经 `ssh` 在主机上准备一个轻量会话服务器；无需 root，不开放端口，也不注册为服务。远程项目镜像到本地，保存时回传；Explorer、Git 与终端照常工作。主机沿用 `~/.ssh/config` 的约定，远程主机的交互方式取自 [Kero](https://kero.sh)。
- **Git 差异页。** `⇧⌘G` 以 `HEAD` 为基准呈现工作区：左侧列出改动文件与状态，右侧是带新旧行号的差异，增删以克制的色块区分；已暂存、未暂存与未跟踪都在其中。双击文件即可打开，刷新按钮随时重读。
- **独立的设置窗口。** `⌘,` 打开独立窗口，形制取自 Kero：左侧是可搜索的分类，右侧是设置本身。General、Appearance、Editor、Fonts、Terminal、Shell、Notifications、Keyboard、Hosts、Git、Advanced 覆盖字体、行距、Tab 宽度、换行、Markdown、终端输入、光标与回滚行数、通知、主机与数据路径。改动即时生效并保存。
- **命令面板。** `⇧⌘P` 以模糊匹配列出命令、文件与标签页，让键盘抵达应用的多数角落。
- **明、暗，或随系统。** 两套调色板皆经手工校准；默认随系统，也可由 View › Appearance 指定。

## 快捷键

| macOS | 动作 |
|---|---|
| `⌘O` | 打开项目文件夹 |
| `⇧⌘O` | 查找文件 |
| `⌘N` | 新建文件 |
| `⇧⌘N` | 新建文件夹 |
| `F2` | 重命名所选条目 |
| `⌘⌫` | 将所选条目移到废纸篓 |
| `⌘T` | 新建标签页 |
| `⌘W` / `⇧⌘W` | 关闭窗格 / 标签页 |
| `⇧⌘R` | 重命名标签页 |
| `⌃⇥` / `⌃⇧⇥` | 下一个 / 上一个标签页 |
| `⌘1…⌘9` | 显示标签页 |
| `⇧⌘B` | 显示或隐藏 Explorer |
| `⇧⌘T` | 打开或关闭终端窗格 |
| `⇧⌘L` | 打开 SSH 终端 |
| `⇧⌘G` | 显示 Git 差异页 |
| `⇧⌘D` / `⇧⌘E` | 向右 / 向下分屏 |
| `⌥⌘←` / `⌥⌘↑` | 向左 / 向上分屏 |
| `⌥←` / `⌥→` / `⌥↑` / `⌥↓` | 切换窗格焦点 |
| `⌥⇧←` / `⌥⇧→` / `⌥⇧↑` / `⌥⇧↓` | 移动所选窗格 |
| `⇧⌘M` | 放大窗格 |
| `⇧⌘P` | 命令面板 |
| `⌘,` | 设置 |
| `⌘S` | 立即保存 |

双击标签页可重命名；双击窗格标题可放大。右键标签页可重置名称、关闭其他标签页，以及执行其余操作。

## 形色

墨 `#1F1B16`，纸 `#FBF7EF`，朱砂 `#D24A2B`。在墨间的品牌里，几乎一切是墨与纸，只有光标保留一点朱色——它是整页唯一愿意提高的声音。品牌文件与导出见 [`logos/README.md`](logos/README.md)。

## 致谢

动作图标来自 [Lucide](https://lucide.dev)（ISC），文件图标来自 [Material Icon Theme](https://github.com/PKief/vscode-material-icon-theme)（MIT），字体为 [JetBrains Mono](https://www.jetbrains.com/lp/mono/)（OFL），终端由 libghostty-vt 驱动，伪终端来自 [creack/pty](https://github.com/creack/pty)（MIT）。Logo 以墨、纸与一点朱砂构成，字标取自 Literata（SIL OFL）。窗口形制参考 [GoRex](https://github.com/egoist/gorex)，远程主机与设置参考 [Kero](https://kero.sh)。全部由 [MyGo](https://mygo.egoist.dev) 绘制。

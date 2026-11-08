<h1 align="center">墨间 · Mojian</h1>

<p align="center"><em>字落于纸，工具退于幕后。</em></p>

<p align="center">
  macOS 13+ &nbsp;·&nbsp; Linux amd64/arm64 &nbsp;·&nbsp; MyGo 原生界面 &nbsp;·&nbsp; 无 WebView &nbsp;·&nbsp; 无 cgo
</p>

<p align="center">
  <a href="README.md">English</a> &nbsp;·&nbsp; <a href="https://github.com/ox0pine/mojian/releases">下载</a>
</p>

<p align="center">
  <img src="docs/screenshot.png" alt="墨间：同一标签页中的编辑器、选择文件的窗格和两个终端" width="1200">
</p>

## 墨间是什么

墨间是一款直接打开普通文件夹的纯文本编辑器。没有资料库、索引、数据库，也不需要账号：项目就是你选择的文件夹，笔记就是磁盘上的文件。编辑内容先留在内存，保存时才写入。

一个标签页是由多个平铺窗格组成的工作区。每个窗格可以编辑文件、预览图片或 PDF，也可以运行终端。界面由 [MyGo](https://mygo.egoist.dev) 绘制为原生 UI：macOS 使用 Metal，Linux 使用 GTK 3。没有 WebView、没有 Electron，也没有 JavaScript 构建流程。

## 安装

从 [Releases](https://github.com/ox0pine/mojian/releases) 下载最新构建。

### macOS

- Apple Silicon：`mojian-<版本>-arm64.dmg`
- Intel：`mojian-<版本>-amd64.dmg`

打开镜像，把墨间拖进「应用程序」。发布包采用 ad-hoc 签名、未经公证；如果系统拦截首次启动，请在 Finder 中右键墨间并选择「打开」。

### Linux

安装 Debian 包：

```sh
sudo apt install ./mojian_<版本>_<架构>.deb
```

或者安装到用户目录，不需要 root：把 `install.sh` 和
`mojian-<版本>-linux-<架构>.tar.gz` 放在同一目录，然后运行：

```sh
sh install.sh
```

Linux 版需要 GTK 3 和桌面会话。

## 窗口

顶部栏就是应用的入口：左边是服务器与项目，中间是当前标签页，右边是文件树、拆分、终端、新建标签页和设置按钮。设置会在独立窗口中打开。

每个窗格都有自己的标题栏，显示文件名、图片或终端，并且无论是否拥有焦点，都保留拆分、放大和关闭按钮。空窗格也有用途：拆分后会出现一个带内联文件搜索的空窗格，下一个文件就在你正在看的位置打开。

### 项目与文件树

macOS 用 `Cmd+O`，Linux 用 `Ctrl+O` 打开文件夹。项目徽标切换最近项目，服务器徽标在本地与已配置主机之间切换。每个项目保留自己的标签页、窗格和最近文件。未指定项目时，首次启动会打开 `~/Documents/Mojian`，文件夹为空时留下一张欢迎笔记。

`Shift+Cmd+B` / `Ctrl+Shift+B` 显示或隐藏文件树。文件树搜索框用于过滤当前目录；**打开文件**（`Shift+Cmd+O` / `Ctrl+Shift+O`）会在窗口中央打开项目文件选择框：输入即可过滤，按 Enter 或点击文件即可打开。文件树支持新建文件与文件夹；右键可打开、重命名、复制、剪切、复制路径、在文件管理器中显示、移到废纸篓。`.git`、`node_modules`、`dist`、`build` 等生成目录默认隐藏。

### 标签页与窗格

`Cmd+T` / `Ctrl+T` 新建标签页。标签页内部是一棵二叉拆分树：`Shift+Cmd+D` / `Shift+Cmd+E` 向右或向下拆分，`Option+Cmd` / `Ctrl+Alt` 加方向键向左或向上拆分。拖动窗格标题到另一个窗格的边缘，即可把窗格放到那里。`Option` / `Alt` 加方向键在窗格之间移动焦点；`Shift+Cmd+M` / `Ctrl+Shift+M` 放大当前窗格。

双击标签页可重命名，双击窗格标题可放大，右键标签页可打开菜单。关闭窗格时，相邻窗格立刻占据空间，关闭的卡片会在原位置淡出。

### 终端

`Shift+Cmd+T` / `Ctrl+Shift+T` 在项目根目录打开终端。终端也是一个普通窗格：可以拆分、移动、放大或关闭。在 macOS 和 Linux 上，终端运行在独立的会话服务器中，退出墨间只会断开，重新打开会接回原来的 shell。

`Shift+Cmd+L` / `Ctrl+Shift+L` 打开到设置 › 主机中某个主机的 SSH 终端。

## 写作与预览

代码保留行号，散文自动换行。语法高亮覆盖 Python、JavaScript、TypeScript、Go、Rust、C/C++、Swift、Kotlin、Ruby、JSON、YAML、TOML、SQL、HTML、CSS、Markdown 等语言。

Markdown 窗格可在源码与预览之间切换；设置 › 编辑器可以让新打开的 `.md` 默认显示渲染结果。图片（包括 SVG）在窗格内查看；安装 `pdftoppm` 后 PDF 可逐页渲染，macOS 还会回退到 Quick Look 首页。

`Cmd+S` / `Ctrl+S` 原子写入所有未保存的笔记；关闭窗口时也会保存未提交的修改。

## Git、命令面板与设置

`Shift+Cmd+G` / `Ctrl+Shift+G` 打开工作区页面：左侧是相对 `HEAD` 的变更状态，右侧是带行号栏的 diff。暂存、未暂存和未跟踪的改动都会列出。

`Shift+Cmd+P` / `Ctrl+Shift+P` 用模糊匹配搜索命令、文件和标签页。`Cmd+,` / `Ctrl+,` 打开设置窗口：通用、外观、编辑器、终端、Shell、通知、快捷键、主机、Git、高级、更新。搜索可以按名称或关键词找到任意设置项，`Cmd+F` 聚焦搜索框。窗口默认跟随系统外观，也可在「视图 › 外观」中覆盖。

## 远程服务器

在设置 › 主机中添加地址，然后在主机行选择「配置服务器」。墨间会把 `mojian-server` 与 `libghostty-vt` 上传到该主机的 `~/.local/share/mojian`，并检查服务器能否响应。不需要 root，不监听端口，也不安装系统服务。

从顶部服务器徽标选择该主机，即可直接编辑远端目录：文件树读取目录信息，打开文件时才读取内容，保存和文件操作直接写回远端，Git 也在远端运行。项目打开后会保持加载，切回时恢复标签页和终端，不需要重新读取。

远端服务器及其终端在 SSH 连接或墨间断开后仍然存活。每个终端都会记住所属的服务器；`mojian-server -stop` 可以显式结束服务器。

## 更新

设置 › 更新会检查 GitHub 上最新的签名版本。**检查更新**会提示当前已是最新版本，或给出新版本；**安装更新**会下载并校验签名；**重启更新**会应用已暂存的更新并重新启动墨间。

自动更新要求应用所在位置可写：macOS 应用包或 Linux 用户目录安装。安装在 `/opt` 的 `.deb` 由包管理器负责更新。

## 快捷键

应用内会按平台显示对应按键；下表同时列出两种写法。

| 操作 | macOS | Linux |
|---|---|---|
| 打开项目 | `⌘O` | `Ctrl+O` |
| 打开文件 | `⇧⌘O` | `Ctrl+Shift+O` |
| 新建文件 | `⌘N` | `Ctrl+N` |
| 新建文件夹 | `⇧⌘N` | `Ctrl+Shift+N` |
| 重命名选中项 | `F2` | `F2` |
| 移到废纸篓 | `⌘⌫` | `Ctrl+Backspace` |
| 保存 | `⌘S` | `Ctrl+S` |
| 新建标签页 | `⌘T` | `Ctrl+T` |
| 关闭窗格 / 标签页 | `⌘W` / `⇧⌘W` | `Ctrl+W` / `Ctrl+Shift+W` |
| 下一个 / 上一个标签页 | `⌃⇥` / `⌃⇧⇥` | `Ctrl+Tab` / `Ctrl+Shift+Tab` |
| 显示标签页 1-9 | `⌘1…⌘9` | `Ctrl+1…Ctrl+9` |
| 向右 / 向下拆分 | `⇧⌘D` / `⇧⌘E` | `Ctrl+Shift+D` / `Ctrl+Shift+E` |
| 向左 / 向上拆分 | `⌥⌘←` / `⌥⌘↑` | `Ctrl+Alt+Left` / `Ctrl+Alt+Up` |
| 聚焦相邻窗格 | `⌥` + 方向键 | `Alt` + 方向键 |
| 移动当前窗格 | `⌥⇧` + 方向键 | `Alt+Shift` + 方向键 |
| 放大窗格 | `⇧⌘M` | `Ctrl+Shift+M` |
| 显示 / 隐藏文件树 | `⇧⌘B` | `Ctrl+Shift+B` |
| 打开 / 关闭终端 | `⇧⌘T` | `Ctrl+Shift+T` |
| SSH 终端 | `⇧⌘L` | `Ctrl+Shift+L` |
| Git 工作区 | `⇧⌘G` | `Ctrl+Shift+G` |
| 命令面板 | `⇧⌘P` | `Ctrl+Shift+P` |
| 设置 | `⌘,` | `Ctrl+,` |

终端与 SSH 快捷键仅在 macOS 和 Linux 上可用。

## 从源码构建

需要 Go 1.27.3 或更高版本。

```sh
git clone https://github.com/ox0pine/mojian.git
cd mojian

go tool mygo dev        # 开发模式，修改后自动重载
go test ./...           # 单元与集成测试
go tool mygo vet .      # go vet 与 MyGo 的 UI 生命周期检查
```

按需构建发布目标：

```sh
go tool mygo build -platform darwin/arm64,darwin/amd64
go tool mygo build -platform linux/amd64,linux/arm64
```

产物位于 `build/<系统>-<架构>/`。macOS 生成 `.app` 与 `.dmg`；Linux 生成 `.tar.gz` 与 `.deb`。MyGo 作为 Go tool 固定在 `go.mod` 中，首次运行 `go tool mygo` 会下载对应版本。

`go tool mygo dev -web` 会在浏览器中打开界面，文件只保存在内存里。修改会话协议或 `cmd/mojian-server` 后，用 `sh tools/build-remote.sh` 重新生成嵌入的远程二进制。

## 发布

发布由 tag 驱动。推送 `vX.Y.Z` 会运行
[`.github/workflows/release.yml`](.github/workflows/release.yml)：先测试，再构建全部平台，创建草稿 Release，上传安装包和 `SHA256SUMS`，所有上传成功后才正式发布。详见 [`RELEASING.md`](RELEASING.md)。

每次发布包含：

- macOS：`mojian-<版本>-arm64.dmg`、`mojian-<版本>-amd64.dmg`
- Linux：`mojian-<版本>-linux-{amd64,arm64}.tar.gz`、对应的 `.deb` 包、`install.sh`
- 签名更新包（`*-darwin-*.tar.gz`、`*-linux-*.tar.gz`）与 `update-<target>.json` 清单
- `SHA256SUMS`

## 设计

墨 `#1F1B16`，纸 `#FBF7EF`，朱砂 `#D24A2B`。墨间的识别里几乎一切都是纸上的墨，只有光标保留朱砂，那是页面唯一提高的声音。标志与导出文件见 [`logos/README.md`](logos/README.md)。

## 致谢

图标来自 [Lucide](https://lucide.dev)（ISC）；文件图标来自 [Material Icon Theme](https://github.com/PKief/vscode-material-icon-theme)（MIT）；字体为 [JetBrains Mono](https://www.jetbrains.com/lp/mono/)（OFL）；终端由 libghostty-vt 驱动，PTY 使用 [creack/pty](https://github.com/creack/pty)（MIT）。标志由墨、纸与一枚朱砂光标组成，字标轮廓取自 Literata（SIL OFL）。窗口与交互参考 [GoRex](https://github.com/egoist/gorex)；远程主机与设置参考 [Kero](https://kero.sh)。界面由 [MyGo](https://mygo.egoist.dev) 绘制。

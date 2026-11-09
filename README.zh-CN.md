# 墨间 · Mojian

> 字落于纸，工具退于幕后。

<p>
  <a href="https://awesomemygo.dev/apps/mojian"><img src="https://awesomemygo.dev/badge.svg" alt="Listed on Awesome MyGo" height="22"></a>
  <a href="https://github.com/ox0pine/mojian/releases"><img src="https://img.shields.io/github/v/release/ox0pine/mojian?style=flat&amp;labelColor=1F1B16&amp;color=D24A2B" alt="最新版本" width="104" height="22"></a>
  <a href="https://github.com/ox0pine/mojian/releases"><img src="https://img.shields.io/github/downloads/ox0pine/mojian/total?style=flat&amp;labelColor=1F1B16&amp;color=D24A2B" alt="下载次数" width="110" height="22"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/ox0pine/mojian?style=flat&amp;labelColor=1F1B16&amp;color=1F1B16" alt="MIT 许可证" width="86" height="22"></a>
</p>

墨间是一款直接打开普通文件夹的原生纯文本编辑器。编辑文件、预览图片与 PDF、运行常驻终端，并在平铺工作区中查看 Git diff。没有数据库、账号、WebView 或 cgo。

[下载最新版本](https://github.com/ox0pine/mojian/releases) · [English](README.md) · [从源码构建](#从源码构建)

![墨间：同一标签页中的编辑器、选择文件的窗格和两个终端](docs/screenshot.png?v=6b38e5e)

## 功能

- **普通文件与文件夹。** 项目就是磁盘上的文件夹。没有导入、索引或同步，编辑内容留在内存，保存时才写入。
- **标签页即工作区。** 每个标签页包含一棵拆分树，装着编辑器、图片与 PDF 查看器和终端。
- **终端常驻。** shell 运行在独立的会话服务器中，重新打开墨间时接回。
- **内置 Git diff。** 不离开窗口即可查看已暂存、未暂存和未跟踪的改动。
- **通过 SSH 的远程项目。** 文件、Git 和终端直接在配置的主机上运行。
- **单个 Go 二进制的原生界面。** [MyGo](https://mygo.egoist.dev) 在 macOS 上用 Metal、在 Linux 上用 GTK 3 绘制界面。没有 Electron，也没有 JavaScript 构建。

## 安装

从 [Releases](https://github.com/ox0pine/mojian/releases) 下载最新构建。

### macOS

| 架构 | 下载 |
|---|---|
| Apple Silicon | `mojian-<版本>-arm64.dmg` |
| Intel | `mojian-<版本>-amd64.dmg` |

打开磁盘映像，把墨间拖进「应用程序」。发布构建采用 ad-hoc 签名、未经公证。如果 macOS 拦截首次启动，请在 Finder 中右键墨间并选择 **打开**。

### Linux

墨间需要 GTK 3 和桌面会话。

安装 Debian 包：

```sh
sudo apt install ./mojian_<版本>_<架构>.deb
```

如果想安装到用户目录、不需要 root：把 `install.sh` 和
`mojian-<版本>-linux-<架构>.tar.gz` 放在同一目录，然后运行：

```sh
sh install.sh
```

## 项目与文件

macOS 用 `Cmd+O`，Linux 用 `Ctrl+O` 打开文件夹。项目徽标切换最近项目，服务器徽标在本地与已配置的 SSH 主机之间切换。每个项目保留自己的标签页、窗格和最近文件。

首次启动且未选择项目时，墨间会打开 `~/Documents/Mojian`；文件夹为空时留下一张欢迎笔记。

文件树展示磁盘上真实存在的目录：

- 用 `Shift+Cmd+B` / `Ctrl+Shift+B` 显示或隐藏。
- 用文件树的搜索框过滤可见目录。
- 用 **打开文件**（`Shift+Cmd+O` / `Ctrl+Shift+O`）搜索项目中的所有文件。
- 直接在文件树中新建文件与文件夹。
- 右键可打开、重命名、创建副本、复制、剪切、复制路径、在文件管理器中显示，或移到废纸篓。

`.git`、`node_modules`、`dist`、`build` 等生成目录默认隐藏。

## 标签页、窗格与终端

按 `Cmd+T` / `Ctrl+T` 打开标签页。用 `Shift+Cmd+D` / `Shift+Cmd+E` 向右或向下拆分当前窗格，用 `Option+Cmd` / `Ctrl+Alt` 加方向键向左或向上拆分。新窗格会带一个内联文件搜索。

拖动窗格标题到另一个窗格的边缘即可移动它。用 `Option` / `Alt` 加方向键在窗格之间移动焦点，用 `Option+Shift` / `Alt+Shift` 加方向键移动当前窗格，用 `Shift+Cmd+M` / `Ctrl+Shift+M` 放大它。

双击标签页重命名，双击窗格标题放大，或右键标签页查看更多操作。

按 `Shift+Cmd+T` / `Ctrl+Shift+T` 在项目根目录打开终端。终端也是窗格，可以像其他窗格一样拆分、移动、放大和关闭。在 macOS 和 Linux 上，退出墨间后 shell 仍然存活，重新打开会接回。

按 `Shift+Cmd+L` / `Ctrl+Shift+L` 打开到 **设置 › 主机** 中某个主机的 SSH 终端。

## 写作与预览

代码保留行号，散文自动换行。语法高亮覆盖 Python、JavaScript、TypeScript、Go、Rust、C 与 C++、Swift、Kotlin、Ruby、JSON、YAML、TOML、SQL、HTML、CSS、Markdown 等语言。

Markdown 窗格可在标题栏切换源码与预览。**设置 › 编辑器** 可以让新打开的 `.md` 文件默认渲染。图片（包括 SVG）在窗格内的查看器中打开；安装 `pdftoppm` 后 PDF 逐页渲染，macOS 上还会回退到 Quick Look 首页。

按 `Cmd+S` / `Ctrl+S` 原子写入所有未保存的笔记；关闭窗口时也会写入未保存的修改。

## Git、命令与设置

### Git

按 `Shift+Cmd+G` / `Ctrl+Shift+G` 将工作区与 `HEAD` 比较。左侧显示状态，右侧是带两列行号的 diff，包含已暂存、未暂存和未跟踪的改动。

### 命令面板

按 `Shift+Cmd+P` / `Ctrl+Shift+P` 用模糊匹配搜索命令、文件和标签页。

### 设置

按 `Cmd+,` / `Ctrl+,` 打开独立的设置窗口：通用、外观、编辑器、终端、Shell、通知、快捷键、主机、Git、高级、更新。

搜索可以按名称或关键词找到设置项，`Cmd+F` 聚焦搜索框。墨间默认跟随系统外观；**视图 › 外观** 可以覆盖它。

## 远程项目

1. 在 **设置 › 主机** 中添加主机。
2. 在该行选择 **配置服务器**。墨间会把 `mojian-server` 与 `libghostty-vt` 上传到 `~/.local/share/mojian`，并检查服务器能否响应。
3. 从服务器徽标选择该主机并打开文件夹。

配置不需要 root 权限、监听端口或系统服务。

文件树列出主机上的文件。打开文件时才读取内容，保存和文件操作直接写回。Git 也在主机上运行。项目打开后保持加载，切回时恢复标签页和终端，不需要重新读取。

远端服务器及其终端在墨间或 SSH 连接断开后仍然存活。每个终端都会记住所属的服务器。运行 `mojian-server -stop` 可以显式结束服务器。

## 更新

打开 **设置 › 更新** 检查 GitHub 上最新的签名版本：

- **检查更新** 报告是否有更新版本。
- **安装更新** 下载并校验签名。
- **重启更新** 应用已暂存的更新并重新启动墨间。

自动更新要求安装位置可写，例如 macOS 应用包或 Linux 用户目录安装。安装在 `/opt` 的 `.deb` 由包管理器负责更新。

## 快捷键

应用内会按平台显示对应按键。下表是默认绑定；**设置 › 快捷键** 可以重新绑定、恢复默认或解除绑定。

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

墨间需要 Go 1.27.3 或更高版本。MyGo 作为 Go tool 固定在 `go.mod` 中，所以第一次运行 `go tool mygo` 会下载对应版本。

```sh
git clone https://github.com/ox0pine/mojian.git
cd mojian

go tool mygo dev        # 开发模式，修改后自动重载
go test ./...           # 单元与集成测试
go tool mygo vet .      # go vet 与 MyGo 的 UI 生命周期检查
```

`go tool mygo dev -web` 会在浏览器中打开界面，文件只保存在内存里。

### 无头 UI 测试工具

`cmd/harness` 通过 [mygo-mcp](https://github.com/auron-labs/mygo-mcp) 把真实的窗口与设置视图提供给 MCP 客户端，agent 无需桌面窗口即可检查、点击、输入并截图界面：

```sh
go run ./cmd/harness                 # http://127.0.0.1:8080/mcp
go run ./cmd/harness -addr 127.0.0.1:8090
```

每次运行都会在临时目录中生成一个一次性项目；这些夹具绝不会打开原生窗口、启动终端，也不会触碰你的项目与设置。该端点没有身份验证，请保持在回环地址上。

### 发布构建

```sh
go tool mygo build -platform darwin/arm64,darwin/amd64
go tool mygo build -platform linux/amd64,linux/arm64
```

产物位于 `build/<系统>-<架构>/`。macOS 生成 `.app` 与 `.dmg`；Linux 生成 `.tar.gz` 与 `.deb`。

修改会话协议或 `cmd/mojian-server` 后，重新生成嵌入的远程二进制：

```sh
sh tools/build-remote.sh
```

### 发布版本

推送 `vX.Y.Z` tag 会运行 [`.github/workflows/release.yml`](.github/workflows/release.yml)：先测试代码树，再构建全部平台，创建草稿 Release，上传安装包和 `SHA256SUMS`，所有上传成功后才正式发布。完整流程见 [`RELEASING.md`](RELEASING.md)。

每次发布包含：

- macOS：`mojian-<版本>-arm64.dmg` 和 `mojian-<版本>-amd64.dmg`
- Linux：`mojian-<版本>-linux-{amd64,arm64}.tar.gz`、对应的 `.deb` 包和 `install.sh`
- 签名更新包（`*-darwin-*.tar.gz`、`*-linux-*.tar.gz`）与 `update-<target>.json` 清单
- `SHA256SUMS`

## 设计

墨 `#1F1B16`，纸 `#FBF7EF`，朱砂 `#D24A2B`。几乎一切都是纸上的墨，只有光标保留朱砂，那是页面唯一提高的声音。标志母版与导出文件见 [`logos/README.md`](logos/README.md)。

## 致谢

- 操作图标：[Lucide](https://lucide.dev)，ISC
- 文件图标：[Material Icon Theme](https://github.com/PKief/vscode-material-icon-theme)，MIT
- 界面字体：[JetBrains Mono](https://www.jetbrains.com/lp/mono/)，OFL
- 字标：Literata，SIL OFL
- 终端：libghostty-vt 与 [creack/pty](https://github.com/creack/pty)，MIT
- 窗口与交互参考：[GoRex](https://github.com/egoist/gorex) 与 [Kero](https://kero.sh)
- 原生界面：[MyGo](https://mygo.egoist.dev)

## 许可证

MIT，详见 [`LICENSE`](LICENSE)。

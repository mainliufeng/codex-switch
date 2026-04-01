# codex-switch

`codex-switch` 是一个面向 Linux + Xorg 系统托盘的小程序，用来在多个 Codex 登录配置之间快速切换。

当前版本已经包含托盘图标和启动器图标。

## 截图

![Codex Switch systray screenshot](assets/systray.png)

它只做本地快照管理，不负责登录流程：

- 保存当前 `~/.codex/auth.json`
- 从已保存账号列表里按邮箱切换
- 可选删除已保存账号
- 提供编译脚本和用户级安装脚本

## 技术栈

- Go 1.26+
- [`getlantern/systray`](https://github.com/getlantern/systray)
- Linux 上依赖 GTK3 + Ayatana AppIndicator

选择 Go 的原因很直接：单二进制构建简单，适合做这种常驻托盘工具；核心逻辑也容易用单元测试覆盖。

## 工作方式

程序会读取当前的 `~/.codex/auth.json`，从其中 `tokens.id_token` 的 JWT payload 解析 `email`，并把这个邮箱当作显示名和快照键。

保存后的快照目录默认是：

```text
~/.local/share/codex-switch/profiles/
```

如果设置了 `XDG_DATA_HOME`，则会写到：

```text
$XDG_DATA_HOME/codex-switch/profiles/
```

每个已保存账号对应一个 JSON 快照文件，切换时会直接用所选快照覆盖 `~/.codex/auth.json`。

## 运行依赖

除了 Go，本项目在 Linux 构建时还需要：

### Debian / Ubuntu

```bash
sudo apt-get install gcc pkg-config libgtk-3-dev libayatana-appindicator3-dev
```

如果系统更老，只提供 `libappindicator3`，构建脚本会自动回退到 `legacy_appindicator`。

### Arch Linux

```bash
sudo pacman -S --needed base-devel pkgconf gtk3 libayatana-appindicator
```

建议再装上：

```bash
sudo pacman -S --needed xdg-utils
```

## 编译

```bash
./scripts/build.sh
```

成功后输出：

```text
dist/codex-switch
```

## 安装

用户级安装：

```bash
./scripts/install.sh
```

这会把程序装到：

```text
~/.local/bin/codex-switch
```

并生成桌面条目（会出现在程序启动器里）：

```text
~/.local/share/applications/codex-switch.desktop
```

安装完成后，可以直接在程序启动器里搜索 `Codex Switch` 启动。

如果你更习惯 Arch 的打包安装方式，仓库里已经带了 [PKGBUILD](/home/liufeng/Code/self/codex-switch/PKGBUILD) 和 [.SRCINFO](/home/liufeng/Code/self/codex-switch/.SRCINFO)。

本地直接打包安装：

```bash
makepkg -si
```

如果希望登录后自动启动：

```bash
./scripts/install.sh --autostart
```

如果要移除自动启动：

```bash
./scripts/install.sh --no-autostart
```

卸载用户级安装：

```bash
./scripts/uninstall.sh
```

如果连同已保存账号快照一起删除：

```bash
./scripts/uninstall.sh --purge-data
```

## 使用

先确保你已经登录过至少一个 Codex 账号，并且当前存在有效的：

```text
~/.codex/auth.json
```

启动程序后，在系统托盘菜单里可以看到：

- `保存当前账号`
- `刷新菜单`
- `切换到已保存账号`
- `删除已保存账号`
- `打开配置目录`
- `退出`

首次使用建议流程：

1. 登录第一个 Codex 账号。
2. 点击 `保存当前账号`。
3. 切换到另一个账号重新登录 Codex。
4. 再次点击 `保存当前账号`。
5. 之后就可以直接在托盘里按邮箱切换。

## 测试

运行：

```bash
go test ./...
```

当前测试覆盖了：

- 从 `auth.json` 解析邮箱
- 保存当前账号快照
- 列出已保存账号
- 切换账号
- 删除已保存账号

## 限制

- 当前目标环境是 Linux + Xorg。
- 托盘菜单是唯一交互入口，没有独立管理窗口。
- 邮箱依赖 `tokens.id_token` 中的 `email` claim；如果当前 `auth.json` 缺少这个字段，程序不会保存该配置。

## 图标来源

- 图形基于 OpenAI 2025 Blossom 标志。
- OpenAI 品牌使用规则见 OpenAI 官方品牌页。
- 本仓库内使用的是根据公开 SVG 生成的本地图标文件，分别用于 tray 和桌面启动器。

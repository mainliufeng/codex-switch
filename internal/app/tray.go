package app

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"codex-switch/internal/assets"
	"codex-switch/internal/profiles"
	"github.com/getlantern/systray"
)

type TrayApp struct {
	store *profiles.Store

	mu sync.Mutex

	currentItem *systray.MenuItem
	statusItem  *systray.MenuItem
	saveItem    *systray.MenuItem
	reloadItem  *systray.MenuItem
	switchRoot  *systray.MenuItem
	deleteRoot  *systray.MenuItem
	openDirItem *systray.MenuItem
	quitItem    *systray.MenuItem

	switchMenu *submenuController
	deleteMenu *submenuController
}

func NewTrayApp(store *profiles.Store) *TrayApp {
	return &TrayApp{store: store}
}

func (a *TrayApp) Run() {
	systray.Run(a.onReady, a.onExit)
}

func (a *TrayApp) onReady() {
	systray.SetIcon(assets.TrayIconPNG)
	systray.SetTitle("Codex")
	systray.SetTooltip("Codex 登录配置切换")

	a.currentItem = systray.AddMenuItem("当前: 检查中...", "当前 ~/.codex/auth.json 对应的邮箱")
	a.currentItem.Disable()

	a.statusItem = systray.AddMenuItem("状态: 启动中...", "最近一次操作结果")
	a.statusItem.Disable()

	systray.AddSeparator()

	a.saveItem = systray.AddMenuItem("保存当前账号", "把当前 ~/.codex/auth.json 按邮箱保存为快照")
	a.reloadItem = systray.AddMenuItem("刷新菜单", "重新扫描已保存账号和当前 auth.json")

	a.switchRoot = systray.AddMenuItem("切换到已保存账号", "选择某个已保存邮箱并覆盖 ~/.codex/auth.json")
	a.deleteRoot = systray.AddMenuItem("删除已保存账号", "删除本地保存的账号快照")
	a.switchMenu = newSubmenuController(
		systrayParent{item: a.switchRoot},
		a.bindSwitchAction,
		"暂无已保存账号",
		"先点击“保存当前账号”",
	)
	a.deleteMenu = newSubmenuController(
		systrayParent{item: a.deleteRoot},
		a.bindDeleteAction,
		"暂无已保存账号",
		"暂无可删除项",
	)

	systray.AddSeparator()

	a.openDirItem = systray.AddMenuItem("打开配置目录", "打开已保存账号所在目录")

	systray.AddSeparator()

	a.quitItem = systray.AddMenuItem("退出", "退出 Codex Switch")

	a.refreshMenus("已就绪")
	a.bindBaseActions()
}

func (a *TrayApp) onExit() {}

func (a *TrayApp) bindBaseActions() {
	go func() {
		for range a.saveItem.ClickedCh {
			profile, err := a.store.SaveCurrent()
			if err != nil {
				a.refreshMenus("保存失败: " + shortenError(err))
				continue
			}

			a.refreshMenus("已保存: " + profile.Email)
		}
	}()

	go func() {
		for range a.reloadItem.ClickedCh {
			a.refreshMenus("已刷新")
		}
	}()

	go func() {
		for range a.openDirItem.ClickedCh {
			if err := exec.Command("xdg-open", a.store.ProfilesDir).Start(); err != nil {
				a.refreshMenus("打开目录失败: " + shortenError(err))
				continue
			}

			a.refreshMenus("已打开配置目录")
		}
	}()

	go func() {
		for range a.quitItem.ClickedCh {
			systray.Quit()
			return
		}
	}()
}

func (a *TrayApp) refreshMenus(status string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	currentEmail, currentErr := a.store.CurrentEmail()
	if currentErr != nil {
		a.currentItem.SetTitle("当前: 未检测到有效账号")
		a.currentItem.SetTooltip(shortenError(currentErr))
	} else {
		a.currentItem.SetTitle("当前: " + currentEmail)
		a.currentItem.SetTooltip("当前 ~/.codex/auth.json 已识别为 " + currentEmail)
	}

	a.statusItem.SetTitle("状态: " + status)
	a.statusItem.SetTooltip(status)

	savedProfiles, err := a.store.List()
	if err != nil {
		a.switchRoot.SetTitle("切换到已保存账号")
		a.deleteRoot.SetTitle("删除已保存账号")
		a.switchMenu.Sync([]submenuEntry{{
			Key:      "__switch_error__",
			Title:    "读取失败",
			Tooltip:  shortenError(err),
			Disabled: true,
		}})
		a.deleteMenu.Sync([]submenuEntry{{
			Key:      "__delete_error__",
			Title:    "读取失败",
			Tooltip:  shortenError(err),
			Disabled: true,
		}})
		return
	}

	a.switchRoot.SetTitle(fmt.Sprintf("切换到已保存账号 (%d)", len(savedProfiles)))
	a.deleteRoot.SetTitle(fmt.Sprintf("删除已保存账号 (%d)", len(savedProfiles)))

	switchEntries := make([]submenuEntry, 0, len(savedProfiles))
	deleteEntries := make([]submenuEntry, 0, len(savedProfiles))

	for _, profile := range savedProfiles {
		switchTitle := profile.Email
		if currentErr == nil && profile.Email == currentEmail {
			switchTitle = profile.Email + " (当前)"
		}

		switchEntries = append(switchEntries, submenuEntry{
			Key:      profile.Email,
			Title:    switchTitle,
			Tooltip:  "切换到 " + profile.Email,
			Disabled: currentErr == nil && profile.Email == currentEmail,
		})
		deleteEntries = append(deleteEntries, submenuEntry{
			Key:      profile.Email,
			Title:    profile.Email,
			Tooltip:  "删除 " + profile.Email,
			Disabled: false,
		})
	}

	a.switchMenu.Sync(switchEntries)
	a.deleteMenu.Sync(deleteEntries)
}

func (a *TrayApp) bindSwitchAction(item clickableMenuItem, email string) {
	go func() {
		for range item.Clicked() {
			profile, err := a.store.Switch(email)
			if err != nil {
				a.refreshMenus("切换失败: " + shortenError(err))
				continue
			}

			a.refreshMenus("已切换到: " + profile.Email)
		}
	}()
}

func (a *TrayApp) bindDeleteAction(item clickableMenuItem, email string) {
	go func() {
		for range item.Clicked() {
			if err := a.store.Delete(email); err != nil {
				a.refreshMenus("删除失败: " + shortenError(err))
				continue
			}

			a.refreshMenus("已删除: " + email)
		}
	}()
}

func shortenError(err error) string {
	if err == nil {
		return ""
	}

	if errors.Is(err, exec.ErrNotFound) {
		return "系统缺少 xdg-open"
	}

	message := strings.TrimSpace(err.Error())
	if len(message) > 120 {
		return message[:117] + "..."
	}

	return message
}

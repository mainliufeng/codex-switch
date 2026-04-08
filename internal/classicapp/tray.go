package classicapp

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"codex-switch/internal/assets"
	"codex-switch/internal/core"
	"codex-switch/internal/profiles"
	"codex-switch/internal/usage"
	"github.com/getlantern/systray"
)

const autoRefreshInterval = 10 * time.Minute

type TrayApp struct {
	store   *profiles.Store
	service *core.Service

	mu sync.Mutex

	currentItem *systray.MenuItem
	statusItem  *systray.MenuItem
	statusText  string
	saveItem    *systray.MenuItem
	reloadItem  *systray.MenuItem
	switchRoot  *systray.MenuItem
	deleteRoot  *systray.MenuItem
	openDirItem *systray.MenuItem
	quitItem    *systray.MenuItem

	switchMenu *submenuController
	deleteMenu *submenuController
	refreshSeq uint64
}

type overviewLoadOptions struct {
	status       string
	updateStatus bool
}

func NewTrayApp(store *profiles.Store) *TrayApp {
	return &TrayApp{
		store:   store,
		service: core.NewService(store, usage.NewFetcher()),
	}
}

func (a *TrayApp) Run() {
	systray.Run(a.onReady, a.onExit)
}

func (a *TrayApp) onReady() {
	systray.SetIcon(assets.TrayIconPNG)
	systray.SetTitle("Codex Switch")
	systray.SetTooltip("Codex 登录配置切换")

	a.currentItem = systray.AddMenuItem("当前: 检查中...", "当前 ~/.codex/auth.json 对应的邮箱")
	a.currentItem.Disable()

	a.statusItem = systray.AddMenuItem("状态: 启动中...", "最近一次操作结果")
	a.statusItem.Disable()
	a.statusText = "启动中..."

	systray.AddSeparator()

	a.saveItem = systray.AddMenuItem("保存当前账号", "把当前 ~/.codex/auth.json 按邮箱保存为快照")
	a.reloadItem = systray.AddMenuItem("刷新菜单", "重新扫描已保存账号和当前 auth.json")

	systray.AddSeparator()

	a.switchRoot = systray.AddMenuItem("切换账号", "选择账号并查看对应用量")
	a.switchMenu = newSubmenuController(
		systrayParent{item: a.switchRoot},
		a.bindSwitchAction,
		"暂无已保存账号",
		"先点击“保存当前账号”",
	)

	systray.AddSeparator()

	a.deleteRoot = systray.AddMenuItem("删除已保存账号", "删除本地保存的账号快照")
	a.deleteMenu = newSubmenuController(
		systrayParent{item: a.deleteRoot},
		a.bindDeleteAction,
		"暂无可删除账号",
		"暂无可删除项",
	)

	systray.AddSeparator()

	a.openDirItem = systray.AddMenuItem("打开配置目录", "打开已保存账号所在目录")

	systray.AddSeparator()

	a.quitItem = systray.AddMenuItem("退出", "退出 Codex Switch")

	a.switchMenu.Sync(nil)
	a.deleteMenu.Sync(nil)
	a.bindBaseActions()

	go a.loadOverview("已就绪")
	go a.startAutoRefreshLoop()
}

func (a *TrayApp) onExit() {}

func (a *TrayApp) bindBaseActions() {
	go func() {
		for range a.saveItem.ClickedCh {
			profile, err := a.store.SaveCurrent()
			if err != nil {
				a.loadOverview("保存失败: " + shortenError(err))
				continue
			}

			a.loadOverview("已保存: " + profile.Email)
		}
	}()

	go func() {
		for range a.reloadItem.ClickedCh {
			a.loadOverview("已刷新")
		}
	}()

	go func() {
		for range a.openDirItem.ClickedCh {
			if err := exec.Command("xdg-open", a.store.ProfilesDir).Start(); err != nil {
				a.loadOverview("打开目录失败: " + shortenError(err))
				continue
			}

			a.loadOverview("已打开配置目录")
		}
	}()

	go func() {
		for range a.quitItem.ClickedCh {
			systray.Quit()
			return
		}
	}()
}

func (a *TrayApp) bindSwitchAction(item clickableMenuItem, email string) {
	go func() {
		for range item.Clicked() {
			profile, err := a.store.Switch(email)
			if err != nil {
				a.loadOverview("切换失败: " + shortenError(err))
				continue
			}

			a.loadOverview("已切换到: " + profile.Email)
		}
	}()
}

func (a *TrayApp) bindDeleteAction(item clickableMenuItem, email string) {
	go func() {
		for range item.Clicked() {
			if err := a.store.Delete(email); err != nil {
				a.loadOverview("删除失败: " + shortenError(err))
				continue
			}

			a.loadOverview("已删除: " + email)
		}
	}()
}

func (a *TrayApp) loadOverview(status string) {
	a.loadOverviewWithOptions(overviewLoadOptions{
		status:       status,
		updateStatus: true,
	})
}

func (a *TrayApp) loadOverviewWithOptions(options overviewLoadOptions) {
	seq := a.beginRefresh()
	currentStatus := a.currentStatusText()

	overview, err := a.service.OverviewWithoutUsage()
	if err != nil {
		a.mu.Lock()
		defer a.mu.Unlock()

		a.currentItem.SetTitle("当前: 未检测到有效账号")
		a.currentItem.SetTooltip(shortenError(err))
		statusText := resolveStatusText(currentStatus, options.status, options.updateStatus)
		a.statusItem.SetTitle("状态: " + statusText)
		a.statusItem.SetTooltip(statusText)
		a.statusText = statusText

		errorEntry := []submenuEntry{{
			Key:      "__error__",
			Title:    "读取失败",
			Tooltip:  shortenError(err),
			Disabled: true,
		}}
		a.switchRoot.SetTitle("切换账号")
		a.deleteRoot.SetTitle("删除已保存账号")
		a.switchMenu.Sync(errorEntry)
		a.deleteMenu.Sync(errorEntry)
		return
	}

	a.syncOverviewWithOptions(overview, overviewLoadOptions{
		status:       resolveStatusText(currentStatus, options.status+" · 用量同步中", options.updateStatus),
		updateStatus: options.updateStatus,
	})

	go func(refreshSeq uint64, options overviewLoadOptions) {
		fullOverview, err := a.service.Overview(a.requestContext())
		if err != nil {
			if a.isCurrentRefresh(refreshSeq) && options.updateStatus {
				a.setStatus(options.status + " · 用量同步失败")
			}
			return
		}

		if !a.isCurrentRefresh(refreshSeq) {
			return
		}

		a.syncOverviewWithOptions(fullOverview, overviewLoadOptions{
			status:       resolveStatusText(currentStatus, options.status, options.updateStatus),
			updateStatus: options.updateStatus,
		})
	}(seq, options)
}

func (a *TrayApp) beginRefresh() uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.refreshSeq++
	return a.refreshSeq
}

func (a *TrayApp) isCurrentRefresh(seq uint64) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.refreshSeq == seq
}

func (a *TrayApp) setStatus(status string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.statusItem.SetTitle("状态: " + status)
	a.statusItem.SetTooltip(status)
}

func (a *TrayApp) syncOverview(overview core.Overview, status string) {
	a.syncOverviewWithOptions(overview, overviewLoadOptions{
		status:       status,
		updateStatus: true,
	})
}

func (a *TrayApp) syncOverviewWithOptions(overview core.Overview, options overviewLoadOptions) {
	a.mu.Lock()
	defer a.mu.Unlock()

	currentTitle := "当前: 未检测到有效账号"
	currentTooltip := "当前 ~/.codex/auth.json 未识别到邮箱"
	if overview.CurrentEmail != "" {
		currentTitle = "当前: " + overview.CurrentEmail
		currentTooltip = "当前 ~/.codex/auth.json 已识别为 " + overview.CurrentEmail
	}
	a.currentItem.SetTitle(currentTitle)
	a.currentItem.SetTooltip(currentTooltip)

	if options.updateStatus {
		a.statusItem.SetTitle("状态: " + options.status)
		a.statusItem.SetTooltip(options.status)
		a.statusText = options.status
	}

	a.switchRoot.SetTitle(fmt.Sprintf("切换账号 (%d)", len(overview.Accounts)))

	deletableCount := 0
	for _, account := range overview.Accounts {
		if account.CanDelete {
			deletableCount++
		}
	}
	a.deleteRoot.SetTitle(fmt.Sprintf("删除已保存账号 (%d)", deletableCount))

	a.switchMenu.Sync(buildSwitchEntries(overview))
	a.deleteMenu.Sync(buildDeleteEntries(overview))
}

func buildSwitchEntries(overview core.Overview) []submenuEntry {
	return buildSwitchUsageEntries(overview)
}

func buildSwitchEntriesAt(overview core.Overview, now time.Time) []submenuEntry {
	return buildSwitchUsageEntriesAt(overview, now)
}

func buildDeleteEntries(overview core.Overview) []submenuEntry {
	entries := make([]submenuEntry, 0, len(overview.Accounts))
	for _, account := range overview.Accounts {
		if !account.CanDelete {
			continue
		}
		entries = append(entries, submenuEntry{
			Key:      account.Email,
			Title:    account.Email,
			Tooltip:  "删除 " + account.Email,
			Disabled: false,
		})
	}
	return entries
}

func (a *TrayApp) requestContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), 25*time.Second)
	return ctx
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

func (a *TrayApp) startAutoRefreshLoop() {
	ticker := time.NewTicker(autoRefreshInterval)
	defer ticker.Stop()

	for range ticker.C {
		a.refreshUsageSilently()
	}
}

func (a *TrayApp) refreshUsageSilently() {
	overview, err := a.service.Overview(a.requestContext())
	if err != nil {
		return
	}

	a.syncOverviewWithOptions(overview, overviewLoadOptions{
		updateStatus: false,
	})
}

func resolveStatusText(current string, next string, update bool) string {
	if update {
		return next
	}
	if strings.TrimSpace(current) != "" {
		return current
	}
	return next
}

func (a *TrayApp) currentStatusText() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.statusText
}

package classicapp

import (
	"fmt"
	"strings"
	"time"

	"codex-switch/internal/core"
)

const progressBarWidth = 10

const (
	fiveHourLabel = "５ｈ"
	weeklyLabel   = "周窗"
)

func buildSwitchUsageEntries(overview core.Overview) []submenuEntry {
	return buildSwitchUsageEntriesAt(overview, time.Now())
}

func buildSwitchUsageEntriesAt(overview core.Overview, now time.Time) []submenuEntry {
	entries := make([]submenuEntry, 0, len(overview.Accounts)*4)
	for idx, account := range overview.Accounts {
		label := "[可切换]"
		tooltip := "切换到 " + account.Email
		if account.Current {
			label = "[当前]"
			tooltip = "当前账号"
		}
		if account.UsageState == "ready" && account.Usage != nil {
			tooltip = tooltip + " · " + formatUsageMeta(account.Usage)
		}

		entries = append(entries, submenuEntry{
			Key:       account.Email + "/select",
			ActionKey: account.Email,
			Title:     fmt.Sprintf("%s %s", label, account.Email),
			Tooltip:   tooltip,
			Disabled:  !account.CanSwitch,
		})

		entries = append(entries, buildUsageStateEntries(account, now)...)

		if idx < len(overview.Accounts)-1 {
			entries = append(entries, submenuEntry{
				Key:      account.Email + "/separator",
				Title:    " ",
				Tooltip:  "",
				Disabled: true,
			})
		}
	}

	return entries
}

func buildUsageStateEntries(account core.Account, now time.Time) []submenuEntry {
	switch account.UsageState {
	case "ready":
		if account.Usage == nil {
			return buildLoadingUsageEntries(account.Email, "读取中...")
		}

		return []submenuEntry{
			{
				Key:      account.Email + "/five-hour",
				Title:    formatProgressLine(fiveHourLabel, account.Usage.FiveHourRemaining),
				Tooltip:  fmt.Sprintf("5h 剩余 %d%%", clampPercentage(account.Usage.FiveHourRemaining)),
				Disabled: true,
			},
			{
				Key:      account.Email + "/five-hour-reset",
				Title:    formatFiveHourResetLine(account.Usage.FiveHourResetAt, now),
				Tooltip:  formatFiveHourResetTooltip(account.Usage.FiveHourResetAt, now),
				Disabled: true,
			},
			{
				Key:      account.Email + "/weekly",
				Title:    formatProgressLine(weeklyLabel, account.Usage.WeeklyRemaining),
				Tooltip:  fmt.Sprintf("周剩余 %d%%", clampPercentage(account.Usage.WeeklyRemaining)),
				Disabled: true,
			},
			{
				Key:      account.Email + "/weekly-reset",
				Title:    formatWeeklyResetLine(account.Usage.WeeklyResetAt, now),
				Tooltip:  formatWeeklyResetTooltip(account.Usage.WeeklyResetAt, now),
				Disabled: true,
			},
		}
	case "error":
		errorText := strings.TrimSpace(account.UsageError)
		if errorText == "" {
			errorText = "未知错误"
		}
		return buildLoadingUsageEntries(account.Email, "暂不可用", errorText)
	default:
		return buildLoadingUsageEntries(account.Email, "读取中...")
	}
}

func buildLoadingUsageEntries(email string, value string, tooltipParts ...string) []submenuEntry {
	tooltip := value
	if len(tooltipParts) > 0 && strings.TrimSpace(tooltipParts[0]) != "" {
		tooltip = tooltipParts[0]
	}

	return []submenuEntry{
		{
			Key:      email + "/five-hour",
			Title:    fmt.Sprintf("%s  %s", fiveHourLabel, value),
			Tooltip:  tooltip,
			Disabled: true,
		},
		{
			Key:      email + "/five-hour-reset",
			Title:    "    " + value,
			Tooltip:  tooltip,
			Disabled: true,
		},
		{
			Key:      email + "/weekly",
			Title:    fmt.Sprintf("%s  %s", weeklyLabel, value),
			Tooltip:  tooltip,
			Disabled: true,
		},
		{
			Key:      email + "/weekly-reset",
			Title:    "    " + value,
			Tooltip:  tooltip,
			Disabled: true,
		},
	}
}

func formatProgressBar(remaining int) string {
	remaining = clampPercentage(remaining)
	filled := int(float64(remaining) / 100 * progressBarWidth)
	if remaining > 0 && filled == 0 {
		filled = 1
	}
	if remaining == 100 {
		filled = progressBarWidth
	}

	return strings.Repeat("█", filled) + strings.Repeat("░", progressBarWidth-filled)
}

func formatProgressLine(label string, remaining int) string {
	remaining = clampPercentage(remaining)
	return fmt.Sprintf("%s  %s  %d%%", label, formatProgressBar(remaining), remaining)
}

func formatUsageMeta(usage *core.UsageState) string {
	if usage == nil {
		return "等待同步"
	}

	plan := strings.TrimSpace(usage.PlanType)
	if plan == "" {
		plan = "Unknown"
	} else {
		plan = strings.ToUpper(plan[:1]) + plan[1:]
	}

	updatedAt := "未知时间"
	if !usage.UpdatedAt.IsZero() {
		updatedAt = usage.UpdatedAt.Format("15:04")
	}

	return fmt.Sprintf("%s · %s 更新", plan, updatedAt)
}

func clampPercentage(value int) int {
	switch {
	case value < 0:
		return 0
	case value > 100:
		return 100
	default:
		return value
	}
}

func shortenDisplay(value string, maxLen int) string {
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	if maxLen <= 3 {
		return value[:maxLen]
	}
	return value[:maxLen-3] + "..."
}

func formatFiveHourResetLine(resetAt time.Time, now time.Time) string {
	return "    " + formatFiveHourResetValue(resetAt, now)
}

func formatWeeklyResetLine(resetAt time.Time, now time.Time) string {
	return "    " + formatWeeklyResetValue(resetAt, now)
}

func formatFiveHourResetTooltip(resetAt time.Time, now time.Time) string {
	return "5h 窗口将在 " + formatFiveHourResetValue(resetAt, now)
}

func formatWeeklyResetTooltip(resetAt time.Time, now time.Time) string {
	return "周窗口将在 " + formatWeeklyResetValue(resetAt, now)
}

func formatFiveHourResetValue(resetAt time.Time, now time.Time) string {
	if resetAt.IsZero() {
		return "刷新时间未知"
	}

	if now.IsZero() {
		now = time.Now()
	}

	resetAt = resetAt.In(now.Location())
	if !resetAt.After(now) {
		return "即将刷新"
	}

	return formatCompactDuration(resetAt.Sub(now)) + " 后刷新"
}

func formatWeeklyResetValue(resetAt time.Time, now time.Time) string {
	if resetAt.IsZero() {
		return "刷新时间未知"
	}

	if now.IsZero() {
		now = time.Now()
	}

	resetAt = resetAt.In(now.Location())
	if sameDay(now, resetAt) {
		return "今天 " + resetAt.Format("15:04") + " 刷新"
	}

	nowYear, nowWeek := now.ISOWeek()
	resetYear, resetWeek := resetAt.ISOWeek()
	if nowYear == resetYear && nowWeek == resetWeek {
		return weekdayLabel(resetAt.Weekday()) + " " + resetAt.Format("15:04") + " 刷新"
	}

	return resetAt.Format("1/2 15:04") + " 刷新"
}

func formatCompactDuration(duration time.Duration) string {
	if duration <= 0 {
		return "0m"
	}

	duration = duration.Round(time.Minute)
	if duration < time.Minute {
		return "1m"
	}

	totalMinutes := int(duration / time.Minute)
	days := totalMinutes / (24 * 60)
	totalMinutes -= days * 24 * 60
	hours := totalMinutes / 60
	minutes := totalMinutes % 60

	switch {
	case days > 0 && hours > 0:
		return fmt.Sprintf("%dd%dh", days, hours)
	case days > 0:
		return fmt.Sprintf("%dd", days)
	case hours > 0 && minutes > 0:
		return fmt.Sprintf("%dh%dm", hours, minutes)
	case hours > 0:
		return fmt.Sprintf("%dh", hours)
	default:
		return fmt.Sprintf("%dm", minutes)
	}
}

func sameDay(a time.Time, b time.Time) bool {
	a = a.In(b.Location())
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

func weekdayLabel(day time.Weekday) string {
	switch day {
	case time.Monday:
		return "周一"
	case time.Tuesday:
		return "周二"
	case time.Wednesday:
		return "周三"
	case time.Thursday:
		return "周四"
	case time.Friday:
		return "周五"
	case time.Saturday:
		return "周六"
	default:
		return "周日"
	}
}

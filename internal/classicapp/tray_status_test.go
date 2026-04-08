package classicapp

import "testing"

func TestResolveStatusText(t *testing.T) {
	if got := resolveStatusText("已打开配置目录", "已刷新", true); got != "已刷新" {
		t.Fatalf("expected explicit refresh status, got %q", got)
	}

	if got := resolveStatusText("已打开配置目录", "自动刷新", false); got != "已打开配置目录" {
		t.Fatalf("expected silent refresh to preserve previous status, got %q", got)
	}

	if got := resolveStatusText("", "启动中", false); got != "启动中" {
		t.Fatalf("expected fallback to incoming status when no previous status exists, got %q", got)
	}
}

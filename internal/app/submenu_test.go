package app

import "testing"

func TestSubmenuControllerReusesExistingItemsAcrossRefreshes(t *testing.T) {
	parent := &fakeSubmenuParent{}
	controller := newSubmenuController(parent, func(item clickableMenuItem, key string) {}, "empty", "empty tooltip")

	controller.Sync([]submenuEntry{
		{Key: "alice@example.com", Title: "alice@example.com", Tooltip: "切换到 alice@example.com"},
		{Key: "bob@example.com", Title: "bob@example.com", Tooltip: "切换到 bob@example.com"},
	})

	if parent.created != 2 {
		t.Fatalf("expected 2 items created on first sync, got %d", parent.created)
	}

	controller.Sync([]submenuEntry{
		{Key: "alice@example.com", Title: "alice@example.com (当前)", Tooltip: "切换到 alice@example.com", Disabled: true},
		{Key: "bob@example.com", Title: "bob@example.com", Tooltip: "切换到 bob@example.com"},
	})

	if parent.created != 2 {
		t.Fatalf("expected controller to reuse items, got %d total creations", parent.created)
	}

	alice := parent.items["alice@example.com"]
	if alice == nil {
		t.Fatal("expected alice item to exist")
	}
	if alice.title != "alice@example.com (当前)" {
		t.Fatalf("expected alice title to update, got %q", alice.title)
	}
	if !alice.disabled {
		t.Fatal("expected alice item to be disabled for current profile")
	}
}

func TestSubmenuControllerHidesUnusedItemsAndReusesPlaceholder(t *testing.T) {
	parent := &fakeSubmenuParent{}
	controller := newSubmenuController(parent, func(item clickableMenuItem, key string) {}, "暂无已保存账号", "先点击保存")

	controller.Sync(nil)
	if parent.created != 1 {
		t.Fatalf("expected one placeholder item, got %d creations", parent.created)
	}

	placeholder := parent.lastCreated
	if !placeholder.disabled {
		t.Fatal("expected placeholder to be disabled")
	}
	if !placeholder.visible {
		t.Fatal("expected placeholder to be visible")
	}

	controller.Sync([]submenuEntry{
		{Key: "alice@example.com", Title: "alice@example.com", Tooltip: "切换到 alice@example.com"},
	})

	if placeholder.visible {
		t.Fatal("expected placeholder to be hidden once profiles exist")
	}
	if parent.created != 2 {
		t.Fatalf("expected one real item to be created after placeholder, got %d creations", parent.created)
	}

	controller.Sync(nil)
	if parent.created != 2 {
		t.Fatalf("expected placeholder reuse without extra creations, got %d creations", parent.created)
	}
	if !placeholder.visible {
		t.Fatal("expected placeholder to be shown again")
	}
}

type fakeSubmenuParent struct {
	items       map[string]*fakeMenuItem
	created     int
	lastCreated *fakeMenuItem
}

func (p *fakeSubmenuParent) AddSubMenuItem(title string, tooltip string) clickableMenuItem {
	if p.items == nil {
		p.items = make(map[string]*fakeMenuItem)
	}

	item := &fakeMenuItem{
		title:   title,
		tooltip: tooltip,
		visible: true,
		clicked: make(chan struct{}),
	}

	p.items[title] = item
	p.created++
	p.lastCreated = item
	return item
}

type fakeMenuItem struct {
	title    string
	tooltip  string
	visible  bool
	disabled bool
	clicked  chan struct{}
}

func (i *fakeMenuItem) SetTitle(title string)     { i.title = title }
func (i *fakeMenuItem) SetTooltip(tooltip string) { i.tooltip = tooltip }
func (i *fakeMenuItem) Enable()                   { i.disabled = false }
func (i *fakeMenuItem) Disable()                  { i.disabled = true }
func (i *fakeMenuItem) Hide()                     { i.visible = false }
func (i *fakeMenuItem) Show()                     { i.visible = true }
func (i *fakeMenuItem) Clicked() <-chan struct{}  { return i.clicked }

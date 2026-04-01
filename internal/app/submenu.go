package app

import "github.com/getlantern/systray"

type submenuEntry struct {
	Key      string
	Title    string
	Tooltip  string
	Disabled bool
}

type clickableMenuItem interface {
	SetTitle(string)
	SetTooltip(string)
	Enable()
	Disable()
	Hide()
	Show()
	Clicked() <-chan struct{}
}

type submenuParent interface {
	AddSubMenuItem(title string, tooltip string) clickableMenuItem
}

type submenuBinder func(item clickableMenuItem, key string)

type submenuController struct {
	parent            submenuParent
	bind              submenuBinder
	emptyTitle        string
	emptyTooltip      string
	items             map[string]clickableMenuItem
	placeholder       clickableMenuItem
	placeholderInited bool
}

func newSubmenuController(parent submenuParent, bind submenuBinder, emptyTitle string, emptyTooltip string) *submenuController {
	return &submenuController{
		parent:       parent,
		bind:         bind,
		emptyTitle:   emptyTitle,
		emptyTooltip: emptyTooltip,
		items:        make(map[string]clickableMenuItem),
	}
}

func (c *submenuController) Sync(entries []submenuEntry) {
	used := make(map[string]struct{}, len(entries))

	if len(entries) == 0 {
		c.showPlaceholder()
	} else {
		c.hidePlaceholder()
	}

	for _, entry := range entries {
		item, ok := c.items[entry.Key]
		if !ok {
			item = c.parent.AddSubMenuItem(entry.Title, entry.Tooltip)
			c.items[entry.Key] = item
			if c.bind != nil {
				c.bind(item, entry.Key)
			}
		}

		item.SetTitle(entry.Title)
		item.SetTooltip(entry.Tooltip)
		if entry.Disabled {
			item.Disable()
		} else {
			item.Enable()
		}
		item.Show()
		used[entry.Key] = struct{}{}
	}

	for key, item := range c.items {
		if _, ok := used[key]; ok {
			continue
		}
		item.Hide()
	}
}

func (c *submenuController) showPlaceholder() {
	if !c.placeholderInited {
		c.placeholder = c.parent.AddSubMenuItem(c.emptyTitle, c.emptyTooltip)
		c.placeholderInited = true
	}

	c.placeholder.SetTitle(c.emptyTitle)
	c.placeholder.SetTooltip(c.emptyTooltip)
	c.placeholder.Disable()
	c.placeholder.Show()
}

func (c *submenuController) hidePlaceholder() {
	if c.placeholderInited {
		c.placeholder.Hide()
	}
}

type systrayParent struct {
	item *systray.MenuItem
}

func (p systrayParent) AddSubMenuItem(title string, tooltip string) clickableMenuItem {
	return systrayMenuItem{item: p.item.AddSubMenuItem(title, tooltip)}
}

type systrayMenuItem struct {
	item *systray.MenuItem
}

func (i systrayMenuItem) SetTitle(title string)     { i.item.SetTitle(title) }
func (i systrayMenuItem) SetTooltip(tooltip string) { i.item.SetTooltip(tooltip) }
func (i systrayMenuItem) Enable()                   { i.item.Enable() }
func (i systrayMenuItem) Disable()                  { i.item.Disable() }
func (i systrayMenuItem) Hide()                     { i.item.Hide() }
func (i systrayMenuItem) Show()                     { i.item.Show() }
func (i systrayMenuItem) Clicked() <-chan struct{}  { return i.item.ClickedCh }

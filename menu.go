package main

import (
	"log"
	"strings"

	"github.com/JoelOtter/termloop"
)

const titleText = `  ,-----.,--.                  ,--. 
 '  .--./` + "`" + `--',--,--,--. ,---.  ` + "`" + `--' 
 |  |    ,--.|        || .-. | ,--. 
 '  '--'\|  ||  |  |  |' '-' ' |  | 
  ` + "`" + `-----'` + "`" + `--'` + "`" + `--` + "`" + `--` + "`" + `--' ` + "`" + `---'.-'  / 
                             '---'  `

var menuChoices = []string{
	"Komencu ludon",
	"Admaru vin mem",
	"Konfiguru opciojn",
}

// CrunchMenu provides the main menu for a CrunchApp.
type CrunchMenu struct {
	menu  *simpleMenu
	level *termloop.BaseLevel
}

// NewCrunchMenu creates a new menu to drive the CrunchApp.  It's recommended
// that the menu never be destroyed, but simple removed from any Entities lists
// to avoid it being drawn.
func NewCrunchMenu() *CrunchMenu {
	m := &CrunchMenu{}
	fg := termloop.ColorWhite
	bg := termloop.ColorBlack

	m.level = termloop.NewBaseLevel(termloop.Cell{
		Fg: fg,
		Bg: bg,
	})

	m.menu = newSimpleMenu(40, 9, fg, bg, menuChoices)
	m.level.AddEntity(m.menu)

	titleLines := strings.Split(titleText, "\n")
	title := termloop.NewEntity(2, 1, 50, 8)
	m.level.AddEntity(title)
	for i := range titleLines {
		log.Print(titleLines[i])
		for j := range titleLines[i] {
			title.SetCell(j, i, &termloop.Cell{
				Fg: fg,
				Bg: bg,
				Ch: rune(titleLines[i][j]),
			})
		}
	}

	m.menu.SetSelection(0, true)

	return m
}

// GetSelection returns the currently selected menu item.
func (m *CrunchMenu) GetSelection() (int, string) {
	return m.menu.GetSelection()
}

// Draw implements termloop.Drawable
func (m *CrunchMenu) Draw(screen *termloop.Screen) {
	m.level.Draw(screen)
}

// Tick implements termloop.Drawable
func (m *CrunchMenu) Tick(event termloop.Event) {
	if event.Type == termloop.EventKey { // Is it a keyboard event?
		switch event.Ch { // If so, switch on the pressed key.
		case 'k':
			m.menu.SetSelection(-1, false)
		case 'j':
			m.menu.SetSelection(1, false)
		}
	}

}

type simpleMenu struct {
	x     int
	y     int
	sel   int
	texts []string
	items []*simpleMenuItem
}

func newSimpleMenu(x, y int, fg, bg termloop.Attr, choices []string) *simpleMenu {
	m := &simpleMenu{
		x:     x,
		y:     y,
		sel:   -1,
		texts: choices,
	}

	m.items = make([]*simpleMenuItem, len(choices))
	for i := range choices {
		m.items[i] = newSimpleMenuItem(x, y+i, choices[i], fg, bg)
	}

	return m
}

func (m *simpleMenu) GetSelection() (int, string) {
	if m.sel < 0 {
		return -1, ""
	}
	return m.sel, m.texts[m.sel]
}

func (m *simpleMenu) SetSelection(i int, abs bool) {
	if abs && (i >= len(m.items) || i < 0) {
		panic("absolute menu index out of range")

	}

	if m.sel < 0 {
		if abs {
			m.sel = i
		} else {
			m.sel = (m.sel + i) % len(m.items)
		}

		m.items[m.sel].SetSelected(true)

		return
	}

	m.items[m.sel].SetSelected(false)
	m.sel = (m.sel + i) % len(m.items)
	if m.sel < 0 {
		m.sel = 0
	} else if m.sel >= len(m.items) {
		m.sel = len(m.items) - 1
	}
	m.items[m.sel].SetSelected(true)
}

func (m *simpleMenu) Draw(screen *termloop.Screen) {
	for i := range m.items {
		m.items[i].Draw(screen)
	}
}

func (m *simpleMenu) Tick(event termloop.Event) {
	for i := range m.items {
		m.items[i].Tick(event)
	}
}

type simpleMenuItem struct {
	x        int
	y        int
	fg       termloop.Attr
	bg       termloop.Attr
	selected bool
	arrow    *termloop.Text
	text     *termloop.Text
}

func newSimpleMenuItem(x, y int, text string, fg, bg termloop.Attr) *simpleMenuItem {
	item := &simpleMenuItem{
		x:  x,
		y:  y,
		fg: fg,
		bg: bg,
	}

	item.arrow = termloop.NewText(x, y, " ", fg, bg)
	item.text = termloop.NewText(x+1, y, text, fg, bg)

	return item
}

func (item *simpleMenuItem) SetSelected(isSelected bool) {
	if isSelected {
		item.selected = true
		item.arrow.SetText("»")
		item.text.SetColor(item.fg|termloop.AttrUnderline, item.bg)
	} else {
		item.selected = false
		item.arrow.SetText(" ")
		item.text.SetColor(item.fg, item.bg)
	}
}

func (item *simpleMenuItem) Draw(screen *termloop.Screen) {
	item.arrow.Draw(screen)
	item.text.Draw(screen)
}

func (item *simpleMenuItem) Tick(event termloop.Event) {
	item.arrow.Tick(event)
	item.text.Tick(event)
}

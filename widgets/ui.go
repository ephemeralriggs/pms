package widgets

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ambientsound/pms/console"
	"github.com/ambientsound/pms/index"
	"github.com/ambientsound/pms/input/parser"
	"github.com/ambientsound/pms/options"
	"github.com/ambientsound/pms/songlist"
	"github.com/ambientsound/pms/version"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
)

type StyleMap map[string]tcell.Style

type UI struct {
	// UI elements
	App    *views.Application
	Layout *views.BoxLayout

	Topbar        *views.TextBar
	Playbar       *PlaybarWidget
	Columnheaders *ColumnheadersWidget
	Multibar      *MultibarWidget
	Songlist      *SonglistWidget

	// Input events
	EventInputCommand chan string
	EventKeyInput     chan parser.KeyEvent

	// Data resources
	Index           *index.Index
	options         *options.Options
	currentSonglist songlist.Songlist
	songlists       []songlist.Songlist
	songlistIndex   int
	searchResult    songlist.Songlist

	// TCell
	view views.View
	widget
	views.WidgetWatchers
}

func NewUI(opts *options.Options) *UI {
	ui := &UI{}

	ui.EventInputCommand = make(chan string, 16)
	ui.EventKeyInput = make(chan parser.KeyEvent, 16)
	ui.songlists = make([]songlist.Songlist, 0)

	ui.App = &views.Application{}
	ui.options = opts

	ui.Topbar = views.NewTextBar()
	ui.Playbar = NewPlaybarWidget()
	ui.Columnheaders = NewColumnheadersWidget()
	ui.Multibar = NewMultibarWidget(ui.EventKeyInput)
	ui.Songlist = NewSonglistWidget()

	ui.Multibar.Watch(ui)
	ui.Songlist.Watch(ui)
	ui.Playbar.Watch(ui)

	styles := StyleMap{
		"album":         tcell.StyleDefault.Foreground(tcell.ColorTeal),
		"artist":        tcell.StyleDefault.Foreground(tcell.ColorYellow),
		"commandText":   tcell.StyleDefault,
		"currentSong":   tcell.StyleDefault.Background(tcell.ColorYellow).Foreground(tcell.ColorBlack),
		"cursor":        tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack),
		"date":          tcell.StyleDefault.Foreground(tcell.ColorGreen),
		"elapsed":       tcell.StyleDefault.Foreground(tcell.ColorGreen),
		"errorText":     tcell.StyleDefault.Background(tcell.ColorRed).Foreground(tcell.ColorWhite).Bold(true),
		"header":        tcell.StyleDefault.Foreground(tcell.ColorGreen).Bold(true),
		"noCurrentSong": tcell.StyleDefault.Foreground(tcell.ColorRed),
		"readout":       tcell.StyleDefault,
		"searchText":    tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true),
		"sequenceText":  tcell.StyleDefault.Foreground(tcell.ColorTeal),
		"statusbar":     tcell.StyleDefault,
		"switches":      tcell.StyleDefault.Foreground(tcell.ColorTeal),
		"time":          tcell.StyleDefault.Foreground(tcell.ColorDarkMagenta),
		"title":         tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true),
		"topbar":        tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true),
		"track":         tcell.StyleDefault.Foreground(tcell.ColorGreen),
		"volume":        tcell.StyleDefault.Foreground(tcell.ColorGreen),
		"year":          tcell.StyleDefault.Foreground(tcell.ColorGreen),
	}

	// Styles for widgets that don't have their own class yet.
	ui.SetStyleMap(styles)
	ui.Topbar.SetStyle(ui.Style("topbar"))
	ui.Topbar.SetLeft(version.ShortName(), ui.Style("topbar"))
	ui.Topbar.SetRight(version.Version(), ui.Style("topbar"))

	ui.Columnheaders.SetStyleMap(styles)
	ui.Playbar.SetStyleMap(styles)
	ui.Songlist.SetStyleMap(styles)
	ui.Multibar.SetStyleMap(styles)

	ui.CreateLayout()
	ui.App.SetRootWidget(ui)

	return ui
}

func (ui *UI) CreateLayout() {
	ui.Layout = views.NewBoxLayout(views.Vertical)
	ui.Layout.AddWidget(ui.Topbar, 0)
	ui.Layout.AddWidget(ui.Playbar, 0)
	ui.Layout.AddWidget(ui.Columnheaders, 0)
	ui.Layout.AddWidget(ui.Songlist, 2)
	ui.Layout.AddWidget(ui.Multibar, 0)
	ui.Layout.SetView(ui.view)
}

func (ui *UI) SetIndex(i *index.Index) {
	ui.Index = i
}

func (ui *UI) AddSonglist(s songlist.Songlist) {
	ui.songlists = append(ui.songlists, s)
	console.Log("Songlist UI: added songlist index %d of type %T at address %p", len(ui.songlists)-1, s, s)
}

// ReplaceSonglist replaces an existing songlist with its new version. Checking
// is done on a type-level, so only the queue and library will be replaced.
func (ui *UI) ReplaceSonglist(s songlist.Songlist) {
	for i := range ui.songlists {
		if reflect.TypeOf(ui.songlists[i]) != reflect.TypeOf(s) {
			continue
		}
		console.Log("Songlist UI: replacing songlist of type %T at %p with new list at %p", s, ui.songlists[i], s)
		console.Log("Songlist UI: comparing %p %p", ui.songlists[i], ui.currentSonglist)
		active := ui.songlists[i] == ui.currentSonglist
		ui.songlists[i] = s
		if active {
			console.Log("Songlist UI: replaced songlist is currently active, switching to new songlist.")
			ui.SetSonglist(s)
		}
		return
	}
	console.Log("Songlist UI: adding songlist of type %T at address %p since no similar exists", s, s)
	ui.AddSonglist(s)
}

func (ui *UI) SetSonglist(s songlist.Songlist) {
	console.Log("SetSonglist(%T %p)", s, s)
	ui.songlistIndex = -1
	for i, stored := range ui.songlists {
		if stored == s {
			ui.SetSonglistIndex(i)
			return
		}
	}
	ui.activateSonglist(s)
}

// FIXME: move functionality into ui.Songlist
func (ui *UI) activateSonglist(s songlist.Songlist) {
	console.Log("activateSonglist(%T %p)", s, s)
	ui.currentSonglist = s
	ui.Songlist.SetSonglist(s)
	ui.Songlist.SetColumns(strings.Split(ui.options.StringValue("columns"), ","))
}

func (ui *UI) SonglistIndex() int {
	return ui.songlistIndex
}

func (ui *UI) ValidSonglistIndex(i int) bool {
	return i >= 0 && i < ui.SonglistsLen()
}

func (ui *UI) SetSonglistIndex(i int) error {
	console.Log("SetSonglistIndex(%d)", i)
	if !ui.ValidSonglistIndex(i) {
		return fmt.Errorf("Index %d is out of bounds (try between 1 and %d)", i+1, ui.SonglistsLen())
	}
	ui.songlistIndex = i
	ui.activateSonglist(ui.songlists[ui.songlistIndex])
	return nil
}

func (ui *UI) Songlists() []songlist.Songlist {
	return ui.songlists
}

func (ui *UI) SonglistsLen() int {
	return len(ui.songlists)
}

func (ui *UI) Start() {
	ui.App.Start()
}

func (ui *UI) Wait() error {
	return ui.App.Wait()
}

func (ui *UI) Quit() {
	ui.App.Quit()
}

func (ui *UI) Draw() {
	ui.Layout.Draw()
}

func (ui *UI) Resize() {
	ui.CreateLayout()
	ui.Layout.Resize()
	ui.PostEventWidgetResize(ui)
}

func (ui *UI) SetView(v views.View) {
	ui.view = v
	ui.Layout.SetView(v)
}

func (ui *UI) Size() (int, int) {
	return ui.view.Size()
}

func (ui *UI) Title() string {
	var index string
	if ui.songlistIndex >= 0 {
		index = fmt.Sprintf("%d", ui.songlistIndex+1)
	} else {
		index = "..."
	}
	return fmt.Sprintf("[%s/%d] %s", index, ui.SonglistsLen(), ui.Songlist.Name())
}

func (ui *UI) HandleEvent(ev tcell.Event) bool {
	switch ev.(type) {

	case *EventListChanged:
		ui.Topbar.SetCenter(ui.Title(), ui.Style("title"))
		ui.Columnheaders.SetColumns(ui.Songlist.Columns())
		ui.App.Update()
		return true

	case *EventInputChanged:
		term := ui.Multibar.RuneString()
		mode := ui.Multibar.Mode()
		switch mode {
		case MultibarModeSearch:
			ui.runIndexSearch(term)
		}
		return true

	case *EventInputFinished:
		term := ui.Multibar.RuneString()
		mode := ui.Multibar.Mode()
		switch mode {
		case MultibarModeInput:
			ui.EventInputCommand <- term
		case MultibarModeSearch:
			ui.AddSonglist(ui.searchResult)
			ui.SetSonglist(ui.searchResult)
		}
		ui.Multibar.SetMode(MultibarModeNormal)
		return true

	case *EventScroll:
		ui.refreshPositionReadout()
		return true
	}

	if ui.Layout.HandleEvent(ev) {
		return true
	}

	return false
}

func (ui *UI) refreshPositionReadout() {
	str := ui.Songlist.PositionReadout()
	ui.Multibar.SetRight(str, ui.Style("readout"))
}

func (ui *UI) runIndexSearch(term string) {
	var err error

	if ui.Index == nil {
		return
	}
	if len(term) == 1 {
		return
	}

	ui.searchResult, err = ui.Index.Search(term)

	if err == nil {
		ui.Songlist.SetCursor(0)
		ui.SetSonglist(ui.searchResult)
		return
	}
}

package main

import (
	"os"
	"os/exec"
	"time"

	"github.com/DexterLB/mpvipc"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

type Player struct {
	WinY     float64 // current cursor position (to determine if controls should be hidden)
	Win      *gtk.Window
	App      *gtk.Application
	Builder  *gtk.Builder
	MpvArea  *gtk.DrawingArea
	Controls *gtk.Revealer
	Conn     *mpvipc.Connection
	Current  Media
	Timer    *time.Timer
	Mpv      *exec.Cmd
	SeekBusy bool
}

type Media struct {
	Duration     float64
	LengthParsed string
}

func main() {
	player := new(Player)
	app, err := gtk.ApplicationNew("io.github.pizza61.jcplayer", glib.APPLICATION_FLAGS_NONE)
	Nerr(err)

	player.App = app

	player.App.Connect("activate", func() { player.windowSetup() })

	os.Exit(player.run(os.Args))
}

func (player *Player) run(args []string) int {
	c := player.App.Run(os.Args)
	player.Mpv.Process.Kill()
	return c
}

func (player *Player) windowSetup() {
	// BUILDER
	builder, err := gtk.BuilderNewFromFile("jc.glade")
	Nerr(err)

	player.Builder = builder

	// WINDOW
	winO, err := player.Builder.GetObject("mainWindow")
	Nerr(err)

	win := winO.(*gtk.Window)

	player.Win = win

	win.SetIconFromFile("icon-128.png")
	win.SetSizeRequest(720, 480)
	win.ShowAll()

	// DRAWINGAREA (mpv)
	mpvO, err := builder.GetObject("mpv")
	Nerr(err)

	mpv := mpvO.(*gtk.DrawingArea)
	mpv.AddEvents(4)

	player.MpvArea = mpv

	// CONTROLS
	controlsO, err := builder.GetObject("controls")
	controls := controlsO.(*gtk.Revealer)

	player.Controls = controls

	// EVENTS
	player.SetupEvents()

	// MPV
	player.SetupMpv()

	player.App.AddWindow(win)

	player.SetupControls()
}

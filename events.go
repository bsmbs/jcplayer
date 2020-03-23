package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/gotk3/gotk3/glib"

	"github.com/DexterLB/mpvipc"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

func (player *Player) SetupEvents() {
	// CURSOR POSITION
	player.Win.AddEvents(4)

	player.Win.Connect("event", func(wind *gtk.Window, event *gdk.Event) {
		ev := gdk.EventMotionNewFromEvent(event)
		if ev.Type() == 3 {
			_, y := ev.MotionVal()
			player.WinY = y
		}

	})

	// CURSOR HIDE
	mpvGdk, err := player.MpvArea.GetWindow()
	sc, err := player.MpvArea.GetScreen()
	ds, err := sc.GetDisplay()
	Nerr(err)

	cursorNone, err := gdk.CursorNewFromName(ds, "none")
	cursorNormal, err := gdk.CursorNewFromName(ds, "default")

	// CONTROLS HIDE
	timer := time.NewTimer(800 * time.Millisecond)

	poptObj, err := player.Builder.GetObject("tracks")
	Nerr(err)

	popt := poptObj.(*gtk.PopoverMenu)

	player.MpvArea.Connect("event", func(da *gtk.DrawingArea, event *gdk.Event) {
		if gdk.EventKeyNewFromEvent(event).Type() == 3 { // Cursor move

			// Show controls and cursor

			go func() {
				player.Controls.SetRevealChild(true)
				mpvGdk.SetCursor(cursorNormal)
				timer.Reset(800 * time.Millisecond)

				<-timer.C // Wait .8s

				controlsHeight := player.Controls.GetAllocation().GetHeight() // Controls height
				winHeight := player.Win.GetAllocatedHeight() - controlsHeight // Window height
				pp := winHeight - controlsHeight                              // Window height - Controls height

				if int(player.WinY) > pp || int(player.WinY) < controlsHeight || popt.IsVisible() { // Cursor is on controls, don't hide (reset)

					timer.Reset(800 * time.Millisecond)
				} else { // Cursor is outside controls, can be hidden
					player.Controls.SetRevealChild(false)
					mpvGdk.SetCursor(cursorNone)
					//timer.Stop()
				}
			}()
		}
	})
}

func (player *Player) SetupMpv() {
	mpvGdk, err := player.MpvArea.GetWindow()
	Nerr(err)

	xid := strconv.Itoa(int(mpvGdk.GetXID()))
	pid := strconv.Itoa(os.Getpid())

	// Start mpv
	player.Mpv = exec.Command("mpv", "--wid="+xid, "--input-vo-keyboard=no", "--force-window", "--no-input-cursor", "--input-ipc-server=/tmp/jc"+pid, "--idle")
	player.Mpv.Start()
	player.MpvArea.Hide()

	// IPC ready check
	ready := make(chan bool)

	go func() {
		for {
			_, err := os.Stat("/tmp/jc" + pid)
			if !os.IsNotExist(err) {
				fmt.Println("/tmp/jc" + pid)
				ready <- true
				return
			}
		}
	}()

	<-ready // IPC is ready!

	player.Conn = mpvipc.NewConnection("/tmp/jc" + pid)
	err = player.Conn.Open()
	Nerr(err)

	// Connection opened

}

func (player *Player) SetupControls() {
	ticker := time.NewTicker(200 * time.Millisecond)

	// Images
	controlPause := player.getImage("controlPause")
	controlPlay := player.getImage("controlPlay")

	// Play
	play := player.getButton("controlsPlay")
	play.SetImage(controlPause)

	play.Connect("clicked", func() {
		playing, err := player.PausePlay()
		if err != nil {
			return
		}

		if playing {
			play.SetImage(controlPause)
		} else {
			play.SetImage(controlPlay)
		}
	})

	// Prev/next
	prev := player.getButton("controlsPrev")
	next := player.getButton("controlsNext")

	prev.Connect("clicked", func() {
		err := player.Seek(-10)
		if err != nil {
			return
		}
	})

	next.Connect("clicked", func() {
		err := player.Seek(10)
		if err != nil {
			return
		}
	})

	// Window keybinds
	player.Win.Connect("key-press-event", func(win *gtk.Window, ev *gdk.Event) {
		fmt.Println("keypress")
		event := gdk.EventKeyNewFromEvent(ev)
		switch event.KeyVal() {
		case gdk.KEY_space:
			playing, err := player.PausePlay()
			if err != nil {
				return
			}

			if playing {
				play.SetImage(controlPause)
			} else {
				play.SetImage(controlPlay)
			}
		case gdk.KEY_Left:
			err := player.Seek(-10)
			if err != nil {
				return
			}
		case gdk.KEY_Right:
			err := player.Seek(10)
			if err != nil {
				return
			}
		}
	})

	// Icon menu

	icon := player.getImage("icon")

	pixbuf, err := gdk.PixbufNewFromFileAtScale("icon-128.png", 22, 22, true)
	Nerr(err)

	icon.SetFromPixbuf(pixbuf)
	popObj, err := player.Builder.GetObject("pop")
	Nerr(err)

	pop := popObj.(*gtk.Popover)

	iconButton := player.getButton("iconButton")
	iconButton.Connect("clicked", func() {
		pop.Popup()
	})

	// Tracks menu
	poptObj, err := player.Builder.GetObject("tracks")
	Nerr(err)

	popt := poptObj.(*gtk.PopoverMenu)

	tracks := player.getButton("controlsTracks")
	tracks.Connect("clicked", func() {
		popt.Popup()
	})

	// Open file
	open := player.getButton("controlsOpen")
	open.Connect("clicked", func() {
		pop.Popdown()

		dialog, err := gtk.FileChooserDialogNewWith1Button("Open with jcplayer", player.Win, gtk.FILE_CHOOSER_ACTION_OPEN, "Play", gtk.RESPONSE_OK)
		Nerr(err)

		response := dialog.Run()
		if response == gtk.RESPONSE_OK {
			filename := dialog.GetFilename()
			dialog.Destroy()

			player.Conn.Call("loadfile", filename)

			ready := make(chan bool)

			player.Win.SetTitle("Loading...")
			go func() {
				for {
					_, err := player.Conn.Get("time-pos")
					if err == nil {
						ready <- true
						return
					}
				}
			}()

			<-ready // Player is ready!

			name, err := player.Conn.Get("filename")
			Nerr(err)

			player.Win.SetTitle(name.(string))

			player.MpvArea.Show()

			player.Current = Media{}
			player.SetupTracks()

		} else {
			dialog.Destroy()
		}
	})

	// Drag and drop
	et, err := gtk.TargetEntryNew("video", gtk.TARGET_SAME_APP, 1)
	Nerr(err)

	player.Win.DragDestSet(gtk.DEST_DEFAULT_DROP, []gtk.TargetEntry{*et}, gdk.ACTION_DEFAULT)

	player.Win.Connect("drag-data-received", func(window *gtk.Window, ctn *gdk.DragContext, x int, y int, data *gtk.SelectionData, info uint, time uint) {
		fmt.Println("ss")
	})

	// Label
	currentObj, err := player.Builder.GetObject("controlsCurrent")
	Nerr(err)

	current := currentObj.(*gtk.Label)
	// Progress
	progressObj, err := player.Builder.GetObject("controlsProgress")
	Nerr(err)

	progress := progressObj.(*gtk.Adjustment)
	go func() {
		for {
			select {
			case <-ticker.C:
				if player.Current.Duration == 0 {
					dur, err := player.Conn.Get("duration")
					if err != nil {
						dur = nil
					} else {
						duration := dur.(float64)
						minutes := strconv.Itoa(int(duration) / 60)
						seconds := strconv.Itoa(int(duration) % 60)
						if len(minutes) == 1 {
							minutes = "0" + minutes
						}
						if len(seconds) == 1 {
							seconds = "0" + seconds
						}
						player.Current.LengthParsed = minutes + ":" + seconds
						player.Current.Duration = duration
					}
				}

				tp, err := player.Conn.Get("time-pos")
				if err != nil { // not playing
					fmt.Println("ll")

					tp = 0
				} else {
					minutes := strconv.Itoa(int(tp.(float64)) / 60)
					seconds := strconv.Itoa(int(tp.(float64)) % 60)
					if len(minutes) == 1 {
						minutes = "0" + minutes
					}
					if len(seconds) == 1 {
						seconds = "0" + seconds
					}
					parsedNow := minutes + ":" + seconds
					glib.IdleAdd(func(value string) {
						current.SetText(value)
					}, parsedNow+"/"+player.Current.LengthParsed)
					glib.IdleAdd(func(value float64) {
						progress.SetValue(value * 100)
					}, (tp.(float64) / player.Current.Duration))
				}
			}
		}
	}()

	scaleObj, err := player.Builder.GetObject("controlsScale")
	Nerr(err)

	scale := scaleObj.(*gtk.Scale)
	scale.Connect("change-value", func(grange *gtk.Scale, sc interface{}, newv float64) {
		if newv >= 0 && newv <= 100 {
			player.Conn.Call("seek", newv, "absolute-percent", "exact")
			fmt.Println(newv)
		}

	})

	// FULLSCREEN
	full := player.getButton("controlsFull")
	unfull := player.getButton("controlsUnfull")
	unfull.Hide()

	full.Connect("clicked", func() {
		player.Win.Fullscreen()
		unfull.Show()
	})

	unfull.Connect("clicked", func() {
		player.Win.Unfullscreen()
		unfull.Hide()
	})
}

func (player *Player) SetupTracks() {
	var prevSub *gtk.RadioButton
	var prevAudio *gtk.RadioButton

	subs := player.getBox("poptSub")
	player.clearBox("poptSub")

	audio := player.getBox("poptAudio")
	player.clearBox("poptAudio")

	utracks := player.GetTracks()

	botn, err := gtk.ButtonNew()
	Nerr(err)

	radion, err := gtk.RadioButtonNewWithLabelFromWidget(prevSub, "None")

	botn.Add(radion)
	botn.Connect("clicked", func() {
		player.Conn.Set("sid", 0)
		radion.SetActive(true)
	})

	subs.Add(botn)
	botn.ShowAll()

	prevSub = radion

	for _, v := range utracks {
		if v.Type == "sub" {
			btn, err := gtk.ButtonNew()
			Nerr(err)

			trackLang, ok := v.Track["lang"].(string)
			if !ok {
				trackLang = "unknown"
			}

			trackTitle, ok := v.Track["title"].(string)
			if !ok {
				trackTitle = ""
			}
			trackId := v.Track["id"]
			radio, err := gtk.RadioButtonNewWithLabelFromWidget(prevSub, fmt.Sprintf("%v. %v (%v)", trackId, trackTitle, trackLang))
			btn.Add(radio)
			if v.Track["selected"].(bool) {
				radio.SetActive(true)
			}

			btn.Connect("clicked", func() {
				player.Conn.Set("sid", trackId)
				radio.SetActive(true)
			})

			subs.Add(btn)
			btn.ShowAll()
		} else if v.Type == "audio" {
			btn, err := gtk.ButtonNew()
			Nerr(err)

			trackLang, ok := v.Track["lang"].(string)
			if !ok {
				trackLang = "unknown"
			}

			trackTitle, ok := v.Track["title"].(string)
			if !ok {
				trackTitle = ""
			}
			trackId := v.Track["id"]
			radio, err := gtk.RadioButtonNewWithLabelFromWidget(prevAudio, fmt.Sprintf("%v. %v (%v)", trackId, trackTitle, trackLang))
			btn.Add(radio)
			if v.Track["selected"].(bool) {
				radio.SetActive(true)
			}

			btn.Connect("clicked", func() {
				fmt.Println("check", trackId)
				player.Conn.Set("aid", trackId)
				radio.SetActive(true)
			})

			audio.Add(btn)
			btn.ShowAll()
		}
	}
}

func (player *Player) clearBox(id string) {
	box, err := player.Builder.GetObject(id)
	Nerr(err)

	boxx := box.(*gtk.Box)
	children := boxx.GetChildren()
	children.Foreach(func(item interface{}) {
		fmt.Println("child")
		v := item.(*gtk.Widget)
		name, err := v.GetName()
		if err == nil {
			if name == "GtkButton" {
				boxx.Remove(v)
			}
		}
	})
}

func (player *Player) isPaused() bool {
	paused, err := player.Conn.Get("pause")
	if err != nil {
		log.Fatalln("Failed to get")
	}

	return paused.(bool)
}

func (player *Player) getButton(id string) *gtk.Button {
	object, err := player.Builder.GetObject(id)
	Nerr(err)

	button := object.(*gtk.Button)

	return button
}

func (player *Player) getButtonM(id string) *gtk.ModelButton {
	object, err := player.Builder.GetObject(id)
	Nerr(err)

	button := object.(*gtk.ModelButton)

	return button
}

func (player *Player) getBox(id string) *gtk.Box {
	object, err := player.Builder.GetObject(id)
	Nerr(err)

	button := object.(*gtk.Box)

	return button
}

func (player *Player) getImage(id string) *gtk.Image {
	object, err := player.Builder.GetObject(id)
	Nerr(err)

	img := object.(*gtk.Image)

	return img
}

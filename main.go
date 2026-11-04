package main

import (
	_ "embed"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"job-tracker-native/internal/tracker"
)

// The smaller runtime icon also works with GTK/X11; packaging uses icon.png.
//
//go:embed resources/window-icon.png
var appIcon []byte

func main() {
	dataDir := flag.String("data-dir", "", "数据与配置目录（默认系统配置目录/job-tracker）")
	flag.Parse()
	if *dataDir == "" {
		root, err := os.UserConfigDir()
		if err != nil {
			log.Fatal(err)
		}
		*dataDir = filepath.Join(root, "job-tracker")
	}
	store, err := tracker.Open(filepath.Join(*dataDir, "applications.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	a := &app{store: store, configPath: filepath.Join(*dataDir, "config.json")}
	a.config, err = tracker.LoadConfig(a.configPath)
	if err != nil {
		// A broken config must not prevent access to local applications.
		a.config = tracker.Config{}
		a.fail(err)
	}
	a.reload()
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title: "投递簿 · Job Tracker", Width: 1280, Height: 860,
			MinWidth: 1060, MinHeight: 700, StateKey: "main",
			Content: ui.View(a.view),
		})
		if err := a.win.SetIcon(appIcon); err != nil {
			log.Printf("设置窗口图标失败: %v", err)
		}
		a.win.OnClosed(a.cancelExtraction)
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

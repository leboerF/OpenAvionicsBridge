package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type cduLayout struct {
	TitleLarge uint64 `json:"titleLarge"`
	TitleSmall uint64 `json:"titleSmall"`
	LabelLayer uint64 `json:"labelLayer"`
	LargeLayer uint64 `json:"largeLayer"`
	SmallLayer uint64 `json:"smallLayer"`
	Scratchpad uint64 `json:"scratchpad"`
	URL        string `json:"url"`
}

type buildProfile struct {
	ID                    string    `json:"id"`
	DisplayName           string    `json:"displayName"`
	ModuleNames           []string  `json:"moduleNames"`
	ModuleSHA256          string    `json:"moduleSha256"`
	LinearMemoryExportRVA uint64    `json:"linearMemoryExportRva"`
	Captain               cduLayout `json:"captain"`
	Copilot               cduLayout `json:"copilot"`
}
type profileFile struct {
	Profiles []buildProfile `json:"profiles"`
}

type appSettings struct {
	CDU         string `json:"cdu"`
	AutoStart   bool   `json:"autoStart"`
	ShowPreview bool   `json:"showPreview"`
}

type statusSnapshot struct {
	MSFS        string
	F100        string
	MobiFlight  string
	Bridge      string
	Build       string
	Error       string
	Preview     [14]string
	Logs        string
	Requested   bool
	CDU         string
	AutoStart   bool
	ShowPreview bool
}

type bridgeEngine struct {
	mu          sync.Mutex
	stop        chan struct{}
	wake        chan struct{}
	requested   bool
	cdu         string
	autoStart   bool
	showPreview bool
	profiles    []buildProfile

	msfsStatus, f100Status, mfStatus, bridgeStatus, buildStatus, errorText string
	preview                                                                [14]string
	logs                                                                   []string

	handle      processHandle
	pid         uint32
	mod         winModule
	profile     *buildProfile
	layout      cduLayout
	linear      uintptr
	ws          *simpleWebSocket
	lastJSON    string
	lastSend    time.Time
	lastConnect time.Time
	lastDetect  time.Time
	hashCache   map[string]string
	stopOnce    sync.Once

	appDir, dataDir, logDir, logPath, settingsPath string
}

func defaultProfiles() []buildProfile {
	return []buildProfile{{
		ID:                    "jf-f100-1.3-analyzed",
		DisplayName:           "Just Flight F70/F100 Professional - compatible 1.3 build",
		ModuleNames:           []string{"m14f2f2d272d86b96_0.dll"},
		ModuleSHA256:          "04ec747e90bb07e88a55e977856d47d5dd0cfc5a7cbee0e899ae2770b6cace87",
		LinearMemoryExportRVA: 0x417040,
		Captain:               cduLayout{TitleLarge: 0x1805F0, TitleSmall: 0x1806B0, LabelLayer: 0x180770, LargeLayer: 0x181070, SmallLayer: 0x181970, Scratchpad: 0x182270, URL: "ws://localhost:8320/winwing/cdu-captain"},
		Copilot:               cduLayout{TitleLarge: 0x180650, TitleSmall: 0x180710, LabelLayer: 0x180BF0, LargeLayer: 0x1814F0, SmallLayer: 0x181DF0, Scratchpad: 0x1822D0, URL: "ws://localhost:8320/winwing/cdu-co-pilot"},
	}}
}

func newBridgeEngine() (*bridgeEngine, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	appDir := filepath.Dir(exe)
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		local = appDir
	}
	dataDir := filepath.Join(local, "OpenAvionicsBridge")
	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}
	e := &bridgeEngine{
		stop: make(chan struct{}), wake: make(chan struct{}, 1),
		requested: false, cdu: "captain", autoStart: true, showPreview: true,
		profiles: defaultProfiles(), hashCache: map[string]string{},
		appDir: appDir, dataDir: dataDir, logDir: logDir,
		settingsPath: filepath.Join(dataDir, "settings.json"),
		logPath:      filepath.Join(logDir, "bridge-"+time.Now().Format("2006-01-02")+".log"),
		msfsStatus:   "Checking...", f100Status: "Waiting for aircraft", mfStatus: "Not connected", bridgeStatus: "Bridge stopped",
	}
	e.loadProfiles()
	// Preserve settings from the pre-OpenAvionicsBridge builds on first launch.
	if _, err := os.Stat(e.settingsPath); os.IsNotExist(err) {
		legacy := filepath.Join(local, "F100WinCtrlBridge", "settings.json")
		if b, readErr := os.ReadFile(legacy); readErr == nil {
			_ = os.WriteFile(e.settingsPath, b, 0644)
		}
	}
	e.loadSettings()
	e.requested = e.autoStart
	e.log("INFO", "Starting "+appName+" "+appVersion)
	go e.loop()
	return e, nil
}

func (e *bridgeEngine) loadProfiles() {
	path := filepath.Join(e.appDir, "profiles.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var pf profileFile
	if json.Unmarshal(b, &pf) == nil && len(pf.Profiles) > 0 {
		e.profiles = pf.Profiles
	}
}
func (e *bridgeEngine) loadSettings() {
	b, err := os.ReadFile(e.settingsPath)
	if err != nil {
		return
	}
	var s appSettings
	if json.Unmarshal(b, &s) != nil {
		return
	}
	if s.CDU == "captain" || s.CDU == "copilot" {
		e.cdu = s.CDU
	}
	e.autoStart = s.AutoStart
	e.showPreview = s.ShowPreview
}
func (e *bridgeEngine) saveSettingsLocked() {
	b, _ := json.MarshalIndent(appSettings{CDU: e.cdu, AutoStart: e.autoStart, ShowPreview: e.showPreview}, "", "  ")
	_ = os.WriteFile(e.settingsPath, b, 0644)
}
func (e *bridgeEngine) log(level, msg string) {
	line := fmt.Sprintf("%s [%s] %s", time.Now().Format("2006-01-02 15:04:05.000"), level, msg)
	if f, err := os.OpenFile(e.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		_, _ = f.WriteString(line + "\r\n")
		_ = f.Close()
	}
	e.mu.Lock()
	e.logs = append(e.logs, line)
	if len(e.logs) > 120 {
		e.logs = e.logs[len(e.logs)-120:]
	}
	e.mu.Unlock()
}
func (e *bridgeEngine) setRequested(v bool) {
	e.mu.Lock()
	e.requested = v
	if !v {
		e.bridgeStatus = "Bridge stopped"
		e.mfStatus = "Not connected"
	}
	e.saveSettingsLocked()
	e.mu.Unlock()
	select {
	case e.wake <- struct{}{}:
	default:
	}
}
func (e *bridgeEngine) setCDU(cdu string) {
	if cdu != "captain" && cdu != "copilot" {
		return
	}
	e.mu.Lock()
	if !e.requested {
		e.cdu = cdu
		e.saveSettingsLocked()
	}
	e.mu.Unlock()
}
func (e *bridgeEngine) setAutoStart(v bool) {
	e.mu.Lock()
	e.autoStart = v
	e.saveSettingsLocked()
	e.mu.Unlock()
}
func (e *bridgeEngine) setShowPreview(v bool) {
	e.mu.Lock()
	e.showPreview = v
	e.saveSettingsLocked()
	e.mu.Unlock()
}

func (e *bridgeEngine) snapshot() statusSnapshot {
	e.mu.Lock()
	defer e.mu.Unlock()
	return statusSnapshot{MSFS: e.msfsStatus, F100: e.f100Status, MobiFlight: e.mfStatus, Bridge: e.bridgeStatus, Build: e.buildStatus, Error: e.errorText, Preview: e.preview, Logs: strings.Join(e.logs, "\r\n"), Requested: e.requested, CDU: e.cdu, AutoStart: e.autoStart, ShowPreview: e.showPreview}
}

func (e *bridgeEngine) Stop() {
	e.stopOnce.Do(func() { close(e.stop) })
}
func (e *bridgeEngine) closeResources() {
	e.mu.Lock()
	ws := e.ws
	e.ws = nil
	h := e.handle
	e.handle = 0
	e.pid = 0
	e.profile = nil
	e.linear = 0
	e.lastJSON = ""
	e.lastSend = time.Time{}
	e.mu.Unlock()
	if ws != nil {
		_ = ws.Close()
	}
	if h != 0 {
		h.Close()
	}
}

func (e *bridgeEngine) loop() {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-e.stop:
			e.closeResources()
			e.log("INFO", "Application closed")
			return
		case <-e.wake:
		case <-tick.C:
		}
		e.step()
	}
}

func (e *bridgeEngine) step() {
	e.mu.Lock()
	requested := e.requested
	auto := e.autoStart
	hasHandle := e.handle != 0
	pid := e.pid
	e.mu.Unlock()

	// If the user stopped the bridge, detach on the worker goroutine. This keeps
	// Close/Stop actions out of the Win32 UI thread.
	if !requested && hasHandle {
		e.closeResources()
		hasHandle = false
		pid = 0
	}

	var p *winProcess
	if hasHandle {
		// Once attached, ReadProcessMemory is the cheapest liveness check. Avoid a
		// full system process snapshot four times per second.
		p = &winProcess{PID: pid, Name: "FlightSimulator2024.exe"}
		e.mu.Lock()
		e.msfsStatus = fmt.Sprintf("Running (PID %d)", pid)
		e.mu.Unlock()
	} else {
		// Process/module discovery is intentionally throttled. The display itself
		// still refreshes at 10 Hz after attachment.
		if time.Since(e.lastDetect) < time.Second {
			return
		}
		e.lastDetect = time.Now()
		var err error
		p, err = findMSFS()
		if err != nil {
			e.setErr(err)
			return
		}
		e.mu.Lock()
		if p != nil {
			e.msfsStatus = fmt.Sprintf("Running (PID %d)", p.PID)
		} else {
			e.msfsStatus = "Not running"
		}
		if !requested && auto && p != nil {
			e.requested = true
			requested = true
			e.bridgeStatus = "Waiting for F100"
		}
		e.mu.Unlock()
	}

	if !requested {
		return
	}
	if p == nil {
		e.setStatus("Waiting for aircraft", "Waiting", "Waiting for F100", "")
		return
	}

	if !hasHandle {
		if err := e.attach(p); err != nil {
			e.setErr(err)
			return
		}
		e.mu.Lock()
		hasHandle = e.handle != 0
		e.mu.Unlock()
		if !hasHandle {
			return
		}
	}

	frame, err := e.readFrame()
	if err != nil {
		e.log("WARN", err.Error())
		e.setErr(err)
		e.closeResources()
		return
	}
	e.mu.Lock()
	if e.showPreview {
		e.preview = frame.Rows
	}
	layout := e.layout
	ws := e.ws
	last := e.lastJSON
	lastSend := e.lastSend
	e.errorText = ""
	e.bridgeStatus = "Bridge running"
	e.mu.Unlock()

	payload := frameJSON(frame)
	if ws == nil || !ws.Alive() {
		if time.Since(e.lastConnect) >= 2*time.Second {
			e.lastConnect = time.Now()
			nws, err := dialWebSocket(layout.URL, 2500*time.Millisecond)
			if err != nil {
				e.mu.Lock()
				e.mfStatus = "MobiFlight not reachable - retrying"
				e.mu.Unlock()
			} else {
				e.mu.Lock()
				e.ws = nws
				e.lastJSON = ""
				e.lastSend = time.Time{}
				e.mfStatus = "Connected"
				e.mu.Unlock()
				ws = nws
				last = ""
				lastSend = time.Time{}
				e.log("INFO", "Connected to "+layout.URL)
			}
		}
	}

	// Repeat the current frame every two seconds as a lightweight health check.
	// Unlike the old ping loop this uses the exact message MobiFlight expects.
	if ws != nil && ws.Alive() && (payload != last || time.Since(lastSend) >= 2*time.Second) {
		if err := ws.SendText(payload); err != nil {
			e.log("WARN", "WinCtrl WebSocket lost: "+err.Error())
			_ = ws.Close()
			e.mu.Lock()
			e.ws = nil
			e.lastJSON = ""
			e.lastSend = time.Time{}
			e.mfStatus = "Connection lost - retrying"
			e.mu.Unlock()
		} else {
			e.mu.Lock()
			e.lastJSON = payload
			e.lastSend = time.Now()
			e.mfStatus = "Connected"
			e.mu.Unlock()
		}
	}
}

func (e *bridgeEngine) setStatus(f100, mf, bridge, build string) {
	e.mu.Lock()
	e.f100Status = f100
	e.mfStatus = mf
	e.bridgeStatus = bridge
	e.buildStatus = build
	e.mu.Unlock()
}
func (e *bridgeEngine) setErr(err error) {
	e.mu.Lock()
	same := e.errorText == err.Error()
	e.errorText = err.Error()
	e.bridgeStatus = "Recovering / waiting"
	e.mu.Unlock()
	if !same {
		e.log("ERROR", err.Error())
	}
}

func (e *bridgeEngine) attach(p *winProcess) error {
	mods, err := listModules(p.PID)
	if err != nil {
		return err
	}
	var candidate *winModule
	var prof *buildProfile
	var mismatch string
	for pi := range e.profiles {
		pr := &e.profiles[pi]
		for mi := range mods {
			m := &mods[mi]
			for _, n := range pr.ModuleNames {
				if strings.EqualFold(m.Name, n) {
					hash, ok := e.hashCache[m.Path]
					if !ok {
						h, er := sha256File(m.Path)
						if er != nil {
							return fmt.Errorf("F100 module found but build could not be verified: %v", er)
						}
						hash = h
						e.hashCache[m.Path] = h
					}
					if !strings.EqualFold(hash, pr.ModuleSHA256) {
						mismatch = hash
						continue
					}
					cp := *m
					candidate = &cp
					prof = pr
					break
				}
			}
			if candidate != nil {
				break
			}
		}
		if candidate != nil {
			break
		}
	}
	if candidate == nil {
		e.mu.Lock()
		e.f100Status = "F100 build not detected / unsupported"
		e.mfStatus = "Waiting"
		e.bridgeStatus = "Waiting for supported F100"
		if mismatch != "" {
			e.buildStatus = "Unsupported module SHA256: " + mismatch
		} else {
			e.buildStatus = ""
		}
		e.mu.Unlock()
		return nil
	}
	h, err := openReadProcess(p.PID)
	if err != nil {
		return err
	}
	linear64, err := h.ReadU64(candidate.Base + uintptr(prof.LinearMemoryExportRVA))
	if err != nil {
		h.Close()
		return err
	}
	if linear64 < 0x10000 {
		h.Close()
		return fmt.Errorf("Implausible WASM linear-memory pointer 0x%X", linear64)
	}
	e.mu.Lock()
	cdu := e.cdu
	layout := prof.Captain
	if cdu == "copilot" {
		layout = prof.Copilot
	}
	e.handle = h
	e.pid = p.PID
	e.mod = *candidate
	e.profile = prof
	e.layout = layout
	e.linear = uintptr(linear64)
	e.f100Status = "Supported F100 build detected"
	e.buildStatus = prof.DisplayName
	e.bridgeStatus = "Bridge running"
	e.errorText = ""
	e.mu.Unlock()
	e.log("INFO", fmt.Sprintf("Attached to MSFS PID %d; module %s; linear memory 0x%X; CDU %s", p.PID, candidate.Name, linear64, cdu))
	return nil
}

func (e *bridgeEngine) readFrame() (displayFrame, error) {
	e.mu.Lock()
	h := e.handle
	linear := e.linear
	l := e.layout
	e.mu.Unlock()
	if h == 0 || linear == 0 {
		return displayFrame{}, fmt.Errorf("not attached")
	}
	read := func(off uint64, n int) ([]byte, error) { return h.Read(linear+uintptr(off), n) }
	a, err := read(l.TitleLarge, 0x60)
	if err != nil {
		return displayFrame{}, err
	}
	b, err := read(l.TitleSmall, 0x60)
	if err != nil {
		return displayFrame{}, err
	}
	c, err := read(l.LabelLayer, 0x480)
	if err != nil {
		return displayFrame{}, err
	}
	d, err := read(l.LargeLayer, 0x480)
	if err != nil {
		return displayFrame{}, err
	}
	s, err := read(l.SmallLayer, 0x480)
	if err != nil {
		return displayFrame{}, err
	}
	x, err := read(l.Scratchpad, 0x60)
	if err != nil {
		return displayFrame{}, err
	}
	return buildDisplay(decodeWasmString(a, 0, 24), decodeWasmString(b, 0, 24), decodeWasmLayer(c), decodeWasmLayer(d), decodeWasmLayer(s), decodeWasmString(x, 0, 24)), nil
}

func frameJSON(f displayFrame) string {
	var sb strings.Builder
	sb.Grow(7000)
	sb.WriteString(`{"Target":"Display","Data":[`)
	for i, c := range f.Cells {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteByte('[')
		sb.WriteString(strconv.Quote(c.Ch))
		sb.WriteString(`,"g",`)
		sb.WriteString(strconv.Itoa(c.Size))
		sb.WriteByte(']')
	}
	sb.WriteString(`]}`)
	return sb.String()
}

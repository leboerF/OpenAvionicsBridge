package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type appSettings struct {
	CDU         string `json:"cdu"`
	AutoStart   bool   `json:"autoStart"`
	ShowPreview bool   `json:"showPreview"`
}

type statusSnapshot struct {
	MSFS           string
	Aircraft       string
	MobiFlight     string
	Bridge         string
	Build          string
	Update         string
	UpdateURL      string
	Error          string
	Preview        [14]string
	Logs           string
	Requested      bool
	CDU            string
	AutoStart      bool
	ShowPreview    bool
	TestRunning    bool
	UpdateChecking bool
}

type bridgeEngine struct {
	mu          sync.Mutex
	stop        chan struct{}
	wake        chan struct{}
	requested   bool
	cdu         string
	autoStart   bool
	showPreview bool
	profiles    []aircraftProfile
	adapters    map[string]aircraftAdapter

	msfsStatus, aircraftStatus, mfStatus, bridgeStatus, buildStatus, errorText string
	updateStatus, updateURL                                                    string
	updateChecking, testRunning                                                bool
	preview                                                                    [14]string
	logs                                                                       []string

	handle      processHandle
	pid         uint32
	mod         winModule
	profile     *aircraftProfile
	layout      cduLayout
	linear      uintptr
	ws          *simpleWebSocket
	lastJSON    string
	lastSend    time.Time
	lastConnect time.Time
	lastDetect  time.Time
	stopOnce    sync.Once

	appDir, dataDir, logDir, logPath, settingsPath, compatibilityReportPath string
	lastCompatibilityReport                                                 string
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
		profiles: defaultProfiles(), adapters: defaultAircraftAdapters(),
		appDir: appDir, dataDir: dataDir, logDir: logDir,
		settingsPath:            filepath.Join(dataDir, "settings.json"),
		compatibilityReportPath: filepath.Join(logDir, "compatibility-report.txt"),
		logPath:                 filepath.Join(logDir, "bridge-"+time.Now().Format("2006-01-02")+".log"),
		msfsStatus:              "Checking...", aircraftStatus: "Waiting for aircraft", mfStatus: "Not connected", bridgeStatus: "Bridge stopped",
		buildStatus:             "Profile: not attached\r\nBuild: waiting for compatible aircraft",
		updateStatus:            "Checking for updates...",
	}
	e.loadProfiles()
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
	e.checkUpdates(false)
	return e, nil
}

func (e *bridgeEngine) loadProfiles() {
	path := filepath.Join(e.appDir, "profiles.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	profiles, err := mergeProfileJSON(e.profiles, b)
	if err != nil {
		e.log("WARN", "External profiles.json ignored: "+err.Error())
		return
	}
	e.profiles = profiles
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
	return statusSnapshot{
		MSFS: e.msfsStatus, Aircraft: e.aircraftStatus, MobiFlight: e.mfStatus, Bridge: e.bridgeStatus,
		Build: e.buildStatus, Update: e.updateStatus, UpdateURL: e.updateURL, Error: e.errorText,
		Preview: e.preview, Logs: strings.Join(e.logs, "\r\n"), Requested: e.requested, CDU: e.cdu,
		AutoStart: e.autoStart, ShowPreview: e.showPreview, TestRunning: e.testRunning, UpdateChecking: e.updateChecking,
	}
}

func (e *bridgeEngine) diagnosticsText() string {
	s := e.snapshot()
	var b strings.Builder
	fmt.Fprintf(&b, "OpenAvionicsBridge diagnostics\r\n")
	fmt.Fprintf(&b, "Version: %s\r\n", appVersion)
	fmt.Fprintf(&b, "MSFS: %s\r\n", s.MSFS)
	fmt.Fprintf(&b, "Aircraft: %s\r\n", s.Aircraft)
	fmt.Fprintf(&b, "MobiFlight: %s\r\n", s.MobiFlight)
	fmt.Fprintf(&b, "Bridge: %s\r\n", s.Bridge)
	fmt.Fprintf(&b, "CDU: %s\r\n", s.CDU)
	fmt.Fprintf(&b, "Update: %s\r\n", s.Update)
	b.WriteString(s.Build)
	b.WriteString("\r\n")
	if report, err := os.ReadFile(e.compatibilityReportPath); err == nil && len(report) > 0 {
		b.WriteString("\r\nCompatibility report\r\n--------------------\r\n")
		b.Write(report)
	}
	return b.String()
}

func (e *bridgeEngine) checkUpdates(manual bool) {
	e.mu.Lock()
	if e.updateChecking {
		e.mu.Unlock()
		return
	}
	e.updateChecking = true
	e.updateStatus = "Checking for updates..."
	e.updateURL = ""
	e.mu.Unlock()

	go func() {
		result, err := queryUpdates(appVersion)
		e.mu.Lock()
		e.updateChecking = false
		if err != nil {
			if appVersion == "dev" {
				e.updateStatus = "Development build · update check disabled"
			} else {
				e.updateStatus = "Update check unavailable"
			}
			e.updateURL = ""
			e.mu.Unlock()
			if manual {
				e.log("WARN", "Update check failed: "+err.Error())
			}
			return
		}
		if result.Available {
			e.updateStatus = "Update " + result.Version + " available · click to open"
			e.updateURL = result.URL
		} else {
			e.updateStatus = "Up to date · click to recheck"
			e.updateURL = ""
		}
		e.mu.Unlock()
	}()
}

func (e *bridgeEngine) startTestDisplay() {
	e.mu.Lock()
	if e.requested || e.testRunning {
		e.mu.Unlock()
		return
	}
	var layout cduLayout
	if e.profile != nil {
		layout = e.profile.Captain
		if e.cdu == "copilot" {
			layout = e.profile.Copilot
		}
	} else if len(e.profiles) > 0 {
		layout = e.profiles[0].Captain
		if e.cdu == "copilot" {
			layout = e.profiles[0].Copilot
		}
	}
	if strings.TrimSpace(layout.URL) == "" {
		e.mu.Unlock()
		e.log("ERROR", "No output endpoint is available for the test display")
		return
	}
	cdu := e.cdu
	e.testRunning = true
	e.mfStatus = "Testing MobiFlight..."
	e.errorText = ""
	e.mu.Unlock()

	go func() {
		defer func() {
			e.mu.Lock()
			e.testRunning = false
			e.mu.Unlock()
		}()
		ws, err := dialWebSocket(layout.URL, 2500*time.Millisecond)
		if err != nil {
			e.mu.Lock()
			e.mfStatus = "Test display failed"
			e.errorText = "Test display: " + err.Error()
			e.mu.Unlock()
			e.log("WARN", "Test display failed: "+err.Error())
			return
		}
		defer ws.Close()
		frame := testDisplayFrame(cdu)
		if err := ws.SendText(frameJSON(frame)); err != nil {
			e.mu.Lock()
			e.mfStatus = "Test display failed"
			e.errorText = "Test display: " + err.Error()
			e.mu.Unlock()
			e.log("WARN", "Test display send failed: "+err.Error())
			return
		}
		e.mu.Lock()
		e.preview = frame.Rows
		e.mfStatus = "Test display active"
		e.mu.Unlock()
		e.log("INFO", "Test display sent to "+layout.URL)
		time.Sleep(2 * time.Second)
		if ws.Alive() {
			_ = ws.SendText(frameJSON(frame))
		}
		time.Sleep(2 * time.Second)
		e.mu.Lock()
		e.mfStatus = "Test display finished"
		e.mu.Unlock()
	}()
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

	if !requested && hasHandle {
		e.closeResources()
		hasHandle = false
		pid = 0
	}

	var p *winProcess
	if hasHandle {
		p = &winProcess{PID: pid, Name: "FlightSimulator2024.exe"}
		e.mu.Lock()
		e.msfsStatus = fmt.Sprintf("Running (PID %d)", pid)
		e.mu.Unlock()
	} else {
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
			e.bridgeStatus = "Waiting for supported aircraft"
		}
		e.mu.Unlock()
	}

	if !requested {
		return
	}
	if p == nil {
		e.setStatus("Waiting for aircraft", "Waiting", "Waiting for supported aircraft", "")
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

func (e *bridgeEngine) setStatus(aircraft, mf, bridge, build string) {
	e.mu.Lock()
	e.aircraftStatus = aircraft
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
	h, err := openReadProcess(p.PID)
	if err != nil {
		return err
	}

	var closest string
	for pi := range e.profiles {
		pr := &e.profiles[pi]
		adapter := e.adapters[pr.Adapter]
		if adapter == nil {
			continue
		}
		att, diagnostic, er := adapter.TryAttach(h, pr, mods)
		if er != nil {
			h.Close()
			return er
		}
		if diagnostic != "" {
			closest = diagnostic
		}
		if att == nil {
			continue
		}

		e.mu.Lock()
		cdu := e.cdu
		layout := pr.Captain
		if cdu == "copilot" {
			layout = pr.Copilot
		}
		e.handle = h
		e.pid = p.PID
		e.mod = att.Module
		e.profile = pr
		e.layout = layout
		e.linear = att.Linear
		e.aircraftStatus = "Supported aircraft detected"
		shortHash := att.ModuleHash
		if len(shortHash) > 12 {
			shortHash = shortHash[:12]
		}
		buildKind := "Runtime validated"
		if att.KnownHash {
			buildKind = "Verified"
		}
		e.buildStatus = "Profile: " + pr.DisplayName + " · " + pr.Adapter + "\r\nBuild: " + buildKind
		if shortHash != "" {
			e.buildStatus += " · SHA256 " + shortHash + "…"
		}
		e.bridgeStatus = "Bridge running"
		e.errorText = ""
		e.mu.Unlock()

		hashText := att.ModuleHash
		if hashText == "" {
			hashText = "unavailable"
		}
		report := fmt.Sprintf(
			"OpenAvionicsBridge compatibility report\r\n"+
				"Version: %s\r\n"+
				"Profile: %s\r\n"+
				"Adapter: %s\r\n"+
				"MSFS PID: %d\r\n"+
				"Module: %s\r\n"+
				"Module path: %s\r\n"+
				"Module SHA256: %s\r\n"+
				"Known hash: %t\r\n"+
				"Linear-memory export: %s\r\n"+
				"Linear-memory export RVA: 0x%X\r\n"+
				"Linear memory: 0x%X\r\n"+
				"Validation: %s\r\n",
			appVersion, pr.DisplayName, pr.Adapter, p.PID, att.Module.Name, att.Module.Path,
			hashText, att.KnownHash, att.LinearExport, att.LinearRVA, att.Linear, att.ValidationMsg,
		)
		e.writeCompatibilityReport(report)
		e.log("INFO", fmt.Sprintf("Attached to MSFS PID %d; profile %s; module %s; knownHash=%t; linear memory 0x%X; CDU %s", p.PID, pr.ID, att.Module.Name, att.KnownHash, att.Linear, cdu))
		return nil
	}

	h.Close()
	e.mu.Lock()
	e.aircraftStatus = "No compatible aircraft detected"
	e.mfStatus = "Waiting"
	e.bridgeStatus = "Waiting for supported aircraft"
	if closest != "" {
		e.buildStatus = "Profile: candidate rejected\r\nBuild: " + closest
	} else {
		e.buildStatus = "Profile: scanning registered profiles\r\nBuild: waiting for a compatible runtime signature"
	}
	e.mu.Unlock()
	if closest != "" {
		e.writeCompatibilityReport("OpenAvionicsBridge compatibility report\r\nVersion: " + appVersion + "\r\nStatus: candidate rejected\r\nReason: " + closest + "\r\n")
	}
	return nil
}

func (e *bridgeEngine) writeCompatibilityReport(report string) {
	e.mu.Lock()
	if report == e.lastCompatibilityReport {
		e.mu.Unlock()
		return
	}
	e.lastCompatibilityReport = report
	path := e.compatibilityReportPath
	e.mu.Unlock()
	_ = os.WriteFile(path, []byte(report), 0644)
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


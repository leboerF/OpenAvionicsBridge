package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

var appVersion = "dev"

const (
	appName      = "OpenAvionicsBridge"
	appCopyright = "© 2026 leboerF · MIT License"
)

const (
	wmDestroy        = 0x0002
	wmPaint          = 0x000F
	wmClose          = 0x0010
	wmEraseBkgnd     = 0x0014
	wmDrawItem       = 0x002B
	wmSetFont        = 0x0030
	wmCtlColorEdit   = 0x0133
	wmCtlColorBtn    = 0x0135
	wmCtlColorStatic = 0x0138
	wmCommand        = 0x0111
	wmTimer          = 0x0113
	emSetSel         = 0x00B1
	emScrollCaret    = 0x00B7
	bmGetCheck       = 0x00F0
	bmSetCheck       = 0x00F1
	bstChecked       = 1
	cbAddString      = 0x0143
	cbGetCurSel      = 0x0147
	cbSetCurSel      = 0x014E
	cbnSelChange     = 1
	bnClicked        = 0

	wsOverlapped   = 0x00000000
	wsCaption      = 0x00C00000
	wsSysMenu      = 0x00080000
	wsMinimizeBox  = 0x00020000
	wsChild        = 0x40000000
	wsVisible      = 0x10000000
	wsVScroll      = 0x00200000
	wsTabStop      = 0x00010000
	wsExClientEdge = 0x00000200

	bsPushButton    = 0x00000000
	bsAutoCheckBox  = 0x00000003
	bsOwnerDraw     = 0x0000000B
	cbsDropDownList = 0x00000003
	esMultiline     = 0x0004
	esAutoVScroll   = 0x0040
	esReadOnly      = 0x0800

	dtCenter     = 0x00000001
	dtVCenter    = 0x00000004
	dtSingleLine = 0x00000020

	transparent = 1
	psSolid     = 0
	odsSelected = 0x0001
	odsDisabled = 0x0004

	swShow  = 5
	idTimer = 1
)

const (
	idCdu          = 1001
	idAuto         = 1002
	idStart        = 1003
	idStop         = 1004
	idPreviewCheck = 1005
	idOpenLogs     = 1006
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	gdi32               = syscall.NewLazyDLL("gdi32.dll")
	shell32             = syscall.NewLazyDLL("shell32.dll")
	pRegisterClassW     = user32.NewProc("RegisterClassW")
	pCreateWindowExW    = user32.NewProc("CreateWindowExW")
	pDefWindowProcW     = user32.NewProc("DefWindowProcW")
	pShowWindow         = user32.NewProc("ShowWindow")
	pUpdateWindow       = user32.NewProc("UpdateWindow")
	pGetMessageW        = user32.NewProc("GetMessageW")
	pTranslateMessage   = user32.NewProc("TranslateMessage")
	pDispatchMessageW   = user32.NewProc("DispatchMessageW")
	pPostQuitMessage    = user32.NewProc("PostQuitMessage")
	pSendMessageW       = user32.NewProc("SendMessageW")
	pSetWindowTextW     = user32.NewProc("SetWindowTextW")
	pEnableWindow       = user32.NewProc("EnableWindow")
	pSetTimer           = user32.NewProc("SetTimer")
	pKillTimer          = user32.NewProc("KillTimer")
	pLoadCursorW        = user32.NewProc("LoadCursorW")
	pLoadIconW          = user32.NewProc("LoadIconW")
	pGetSystemMetrics   = user32.NewProc("GetSystemMetrics")
	pSetProcessDPIAware = user32.NewProc("SetProcessDPIAware")
	pMessageBoxW        = user32.NewProc("MessageBoxW")
	pBeginPaint         = user32.NewProc("BeginPaint")
	pEndPaint           = user32.NewProc("EndPaint")
	pGetClientRect      = user32.NewProc("GetClientRect")
	pInvalidateRect     = user32.NewProc("InvalidateRect")
	pDrawTextW          = user32.NewProc("DrawTextW")
	pCreateFontW        = gdi32.NewProc("CreateFontW")
	pCreateSolidBrush   = gdi32.NewProc("CreateSolidBrush")
	pCreatePen          = gdi32.NewProc("CreatePen")
	pDeleteObject       = gdi32.NewProc("DeleteObject")
	pSelectObject       = gdi32.NewProc("SelectObject")
	pFillRect           = user32.NewProc("FillRect")
	pRoundRect          = gdi32.NewProc("RoundRect")
	pSetTextColor       = gdi32.NewProc("SetTextColor")
	pSetBkColor         = gdi32.NewProc("SetBkColor")
	pSetBkMode          = gdi32.NewProc("SetBkMode")
	pShellExecuteW      = shell32.NewProc("ShellExecuteW")
	pGetModuleHandleW   = k32.NewProc("GetModuleHandleW")
)

type wndClass struct {
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
}

type point struct{ X, Y int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
type paintStruct struct {
	Hdc         uintptr
	Erase       int32
	Paint       rect
	Restore     int32
	IncUpdate   int32
	RGBReserved [32]byte
}
type msg struct {
	Hwnd           uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             point
	Private        uint32
}
type drawItemStruct struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	HwndItem   uintptr
	HDC        uintptr
	RcItem     rect
	ItemData   uintptr
}

type guiState struct {
	engine *bridgeEngine
	hwnd   uintptr

	statusMSFS, statusF100, statusMF, statusBridge, statusBuild uintptr
	dotMSFS, dotF100, dotMF, dotBridge                          uintptr
	headerStatus                                                uintptr
	cdu, auto, start, stop                                      uintptr
	previewCheck, preview, logBox, errorLabel, openLogs         uintptr

	font, fontTitle, fontSection, fontLabel, fontSmall, fontMono uintptr
	brushMain, brushWhite, brushPreview, brushLog                uintptr
	last                                                         statusSnapshot
	textColors                                                   map[uintptr]uint32
}

var gui *guiState

func ptr(s string) *uint16    { p, _ := syscall.UTF16PtrFromString(s); return p }
func loword(v uintptr) uint16 { return uint16(v & 0xffff) }
func hiword(v uintptr) uint16 { return uint16((v >> 16) & 0xffff) }
func rgb(r, g, b byte) uint32 { return uint32(r) | uint32(g)<<8 | uint32(b)<<16 }

var (
	clrMain       = rgb(244, 247, 249)
	clrWhite      = rgb(255, 255, 255)
	clrBorder     = rgb(219, 226, 232)
	clrText       = rgb(30, 38, 45)
	clrMuted      = rgb(102, 114, 124)
	clrGreen      = rgb(34, 132, 90)
	clrGreenDark  = rgb(27, 108, 73)
	clrAmber      = rgb(207, 137, 37)
	clrRed        = rgb(190, 62, 62)
	clrGray       = rgb(145, 154, 163)
	clrPreviewBG  = rgb(14, 20, 17)
	clrPreviewFG  = rgb(113, 230, 151)
	clrLogBG      = rgb(248, 250, 251)
	clrButtonSoft = rgb(235, 240, 243)
)

func setText(hwnd uintptr, s string) {
	p := ptr(s)
	pSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(p)))
}
func send(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	r, _, _ := pSendMessageW.Call(hwnd, uintptr(msg), wp, lp)
	return r
}
func enable(hwnd uintptr, v bool) {
	x := uintptr(0)
	if v {
		x = 1
	}
	pEnableWindow.Call(hwnd, x)
}
func isChecked(hwnd uintptr) bool { return send(hwnd, bmGetCheck, 0, 0) == bstChecked }
func setChecked(hwnd uintptr, v bool) {
	x := uintptr(0)
	if v {
		x = bstChecked
	}
	send(hwnd, bmSetCheck, x, 0)
}
func setColor(hwnd uintptr, color uint32) {
	if gui == nil || hwnd == 0 {
		return
	}
	gui.textColors[hwnd] = color
	pInvalidateRect.Call(hwnd, 0, 1)
}

func createFont(face string, height int32, weight int32) uintptr {
	f := ptr(face)
	r, _, _ := pCreateFontW.Call(uintptr(uint32(height)), 0, 0, 0, uintptr(uint32(weight)), 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(f)))
	return r
}
func createControl(exStyle uint32, class, text string, style uint32, x, y, w, h int, id int, font uintptr) uintptr {
	inst, _, _ := pGetModuleHandleW.Call(0)
	hwnd, _, _ := pCreateWindowExW.Call(uintptr(exStyle), uintptr(unsafe.Pointer(ptr(class))), uintptr(unsafe.Pointer(ptr(text))), uintptr(style), uintptr(x), uintptr(y), uintptr(w), uintptr(h), gui.hwnd, uintptr(id), inst, 0)
	if font != 0 {
		send(hwnd, wmSetFont, font, 1)
	}
	return hwnd
}
func static(text string, x, y, w, h int, font uintptr) uintptr {
	return createControl(0, "STATIC", text, wsChild|wsVisible, x, y, w, h, 0, font)
}

func fill(hdc uintptr, r rect, brush uintptr) {
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&r)), brush)
}
func card(hdc uintptr, r rect) {
	brush, _, _ := pCreateSolidBrush.Call(uintptr(clrWhite))
	pen, _, _ := pCreatePen.Call(psSolid, 1, uintptr(clrBorder))
	oldBrush, _, _ := pSelectObject.Call(hdc, brush)
	oldPen, _, _ := pSelectObject.Call(hdc, pen)
	pRoundRect.Call(hdc, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom), 14, 14)
	pSelectObject.Call(hdc, oldBrush)
	pSelectObject.Call(hdc, oldPen)
	pDeleteObject.Call(brush)
	pDeleteObject.Call(pen)
}

func paintWindow(hwnd uintptr) {
	var ps paintStruct
	hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))

	var cr rect
	pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&cr)))
	fill(hdc, cr, gui.brushMain)

	fill(hdc, rect{0, 0, cr.Right, 92}, gui.brushWhite)
	accent, _, _ := pCreateSolidBrush.Call(uintptr(clrGreen))
	fill(hdc, rect{0, 0, cr.Right, 4}, accent)
	pDeleteObject.Call(accent)

	card(hdc, rect{20, 108, 550, 310})
	card(hdc, rect{570, 108, 900, 310})
	card(hdc, rect{20, 330, 590, 655})
	card(hdc, rect{610, 330, 900, 655})

	fill(hdc, rect{0, 670, cr.Right, cr.Bottom}, gui.brushWhite)
}

func statusColor(v string, requested bool) uint32 {
	s := strings.ToLower(strings.TrimSpace(v))
	switch {
	case strings.Contains(s, "error"), strings.Contains(s, "failed"), strings.Contains(s, "unsupported"):
		return clrRed
	case strings.Contains(s, "not connected"), strings.Contains(s, "stopped"):
		if requested {
			return clrAmber
		}
		return clrGray
	case strings.Contains(s, "connected"), strings.Contains(s, "running"), strings.Contains(s, "ready"), strings.Contains(s, "compatible"):
		return clrGreen
	case strings.Contains(s, "checking"), strings.Contains(s, "waiting"), strings.Contains(s, "detect"), strings.Contains(s, "connecting"):
		return clrAmber
	default:
		return clrGray
	}
}

func drawOwnerButton(dis *drawItemStruct) {
	if dis == nil || gui == nil {
		return
	}
	primary := int(dis.CtlID) == idStart
	pressed := dis.ItemState&odsSelected != 0
	disabled := dis.ItemState&odsDisabled != 0

	bg := clrButtonSoft
	fg := clrText
	border := clrBorder
	if primary {
		bg = clrGreen
		fg = clrWhite
		border = clrGreen
		if pressed {
			bg = clrGreenDark
			border = clrGreenDark
		}
	} else if pressed {
		bg = rgb(220, 227, 231)
	}
	if disabled {
		bg = rgb(242, 245, 247)
		fg = rgb(158, 166, 172)
		border = rgb(229, 233, 236)
	}

	brush, _, _ := pCreateSolidBrush.Call(uintptr(bg))
	pen, _, _ := pCreatePen.Call(psSolid, 1, uintptr(border))
	oldBrush, _, _ := pSelectObject.Call(dis.HDC, brush)
	oldPen, _, _ := pSelectObject.Call(dis.HDC, pen)
	pRoundRect.Call(dis.HDC, uintptr(dis.RcItem.Left), uintptr(dis.RcItem.Top), uintptr(dis.RcItem.Right), uintptr(dis.RcItem.Bottom), 10, 10)
	pSelectObject.Call(dis.HDC, oldBrush)
	pSelectObject.Call(dis.HDC, oldPen)
	pDeleteObject.Call(brush)
	pDeleteObject.Call(pen)

	font := gui.fontLabel
	oldFont, _, _ := pSelectObject.Call(dis.HDC, font)
	pSetBkMode.Call(dis.HDC, transparent)
	pSetTextColor.Call(dis.HDC, uintptr(fg))
	caption := ""
	switch int(dis.CtlID) {
	case idStart:
		caption = "Start bridge"
	case idStop:
		caption = "Stop"
	case idOpenLogs:
		caption = "Open logs"
	}
	t := ptr(caption)
	r := dis.RcItem
	pDrawTextW.Call(dis.HDC, uintptr(unsafe.Pointer(t)), ^uintptr(0), uintptr(unsafe.Pointer(&r)), dtCenter|dtVCenter|dtSingleLine)
	pSelectObject.Call(dis.HDC, oldFont)
}

func wndProc(hwnd uintptr, m uint32, wparam, lparam uintptr) uintptr {
	if gui != nil {
		switch m {
		case wmPaint:
			paintWindow(hwnd)
			return 0
		case wmEraseBkgnd:
			return 1
		case wmCtlColorStatic:
			hdc := wparam
			ctrl := lparam
			pSetBkMode.Call(hdc, transparent)
			color := clrText
			if c, ok := gui.textColors[ctrl]; ok {
				color = c
			}
			pSetTextColor.Call(hdc, uintptr(color))
			if ctrl == gui.preview {
				pSetBkColor.Call(hdc, uintptr(clrPreviewBG))
				return gui.brushPreview
			}
			if ctrl == gui.logBox {
				pSetBkColor.Call(hdc, uintptr(clrLogBG))
				pSetTextColor.Call(hdc, uintptr(clrText))
				return gui.brushLog
			}
			return gui.brushWhite
		case wmCtlColorBtn:
			hdc := wparam
			pSetBkColor.Call(hdc, uintptr(clrWhite))
			pSetTextColor.Call(hdc, uintptr(clrText))
			return gui.brushWhite
		case wmCtlColorEdit:
			hdc := wparam
			ctrl := lparam
			if ctrl == gui.preview {
				pSetBkColor.Call(hdc, uintptr(clrPreviewBG))
				pSetTextColor.Call(hdc, uintptr(clrPreviewFG))
				return gui.brushPreview
			}
			if ctrl == gui.logBox {
				pSetBkColor.Call(hdc, uintptr(clrLogBG))
				pSetTextColor.Call(hdc, uintptr(clrText))
				return gui.brushLog
			}
		case wmDrawItem:
			dis := (*drawItemStruct)(unsafe.Pointer(lparam))
			drawOwnerButton(dis)
			return 1
		case wmCommand:
			id := int(loword(wparam))
			code := hiword(wparam)
			switch id {
			case idStart:
				if code == bnClicked {
					gui.engine.setRequested(true)
				}
			case idStop:
				if code == bnClicked {
					gui.engine.setRequested(false)
				}
			case idCdu:
				if code == cbnSelChange {
					idx := send(gui.cdu, cbGetCurSel, 0, 0)
					if idx == 1 {
						gui.engine.setCDU("copilot")
					} else {
						gui.engine.setCDU("captain")
					}
				}
			case idAuto:
				if code == bnClicked {
					gui.engine.setAutoStart(isChecked(gui.auto))
				}
			case idPreviewCheck:
				if code == bnClicked {
					gui.engine.setShowPreview(isChecked(gui.previewCheck))
				}
			case idOpenLogs:
				if code == bnClicked {
					openFolder(gui.engine.logDir)
				}
			}
			return 0
		case wmTimer:
			updateGUI()
			return 0
		case wmClose:
			pKillTimer.Call(hwnd, idTimer)
			gui.engine.Stop()
			pPostQuitMessage.Call(0)
			return 0
		case wmDestroy:
			pPostQuitMessage.Call(0)
			return 0
		}
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, uintptr(m), wparam, lparam)
	return r
}

func openFolder(path string) {
	verb := ptr("open")
	file := ptr(path)
	pShellExecuteW.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), 0, 0, 1)
}

func updateGUI() {
	s := gui.engine.snapshot()

	if s.MSFS != gui.last.MSFS {
		setText(gui.statusMSFS, s.MSFS)
		setColor(gui.dotMSFS, statusColor(s.MSFS, s.Requested))
	}
	if s.F100 != gui.last.F100 {
		setText(gui.statusF100, s.F100)
		setColor(gui.dotF100, statusColor(s.F100, s.Requested))
	}
	if s.MobiFlight != gui.last.MobiFlight {
		setText(gui.statusMF, s.MobiFlight)
		setColor(gui.dotMF, statusColor(s.MobiFlight, s.Requested))
	}
	if s.Bridge != gui.last.Bridge || s.Requested != gui.last.Requested {
		setText(gui.statusBridge, s.Bridge)
		setColor(gui.dotBridge, statusColor(s.Bridge, s.Requested))
		setText(gui.headerStatus, strings.ToUpper(s.Bridge))
		setColor(gui.headerStatus, statusColor(s.Bridge, s.Requested))
	}
	if s.Build != gui.last.Build {
		setText(gui.statusBuild, s.Build)
	}
	if s.Error != gui.last.Error {
		setText(gui.errorLabel, s.Error)
		if s.Error != "" {
			setColor(gui.errorLabel, clrRed)
		} else {
			setColor(gui.errorLabel, clrMuted)
		}
	}
	if s.Logs != gui.last.Logs {
		setText(gui.logBox, s.Logs)
		send(gui.logBox, emSetSel, ^uintptr(0), ^uintptr(0))
		send(gui.logBox, emScrollCaret, 0, 0)
	}
	if s.ShowPreview {
		p := strings.Join(s.Preview[:], "\r\n")
		old := strings.Join(gui.last.Preview[:], "\r\n")
		if p != old || s.ShowPreview != gui.last.ShowPreview {
			setText(gui.preview, p)
		}
	} else if gui.last.ShowPreview {
		setText(gui.preview, "Live preview disabled")
	}
	if s.Requested != gui.last.Requested {
		enable(gui.start, !s.Requested)
		enable(gui.stop, s.Requested)
		enable(gui.cdu, !s.Requested)
		pInvalidateRect.Call(gui.start, 0, 1)
		pInvalidateRect.Call(gui.stop, 0, 1)
	}
	if s.CDU != gui.last.CDU {
		if s.CDU == "copilot" {
			send(gui.cdu, cbSetCurSel, 1, 0)
		} else {
			send(gui.cdu, cbSetCurSel, 0, 0)
		}
	}
	if s.AutoStart != gui.last.AutoStart {
		setChecked(gui.auto, s.AutoStart)
	}
	if s.ShowPreview != gui.last.ShowPreview {
		setChecked(gui.previewCheck, s.ShowPreview)
	}
	gui.last = s
}

func initGUI(e *bridgeEngine) error {
	gui = &guiState{engine: e, textColors: map[uintptr]uint32{}}
	_, _, _ = pSetProcessDPIAware.Call()
	inst, _, _ := pGetModuleHandleW.Call(0)
	cls := ptr("OpenAvionicsBridgeWindow")
	cursor, _, _ := pLoadCursorW.Call(0, 32512)
	// Icon resource ID 1 is embedded in resource_windows_amd64.syso.
	icon, _, _ := pLoadIconW.Call(inst, 1)
	if icon == 0 {
		icon, _, _ = pLoadIconW.Call(0, 32512)
	}
	wc := wndClass{WndProc: syscall.NewCallback(wndProc), Instance: inst, Icon: icon, Cursor: cursor, Background: 0, ClassName: cls}
	atom, _, e1 := pRegisterClassW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return fmt.Errorf("RegisterClassW failed: %v", e1)
	}

	gui.font = createFont("Segoe UI", -14, 400)
	gui.fontTitle = createFont("Segoe UI", -27, 600)
	gui.fontSection = createFont("Segoe UI", -17, 600)
	gui.fontLabel = createFont("Segoe UI", -14, 600)
	gui.fontSmall = createFont("Segoe UI", -12, 400)
	gui.fontMono = createFont("Consolas", -17, 400)
	gui.brushMain, _, _ = pCreateSolidBrush.Call(uintptr(clrMain))
	gui.brushWhite, _, _ = pCreateSolidBrush.Call(uintptr(clrWhite))
	gui.brushPreview, _, _ = pCreateSolidBrush.Call(uintptr(clrPreviewBG))
	gui.brushLog, _, _ = pCreateSolidBrush.Call(uintptr(clrLogBG))

	sw, _, _ := pGetSystemMetrics.Call(0)
	sh, _, _ := pGetSystemMetrics.Call(1)
	width, height := 936, 742
	x := (int(sw) - width) / 2
	y := (int(sh) - height) / 2
	style := uint32(wsOverlapped | wsCaption | wsSysMenu | wsMinimizeBox)
	hwnd, _, e2 := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(ptr(appName+" "+appVersion))), uintptr(style), uintptr(x), uintptr(y), uintptr(width), uintptr(height), 0, 0, inst, 0)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW failed: %v", e2)
	}
	gui.hwnd = hwnd

	// Header
	title := static(appName, 24, 18, 500, 34, gui.fontTitle)
	setColor(title, clrText)
	sub := static("F70/F100 CDU Bridge  ·  WinWing / MobiFlight", 26, 54, 520, 20, gui.font)
	setColor(sub, clrMuted)
	gui.headerStatus = static("BRIDGE STOPPED", 650, 31, 245, 24, gui.fontLabel)
	setColor(gui.headerStatus, clrGray)

	// Status card
	sec := static("System status", 38, 124, 220, 26, gui.fontSection)
	setColor(sec, clrText)
	statusRows := []struct {
		y       int
		label   string
		dot     *uintptr
		value   *uintptr
		initial string
	}{
		{160, "Microsoft Flight Simulator 2024", &gui.dotMSFS, &gui.statusMSFS, "Checking..."},
		{196, "Just Flight F70/F100", &gui.dotF100, &gui.statusF100, "Waiting for aircraft"},
		{232, "MobiFlight / WinCtrl", &gui.dotMF, &gui.statusMF, "Not connected"},
		{268, "Bridge", &gui.dotBridge, &gui.statusBridge, "Bridge stopped"},
	}
	for _, row := range statusRows {
		*row.dot = static("●", 40, row.y-2, 20, 24, gui.fontLabel)
		setColor(*row.dot, clrGray)
		lbl := static(row.label, 66, row.y, 228, 20, gui.fontLabel)
		setColor(lbl, clrText)
		*row.value = static(row.initial, 300, row.y, 225, 20, gui.font)
		setColor(*row.value, clrMuted)
	}
	gui.statusBuild = static("", 66, 292, 455, 16, gui.fontSmall)
	setColor(gui.statusBuild, clrMuted)

	// Controls card
	sec = static("Bridge control", 588, 124, 220, 26, gui.fontSection)
	setColor(sec, clrText)
	lbl := static("Output CDU", 590, 166, 100, 20, gui.fontLabel)
	setColor(lbl, clrText)
	gui.cdu = createControl(0, "COMBOBOX", "", wsChild|wsVisible|wsTabStop|cbsDropDownList, 590, 190, 288, 200, idCdu, gui.font)
	send(gui.cdu, cbAddString, 0, uintptr(unsafe.Pointer(ptr("Captain"))))
	send(gui.cdu, cbAddString, 0, uintptr(unsafe.Pointer(ptr("First Officer"))))
	gui.auto = createControl(0, "BUTTON", "Connect automatically when aircraft is loaded", wsChild|wsVisible|wsTabStop|bsAutoCheckBox, 590, 228, 288, 25, idAuto, gui.font)
	gui.start = createControl(0, "BUTTON", "Start bridge", wsChild|wsVisible|wsTabStop|bsOwnerDraw, 590, 265, 137, 34, idStart, gui.fontLabel)
	gui.stop = createControl(0, "BUTTON", "Stop", wsChild|wsVisible|wsTabStop|bsOwnerDraw, 741, 265, 137, 34, idStop, gui.fontLabel)

	// Preview card
	sec = static("CDU preview", 38, 346, 190, 26, gui.fontSection)
	setColor(sec, clrText)
	gui.previewCheck = createControl(0, "BUTTON", "Live preview", wsChild|wsVisible|wsTabStop|bsAutoCheckBox, 445, 346, 125, 24, idPreviewCheck, gui.font)
	gui.preview = createControl(0, "EDIT", "", wsChild|wsVisible|esMultiline|esReadOnly, 40, 382, 530, 252, 0, gui.fontMono)
	setColor(gui.preview, clrPreviewFG)

	// Activity card
	sec = static("Activity", 628, 346, 160, 26, gui.fontSection)
	setColor(sec, clrText)
	gui.openLogs = createControl(0, "BUTTON", "Open logs", wsChild|wsVisible|wsTabStop|bsOwnerDraw, 785, 342, 94, 31, idOpenLogs, gui.fontLabel)
	gui.logBox = createControl(0, "EDIT", "", wsChild|wsVisible|wsVScroll|esMultiline|esAutoVScroll|esReadOnly, 628, 382, 252, 205, 0, gui.fontSmall)
	gui.errorLabel = static("", 628, 600, 252, 42, gui.fontSmall)
	setColor(gui.errorLabel, clrMuted)

	// Footer
	foot := static("READ-ONLY  ·  No aircraft files are modified", 24, 684, 440, 22, gui.fontSmall)
	setColor(foot, clrMuted)
	copyright := static(appCopyright, 565, 684, 235, 22, gui.fontSmall)
	setColor(copyright, clrMuted)
	ver := static("v"+appVersion, 820, 684, 80, 22, gui.fontSmall)
	setColor(ver, clrMuted)

	snap := e.snapshot()
	gui.last = statusSnapshot{}
	if snap.CDU == "copilot" {
		send(gui.cdu, cbSetCurSel, 1, 0)
	} else {
		send(gui.cdu, cbSetCurSel, 0, 0)
	}
	setChecked(gui.auto, snap.AutoStart)
	setChecked(gui.previewCheck, snap.ShowPreview)
	pSetTimer.Call(hwnd, idTimer, 500, 0)
	updateGUI()
	pShowWindow.Call(hwnd, swShow)
	pUpdateWindow.Call(hwnd)
	return nil
}

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	e, err := newBridgeEngine()
	if err != nil {
		showFatal(err)
		return
	}
	if err := initGUI(e); err != nil {
		e.Stop()
		showFatal(err)
		return
	}
	var m msg
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	if gui != nil {
		for _, h := range []uintptr{gui.font, gui.fontTitle, gui.fontSection, gui.fontLabel, gui.fontSmall, gui.fontMono, gui.brushMain, gui.brushWhite, gui.brushPreview, gui.brushLog} {
			if h != 0 {
				pDeleteObject.Call(h)
			}
		}
	}
}

func showFatal(err error) {
	exe, _ := os.Executable()
	p := filepath.Join(filepath.Dir(exe), "startup-error.txt")
	_ = os.WriteFile(p, []byte(err.Error()), 0644)
	msg := ptr(appName + " could not start:\r\n\r\n" + err.Error() + "\r\n\r\nDetails were written to startup-error.txt.")
	title := ptr(appName)
	pMessageBoxW.Call(0, uintptr(unsafe.Pointer(msg)), uintptr(unsafe.Pointer(title)), 0x10)
}

//go:build windows

package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	modUser32 = syscall.NewLazyDLL("user32.dll")
	modGdi32  = syscall.NewLazyDLL("gdi32.dll")
	modKernel = syscall.NewLazyDLL("kernel32.dll")

	pRegisterClassExW   = modUser32.NewProc("RegisterClassExW")
	pCreateWindowExW    = modUser32.NewProc("CreateWindowExW")
	pDefWindowProcW     = modUser32.NewProc("DefWindowProcW")
	pShowWindow         = modUser32.NewProc("ShowWindow")
	pUpdateWindow       = modUser32.NewProc("UpdateWindow")
	pGetMessageW        = modUser32.NewProc("GetMessageW")
	pTranslateMessage   = modUser32.NewProc("TranslateMessage")
	pDispatchMessageW   = modUser32.NewProc("DispatchMessageW")
	pPostQuitMessage    = modUser32.NewProc("PostQuitMessage")
	pBeginPaint         = modUser32.NewProc("BeginPaint")
	pEndPaint           = modUser32.NewProc("EndPaint")
	pGetClientRect      = modUser32.NewProc("GetClientRect")
	pInvalidateRect     = modUser32.NewProc("InvalidateRect")
	pSetWindowTextW     = modUser32.NewProc("SetWindowTextW")
	pMessageBoxW        = modUser32.NewProc("MessageBoxW")
	pLoadCursorW        = modUser32.NewProc("LoadCursorW")
	pLoadImageW         = modUser32.NewProc("LoadImageW")
	pSendMessageW       = modUser32.NewProc("SendMessageW")
	pGetSystemMetrics   = modUser32.NewProc("GetSystemMetrics")
	pSetFocus           = modUser32.NewProc("SetFocus")
	pEnableWindow       = modUser32.NewProc("EnableWindow")
	pDestroyWindow      = modUser32.NewProc("DestroyWindow")
	pSetProcessDPIAware = modUser32.NewProc("SetProcessDPIAware")
	pGetKeyState        = modUser32.NewProc("GetKeyState")

	pCreateSolidBrush = modGdi32.NewProc("CreateSolidBrush")
	pCreatePen        = modGdi32.NewProc("CreatePen")
	pCreateFontW      = modGdi32.NewProc("CreateFontW")
	pDeleteObject     = modGdi32.NewProc("DeleteObject")
	pSelectObject     = modGdi32.NewProc("SelectObject")
	pSetTextColor     = modGdi32.NewProc("SetTextColor")
	pSetBkMode        = modGdi32.NewProc("SetBkMode")
	pFillRect         = modUser32.NewProc("FillRect")
	pRectangle        = modGdi32.NewProc("Rectangle")
	pRoundRect        = modGdi32.NewProc("RoundRect")
	pMoveToEx         = modGdi32.NewProc("MoveToEx")
	pLineTo           = modGdi32.NewProc("LineTo")
	pEllipse          = modGdi32.NewProc("Ellipse")
	pArc              = modGdi32.NewProc("Arc")
	pDrawTextW        = modUser32.NewProc("DrawTextW")
	pSetPixel         = modGdi32.NewProc("SetPixel")
	pGetModuleHandleW = modKernel.NewProc("GetModuleHandleW")
)

const (
	WM_DESTROY       = 0x0002
	WM_PAINT         = 0x000F
	WM_CLOSE         = 0x0010
	WM_COMMAND       = 0x0111
	WM_LBUTTONUP     = 0x0202
	WM_MOUSEMOVE     = 0x0200
	WM_KEYDOWN       = 0x0100
	WM_CHAR          = 0x0102
	WM_GETMINMAXINFO = 0x0024
	WM_SETICON       = 0x0080

	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_BORDER           = 0x00800000
	WS_TABSTOP          = 0x00010000
	ES_AUTOHSCROLL      = 0x0080
	BS_PUSHBUTTON       = 0x00000000

	SW_SHOW         = 5
	CW_USEDEFAULT   = ^uintptr(0x7fffffff)
	IDC_ARROW       = 32512
	IMAGE_ICON      = 1
	ICON_SMALL      = 0
	ICON_BIG        = 1
	LR_LOADFROMFILE = 0x0010
	LR_DEFAULTSIZE  = 0x0040

	DT_LEFT         = 0x00000000
	DT_CENTER       = 0x00000001
	DT_RIGHT        = 0x00000002
	DT_VCENTER      = 0x00000004
	DT_SINGLELINE   = 0x00000020
	DT_WORDBREAK    = 0x00000010
	DT_END_ELLIPSIS = 0x00008000

	TRANSPARENT = 1
	PS_SOLID    = 0

	VK_ESCAPE  = 0x1B
	VK_F1      = 0x70
	VK_CONTROL = 0x11
)

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type point struct{ X, Y int32 }
type msg struct {
	HWnd           uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             point
	LPrivate       uint32
}
type rect struct{ Left, Top, Right, Bottom int32 }
type minMaxInfo struct {
	PtReserved     point
	PtMaxSize      point
	PtMaxPosition  point
	PtMinTrackSize point
	PtMaxTrackSize point
}
type paintStruct struct {
	Hdc                  uintptr
	FErase               int32
	RcPaint              rect
	FRestore, FIncUpdate int32
	RgbReserved          [32]byte
}

type hitBox struct {
	R  rect
	ID string
}

type nativeState struct {
	hwnd           uintptr
	screen         string
	current        *Empresa
	cat            Catalogo
	sc             CatalogoInsumos
	hits           []hitBox
	hover          string
	selectedSector int
	selectedType   int
	selectedModel  string
	last           Registro
	status         string
	tutorStatus    string
	lastSavedAt    string
}

var ns nativeState

func rgb(r, g, b byte) uintptr { return uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16 }

var (
	colBg      = rgb(250, 250, 250)
	colPanel   = rgb(255, 255, 255)
	colInk     = rgb(16, 22, 28)
	colMuted   = rgb(92, 92, 92)
	colLine    = rgb(190, 190, 190)
	colGrid    = rgb(238, 238, 238)
	colCyan    = rgb(62, 214, 220)
	colBlue    = rgb(58, 64, 70)
	colMagenta = rgb(239, 36, 109)
	colAmber   = rgb(242, 170, 42)
	colGreen   = rgb(45, 45, 45)
	colDark    = rgb(37, 45, 53)
)

func ptr(s string) *uint16   { p, _ := syscall.UTF16PtrFromString(s); return p }
func loword(v uintptr) int32 { return int32(uint16(v & 0xffff)) }
func hiword(v uintptr) int32 { return int32(uint16((v >> 16) & 0xffff)) }

func loadIconFromFile(path string, cx, cy int) uintptr {
	if strings.TrimSpace(path) == "" {
		return 0
	}
	r, _, _ := pLoadImageW.Call(0, uintptr(unsafe.Pointer(ptr(path))), IMAGE_ICON, uintptr(cx), uintptr(cy), LR_LOADFROMFILE|LR_DEFAULTSIZE)
	return r
}

func appIconPath() string {
	candidates := []string{}
	if ex, err := os.Executable(); err == nil {
		dir := filepath.Dir(ex)
		candidates = append(candidates, filepath.Join(dir, "jed_icon.ico"))
	}
	candidates = append(candidates, "jed_icon.ico")
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func appIcons() (uintptr, uintptr) {
	iconPath := appIconPath()
	if iconPath == "" {
		return 0, 0
	}
	big := loadIconFromFile(iconPath, 32, 32)
	small := loadIconFromFile(iconPath, 16, 16)
	if small == 0 {
		small = big
	}
	return big, small
}

func runNativeUI() error {
	pSetProcessDPIAware.Call()
	if err := ensureDirs(); err != nil {
		return err
	}
	c, err := loadCatalog()
	if err != nil {
		return err
	}
	sc, err := loadSupplyCatalog()
	if err != nil {
		return err
	}
	ns.cat, ns.sc, ns.screen = c, sc, "home"
	loadOnlineConfig()

	hInst, _, _ := pGetModuleHandleW.Call(0)
	className := ptr("JEDNativeOrbitClass")
	cursor, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
	bigIcon, smallIcon := appIcons()
	wc := wndClassEx{CbSize: uint32(unsafe.Sizeof(wndClassEx{})), LpfnWndProc: syscall.NewCallback(nativeWndProc), HInstance: hInst, HIcon: bigIcon, HCursor: cursor, LpszClassName: className, HIconSm: smallIcon}
	if r, _, e := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return fmt.Errorf("RegisterClassExW: %v", e)
	}

	sw, _, _ := pGetSystemMetrics.Call(0)
	sh, _, _ := pGetSystemMetrics.Call(1)
	w, h := int32(1360), int32(860)
	if int32(sw) < 1450 {
		w = int32(sw) - 60
	}
	if int32(sh) < 940 {
		h = int32(sh) - 60
	}
	if w < 1100 {
		w = 1100
	}
	if h < 720 {
		h = 720
	}
	x, y := (int32(sw)-w)/2, (int32(sh)-h)/2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	hwnd, _, e := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(ptr("JED Simulador — Órbita Clean"))), WS_OVERLAPPEDWINDOW,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h), 0, 0, hInst, 0)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW: %v", e)
	}
	ns.hwnd = hwnd
	if bigIcon != 0 {
		pSendMessageW.Call(hwnd, WM_SETICON, ICON_BIG, bigIcon)
	}
	if smallIcon != 0 {
		pSendMessageW.Call(hwnd, WM_SETICON, ICON_SMALL, smallIcon)
	}
	pShowWindow.Call(hwnd, SW_SHOW)
	pUpdateWindow.Call(hwnd)

	var m msg
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	return nil
}

func nativeWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case WM_GETMINMAXINFO:
		mmi := (*minMaxInfo)(unsafe.Pointer(lParam))
		mmi.PtMinTrackSize = point{X: 1100, Y: 720}
		return 0
	case WM_PAINT:
		var ps paintStruct
		hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		drawNative(hdc, hwnd)
		pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case WM_LBUTTONUP:
		x, y := loword(lParam), hiword(lParam)
		for _, h := range ns.hits {
			if inRect(x, y, h.R) {
				handleHit(h.ID)
				break
			}
		}
		return 0
	case WM_MOUSEMOVE:
		x, y := loword(lParam), hiword(lParam)
		hov := ""
		for _, h := range ns.hits {
			if inRect(x, y, h.R) {
				hov = h.ID
				break
			}
		}
		if hov != ns.hover {
			ns.hover = hov
			invalidate()
		}
		return 0
	case WM_KEYDOWN:
		if wParam == VK_F1 {
			nativeInfo("Ajuda rápida", "F1 abre esta ajuda.\n\nESC volta à tela anterior.\nCTRL+S salva a empresa atual.\n\nNo JED, ciano indica operação normal; magenta indica alerta ou risco.")
			return 0
		}
		if wParam == 'S' || wParam == 's' {
			ctrl, _, _ := pGetKeyState.Call(VK_CONTROL)
			if int16(ctrl) < 0 && ns.current != nil {
				if _, err := saveCompany(ns.current); err == nil {
					ns.status = "SALVO • " + time.Now().Format("15:04")
				} else {
					ns.status = "ERRO AO SALVAR • " + err.Error()
				}
				invalidate()
				return 0
			}
		}
		if wParam == VK_ESCAPE {
			if ns.screen == "dashboard" || ns.screen == "catalog" || ns.screen == "saved" || ns.screen == "tutor" {
				ns.screen = "home"
			} else if ns.screen != "home" {
				ns.screen = "dashboard"
			} else {
				pDestroyWindow.Call(hwnd)
			}
			invalidate()
			return 0
		}
	case WM_CLOSE:
		pDestroyWindow.Call(hwnd)
		return 0
	case WM_DESTROY:
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return r
}

func inRect(x, y int32, r rect) bool {
	return x >= r.Left && x <= r.Right && y >= r.Top && y <= r.Bottom
}
func invalidate()              { pInvalidateRect.Call(ns.hwnd, 0, 1) }
func addHit(id string, r rect) { ns.hits = append(ns.hits, hitBox{r, id}) }

func brush(c uintptr) uintptr { b, _, _ := pCreateSolidBrush.Call(c); return b }
func pen(c uintptr, width int32) uintptr {
	p, _, _ := pCreatePen.Call(PS_SOLID, uintptr(width), c)
	return p
}
func font(size int32, weight int32, face string) uintptr {
	f, _, _ := pCreateFontW.Call(uintptr(-size), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 0, 0, uintptr(unsafe.Pointer(ptr(face))))
	return f
}
func withObj(hdc, obj uintptr, fn func()) {
	old, _, _ := pSelectObject.Call(hdc, obj)
	fn()
	pSelectObject.Call(hdc, old)
	pDeleteObject.Call(obj)
}
func fill(hdc uintptr, r rect, c uintptr) {
	b := brush(c)
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&r)), b)
	pDeleteObject.Call(b)
}
func line(hdc uintptr, x1, y1, x2, y2 int32, c uintptr, w int32) {
	withObj(hdc, pen(c, w), func() { pMoveToEx.Call(hdc, uintptr(x1), uintptr(y1), 0); pLineTo.Call(hdc, uintptr(x2), uintptr(y2)) })
}
func outline(hdc uintptr, r rect, c uintptr, w int32) {
	withObj(hdc, pen(c, w), func() {
		withObj(hdc, brush(colPanel), func() { pRectangle.Call(hdc, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom)) })
	})
}
func text(hdc uintptr, s string, r rect, size int32, c uintptr, align uint32, bold bool) {
	weight := int32(400)
	if bold {
		weight = 700
	}
	f := font(size, weight, "Segoe UI")
	old, _, _ := pSelectObject.Call(hdc, f)
	pSetTextColor.Call(hdc, c)
	pSetBkMode.Call(hdc, TRANSPARENT)
	flags := uintptr(DT_SINGLELINE | DT_VCENTER | align | DT_END_ELLIPSIS)
	u, _ := syscall.UTF16FromString(s)
	pDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&r)), flags)
	pSelectObject.Call(hdc, old)
	pDeleteObject.Call(f)
}
func multiText(hdc uintptr, s string, r rect, size int32, c uintptr, bold bool) {
	weight := int32(400)
	if bold {
		weight = 700
	}
	f := font(size, weight, "Segoe UI")
	old, _, _ := pSelectObject.Call(hdc, f)
	pSetTextColor.Call(hdc, c)
	pSetBkMode.Call(hdc, TRANSPARENT)
	u, _ := syscall.UTF16FromString(s)
	pDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&r)), uintptr(DT_LEFT|DT_WORDBREAK))
	pSelectObject.Call(hdc, old)
	pDeleteObject.Call(f)
}

func drawNative(hdc, hwnd uintptr) {
	var cr rect
	pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&cr)))
	fill(hdc, cr, colBg)
	ns.hits = nil
	drawTechnicalGrid(hdc, cr)
	switch ns.screen {
	case "home":
		drawHome(hdc, cr)
	case "catalog":
		drawCatalog(hdc, cr)
	case "dashboard":
		drawDashboard(hdc, cr)
	case "inventory":
		drawInventory(hdc, cr)
	case "finance":
		drawFinance(hdc, cr)
	case "lean":
		drawLean(hdc, cr)
	case "persona":
		drawPersona(hdc, cr)
	case "channels":
		drawChannels(hdc, cr)
	case "tools":
		drawTools(hdc, cr)
	case "online":
		drawOnline(hdc, cr)
	case "journey":
		drawJourney(hdc, cr)
	case "indicators":
		drawIndicators(hdc, cr)
	case "tutor":
		drawTutor(hdc, cr)
	case "saved":
		drawSaved(hdc, cr)
	default:
		drawHome(hdc, cr)
	}
}

func drawTechnicalGrid(hdc uintptr, cr rect) {
	for x := int32(0); x < cr.Right; x += 120 {
		line(hdc, x, 0, x, cr.Bottom, colGrid, 1)
	}
	for y := int32(0); y < cr.Bottom; y += 90 {
		line(hdc, 0, y, cr.Right, y, colGrid, 1)
	}
	// faint diagnostic circles
	withObj(hdc, pen(colGrid, 1), func() {
		negY := int32(-150)
		pArc.Call(hdc, uintptr(cr.Right-370), uintptr(uint32(negY)), uintptr(cr.Right+150), uintptr(370), uintptr(cr.Right), 0, uintptr(cr.Right), 0)
	})
}

func drawChrome(hdc uintptr, cr rect, title, sub string) {
	fill(hdc, rect{0, 0, cr.Right, 56}, colInk)
	text(hdc, "JED", rect{28, 0, 100, 56}, 24, rgb(255, 255, 255), DT_LEFT, true)
	text(hdc, "BUSINESS SIMULATION / ORBITA CLEAN", rect{100, 0, 500, 56}, 11, rgb(210, 220, 228), DT_LEFT, true)
	text(hdc, "SYS 2.0 RC1.8", rect{cr.Right - 210, 0, cr.Right - 28, 56}, 11, colCyan, DT_RIGHT, true)
	text(hdc, "F1 AJUDA  •  CTRL+S SALVAR", rect{cr.Right - 410, 0, cr.Right - 220, 56}, 9, rgb(160, 178, 190), DT_RIGHT, false)
	text(hdc, title, rect{32, 66, cr.Right - 32, 104}, 24, colInk, DT_LEFT, true)
	text(hdc, strings.ToUpper(sub), rect{32, 101, cr.Right - 32, 126}, 10, colMuted, DT_LEFT, true)
	line(hdc, 32, 132, cr.Right-32, 132, colInk, 2)
	line(hdc, 32, 136, cr.Right-32, 136, colCyan, 1)
}

func button(hdc uintptr, id, label, sub string, r rect, accent uintptr) {
	bg := colPanel
	if ns.hover == id {
		bg = rgb(245, 245, 245)
	}
	fill(hdc, r, bg)
	line(hdc, r.Left, r.Top, r.Right, r.Top, accent, 4)
	line(hdc, r.Left, r.Bottom, r.Right, r.Bottom, colLine, 1)
	line(hdc, r.Left, r.Top, r.Left, r.Bottom, colLine, 1)
	line(hdc, r.Right, r.Top, r.Right, r.Bottom, colLine, 1)
	text(hdc, label, rect{r.Left + 18, r.Top + 12, r.Right - 18, r.Top + 34}, 13, colInk, DT_LEFT, true)
	multiText(hdc, sub, rect{r.Left + 18, r.Top + 40, r.Right - 18, r.Bottom - 10}, 9, colMuted, false)
	addHit(id, r)
}

func drawHome(hdc uintptr, cr rect) {
	drawChrome(hdc, cr, "PAINEL DE INICIALIZAÇÃO", "ambiente de simulação empresarial")
	// Hero analytic dial
	cx, cy := int32(300), int32(400)
	rad := int32(170)
	withObj(hdc, pen(colLine, 2), func() { pEllipse.Call(hdc, uintptr(cx-rad), uintptr(cy-rad), uintptr(cx+rad), uintptr(cy+rad)) })
	withObj(hdc, pen(colCyan, 8), func() {
		pEllipse.Call(hdc, uintptr(cx-rad+14), uintptr(cy-rad+14), uintptr(cx+rad-14), uintptr(cy+rad-14))
	})
	withObj(hdc, pen(colBlue, 8), func() {
		pEllipse.Call(hdc, uintptr(cx-rad+30), uintptr(cy-rad+30), uintptr(cx+rad-30), uintptr(cy+rad-30))
	})
	text(hdc, "JED", rect{cx - 120, cy - 55, cx + 120, cy + 10}, 48, colInk, DT_CENTER, true)
	text(hdc, "EMPREENDEDORISMO EM JOGO", rect{cx - 150, cy + 15, cx + 150, cy + 45}, 10, colMuted, DT_CENTER, true)
	text(hdc, "READY", rect{cx - 80, cy + 64, cx + 80, cy + 92}, 14, colCyan, DT_CENTER, true)

	x := int32(560)
	y := int32(190)
	bw := cr.Right - x - 60
	bh := int32(78)
	bh = 66
	button(hdc, "online", "JED ONLINE", onlineHomeSubtitle(), rect{x, y, x + bw, y + bh}, colCyan)
	y += 78
	button(hdc, "new", "NOVO EMPREENDIMENTO", "Modo local/offline: crie uma nova empresa.", rect{x, y, x + bw, y + bh}, colInk)
	y += 78
	button(hdc, "load", "CONTINUAR SIMULAÇÃO", "Abra uma empresa salva neste computador.", rect{x, y, x + bw, y + bh}, colBlue)
	y += 78
	button(hdc, "tutor", "MODO TUTOR LOCAL", "Cenários e turmas deste computador.", rect{x, y, x + bw, y + bh}, colInk)
	y += 78
	button(hdc, "tutorial", "TUTORIAL", "Guia rápido dos conceitos essenciais.", rect{x, y, x + bw, y + bh}, colAmber)
	y += 78
	button(hdc, "exit", "SAIR", "Encerrar o JED Simulador.", rect{x, y, x + bw, y + bh}, colMagenta)
}

func onlineHomeSubtitle() string {
	if onlineConfig.Token != "" {
		role := strings.ToUpper(onlineConfig.Role)
		if onlineConfig.Role == "tutor" || onlineConfig.Role == "mentor" {
			role = "MENTOR"
		} else if onlineConfig.Role == "admin" {
			role = "ADMINISTRADOR"
		}
		return fmt.Sprintf("%s • %s • %s", role, onlineConfig.Name, onlineConfig.ServerURL)
	}
	return "Contas são cadastradas por Administradores. Use seu e-mail e a senha temporária recebida."
}

func drawOnline(hdc uintptr, cr rect) {
	drawChrome(hdc, cr, "JED ONLINE", "contas • turmas • administração • sincronização")
	if onlineConfig.Token == "" {
		text(hdc, "NENHUMA SESSÃO ATIVA", rect{60, 175, cr.Right - 60, 220}, 22, colInk, DT_LEFT, true)
		multiText(hdc, "Contas de Aluno, Mentor e Administrador são cadastradas exclusivamente por Administradores. No primeiro acesso, use a senha temporária recebida por e-mail e substitua-a por uma senha pessoal.", rect{60, 225, cr.Right - 60, 300}, 11, colMuted, false)
		button(hdc, "online:login", "ENTRAR", "E-mail e senha de uma conta cadastrada pelo Administrador.", rect{60, 325, 520, 400}, colCyan)
		button(hdc, "online:server", "CONFIGURAR SERVIDOR", "Alterar somente o endereço do servidor.", rect{540, 325, 1000, 400}, colInk)
		backButton(hdc, cr)
		return
	}

	role := strings.ToUpper(onlineConfig.Role)
	if onlineConfig.Role == "tutor" || onlineConfig.Role == "mentor" {
		role = "MENTOR"
	} else if onlineConfig.Role == "admin" {
		role = "ADMINISTRADOR"
	}
	title := role + " / " + onlineConfig.Name
	if onlineConfig.Role == "admin" && onlineConfig.IsPrimaryAdmin {
		title += " • PRINCIPAL"
	}
	text(hdc, title, rect{60, 170, cr.Right - 60, 215}, 22, colInk, DT_LEFT, true)
	text(hdc, onlineConfig.Email+"  •  "+onlineConfig.ServerURL, rect{60, 215, cr.Right - 60, 245}, 10, colMuted, DT_LEFT, false)

	y := int32(285)
	if onlineConfig.Role == "aluno" {
		button(hdc, "online:sync", "SINCRONIZAR EMPRESA ATUAL", "Enviar a empresa aberta para o servidor.", rect{60, y, 530, y + 74}, colCyan)
		y += 90
		button(hdc, "online:join", "ENTRAR EM TURMA", "Usar o código fornecido pelo mentor.", rect{60, y, 530, y + 74}, colInk)
		y += 90
		button(hdc, "online:mycompanies", "MINHAS EMPRESAS ONLINE", "Consultar versões armazenadas no servidor.", rect{60, y, 530, y + 74}, colInk)
		button(hdc, "online:classes", "MINHAS TURMAS", "Consultar turmas vinculadas à conta.", rect{550, 285, 1020, 359}, colInk)
	} else if onlineConfig.Role == "tutor" || onlineConfig.Role == "mentor" {
		left := int32(60)
		right := cr.Right - 60
		gap := int32(18)
		cw := (right - left - 2*gap) / 3
		x1 := left
		x2 := x1 + cw + gap
		x3 := x2 + cw + gap

		button(hdc, "online:createclass", "CRIAR TURMA", "Criar turma e cenário.", rect{x1, 285, x1 + cw, 359}, colCyan)
		button(hdc, "online:classes", "MINHAS TURMAS", "Listar turmas vinculadas.", rect{x2, 285, x2 + cw, 359}, colInk)
		button(hdc, "online:classcompanies", "RESULTADOS", "Ver empresas sincronizadas.", rect{x3, 285, x3 + cw, 359}, colInk)

		multiText(hdc, "O cadastro de Alunos, Mentores e Administradores é exclusivo do painel de Administração. O Mentor organiza turmas e acompanha resultados.", rect{x1, 385, right, 455}, 11, colMuted, false)
	} else if onlineConfig.Role == "admin" {
		left := int32(60)
		right := cr.Right - 60
		gap := int32(18)
		cw := (right - left - 2*gap) / 3
		x1 := left
		x2 := x1 + cw + gap
		x3 := x2 + cw + gap

		button(hdc, "online:adminusers", "USUÁRIOS", "Listar contas e estado.", rect{x1, 285, x1 + cw, 359}, colCyan)
		button(hdc, "online:admincreate", "CRIAR ADMIN", "Cria com a senha temporária padrão.", rect{x2, 285, x2 + cw, 359}, colInk)
		button(hdc, "online:adminstatus", "ATIVAR / DESATIVAR", "Controlar acesso de uma conta.", rect{x3, 285, x3 + cw, 359}, colInk)
		multiText(hdc, "Para cadastrar Alunos e Mentores, e para importação CSV, use o painel Web de Administração.", rect{x1, 390, right, 445}, 11, colMuted, false)
		if onlineConfig.IsPrimaryAdmin {
			button(hdc, "online:admintransfer", "TRANSFERIR PRINCIPAL", "Definir outro Admin como principal.", rect{x1, 465, x1 + cw, 539}, colMagenta)
		}
	}

	button(hdc, "online:password", "ALTERAR SENHA", "Trocar a senha desta conta.", rect{cr.Right - 810, cr.Bottom - 100, cr.Right - 450, cr.Bottom - 35}, colInk)
	button(hdc, "online:logout", "SAIR DA CONTA", "Encerrar esta sessão neste computador.", rect{cr.Right - 430, cr.Bottom - 100, cr.Right - 60, cr.Bottom - 35}, colMagenta)
	backButton(hdc, cr)
}

func nativeOnlineRegisterStudent() {
	server := onlineConfig.ServerURL
	if server == "" {
		server = "http://127.0.0.1:8787"
	}
	server, ok := nativePrompt("CADASTRAR ALUNO", "Endereço do servidor", server)
	if !ok {
		return
	}
	name, ok := nativePrompt("CADASTRAR ALUNO", "Seu nome", "")
	if !ok {
		return
	}
	email, ok := nativePrompt("CADASTRAR ALUNO", "Seu e-mail", "")
	if !ok {
		return
	}
	inst, ok := nativePrompt("CADASTRAR ALUNO", "Identificador institucional (opcional)", "")
	if !ok {
		inst = ""
	}
	pass, ok := nativePromptSecret("CADASTRAR ALUNO", "Crie sua senha (mínimo 8 caracteres)")
	if !ok {
		return
	}
	confirm, ok := nativePromptSecret("CADASTRAR ALUNO", "Repita sua senha")
	if !ok {
		return
	}
	if pass != confirm {
		nativeInfo("Cadastro", "As senhas não coincidem.")
		return
	}
	if err := onlineRegisterStudent(server, name, email, pass, inst); err != nil {
		nativeInfo("Falha no cadastro", err.Error())
		return
	}
	nativeInfo("Conta criada", "Conta de ALUNO criada com sucesso.\nAgora use ENTRAR EM TURMA para informar o código recebido do mentor.")
}

func nativeOnlineRegisterMentor() {
	server := onlineConfig.ServerURL
	if server == "" {
		server = "http://127.0.0.1:8787"
	}
	server, ok := nativePrompt("CADASTRAR MENTOR", "Endereço do servidor", server)
	if !ok {
		return
	}
	name, ok := nativePrompt("CADASTRAR MENTOR", "Seu nome", "")
	if !ok {
		return
	}
	email, ok := nativePrompt("CADASTRAR MENTOR", "Seu e-mail", "")
	if !ok {
		return
	}
	institution, ok := nativePrompt("CADASTRAR MENTOR", "Instituição", "")
	if !ok {
		return
	}
	instID, ok := nativePrompt("CADASTRAR MENTOR", "Identificador institucional (opcional)", "")
	if !ok {
		instID = ""
	}
	code, ok := nativePrompt("CADASTRAR MENTOR", "Código de credenciamento MTR-XXXX-XXXX", "")
	if !ok {
		return
	}
	pass, ok := nativePromptSecret("CADASTRAR MENTOR", "Crie sua senha (mínimo 8 caracteres)")
	if !ok {
		return
	}
	confirm, ok := nativePromptSecret("CADASTRAR MENTOR", "Repita sua senha")
	if !ok {
		return
	}
	if pass != confirm {
		nativeInfo("Cadastro", "As senhas não coincidem.")
		return
	}
	if err := onlineRegisterMentor(server, name, email, pass, institution, instID, code); err != nil {
		nativeInfo("Falha no cadastro", err.Error())
		return
	}
	nativeInfo("Conta criada", "Conta de MENTOR criada e credencial consumida com sucesso.")
}

func nativeOnlineLogin() {
	server := onlineConfig.ServerURL
	if server == "" {
		server = "http://127.0.0.1:8787"
	}
	server, ok := nativePrompt("JED ONLINE", "Endereço do servidor (https://... ou http://...)", server)
	if !ok {
		return
	}
	email, ok := nativePrompt("JED ONLINE", "E-mail", "")
	if !ok {
		return
	}
	password, ok := nativePrompt("JED ONLINE", "Senha (mínimo 8 caracteres)", "")
	if !ok {
		return
	}
	if err := onlineLogin(server, email, password); err != nil {
		nativeInfo("Falha no login", err.Error())
		return
	}
	if onlineConfig.MustChangePassword {
		nativeInfo("PRIMEIRO ACESSO", "Esta conta usa uma senha temporária. Defina agora uma nova senha pessoal.")
		next, ok := nativePromptSecret("PRIMEIRO ACESSO", "Nova senha (mínimo 8 caracteres)")
		if !ok {
			nativeInfo("Primeiro acesso", "A troca de senha é obrigatória antes de usar o JED Online.")
			return
		}
		confirm, ok := nativePromptSecret("PRIMEIRO ACESSO", "Repita a nova senha")
		if !ok || next != confirm {
			nativeInfo("Primeiro acesso", "As novas senhas não coincidem.")
			return
		}
		if err := onlineChangePassword(password, next); err != nil {
			nativeInfo("Primeiro acesso", err.Error())
			return
		}
	}
	role := strings.ToUpper(onlineConfig.Role)
	if onlineConfig.Role == "tutor" || onlineConfig.Role == "mentor" {
		role = "MENTOR"
	} else if onlineConfig.Role == "admin" {
		role = "ADMINISTRADOR"
	}
	nativeInfo("JED Online", "Conectado como "+role+": "+onlineConfig.Name)
}

func onlineClassesText(cs []OnlineClass) string {
	if len(cs) == 0 {
		return "Nenhuma turma encontrada."
	}
	var b strings.Builder
	for _, c := range cs {
		fmt.Fprintf(&b, "%s\nCódigo: %s\nID: %s\nAlunos: %d\nCenário: %s\n\n", c.Name, c.JoinCode, c.ID, len(c.StudentIDs), c.Scenario.Nome)
	}
	return b.String()
}

func nativeOnlineActivate() {
	server := onlineConfig.ServerURL
	if server == "" {
		server = "http://127.0.0.1:8787"
	}
	server, ok := nativePrompt("ATIVAR CONVITE", "Endereço do servidor", server)
	if !ok {
		return
	}
	code, ok := nativePrompt("ATIVAR CONVITE", "Código de convite", "")
	if !ok {
		return
	}
	pass, ok := nativePromptSecret("ATIVAR CONVITE", "Crie sua senha (mínimo 8 caracteres)")
	if !ok {
		return
	}
	if err := onlineRedeemInvitation(server, code, pass); err != nil {
		nativeInfo("Falha na ativação", err.Error())
		return
	}
	nativeInfo("Conta ativada", "Conta ALUNO ativada para "+onlineConfig.Name+".\nVocê já está vinculado à turma do convite.")
}

func chooseOnlineClass(title string) (OnlineClass, bool) {
	cs, err := onlineClasses()
	if err != nil {
		nativeInfo("Erro", err.Error())
		return OnlineClass{}, false
	}
	if len(cs) == 0 {
		nativeInfo(title, "Nenhuma turma encontrada.")
		return OnlineClass{}, false
	}
	items := []string{}
	for _, cl := range cs {
		items = append(items, cl.Name)
	}
	i := nativeChoose(title, "Escolha a turma", items, 0)
	if i < 0 {
		return OnlineClass{}, false
	}
	return cs[i], true
}

func nativeOnlineInvite() {
	cl, ok := chooseOnlineClass("CONVIDAR ALUNO")
	if !ok {
		return
	}
	name, ok := nativePrompt("CONVIDAR ALUNO", "Nome do aluno", "")
	if !ok {
		return
	}
	email, ok := nativePrompt("CONVIDAR ALUNO", "E-mail do aluno", "")
	if !ok {
		return
	}
	inst, ok := nativePrompt("CONVIDAR ALUNO", "Identificador institucional (opcional)", "")
	if !ok {
		inst = ""
	}
	inv, err := onlineCreateInvitation(cl.ID, name, email, inst)
	if err != nil {
		nativeInfo("Erro", err.Error())
		return
	}
	msg := fmt.Sprintf("%s\n%s\n\nCódigo de convite: %s", inv.Name, inv.Email, inv.Code)
	if inv.RedeemedAt != "" {
		msg += "\n\nEsse e-mail já possuía conta e foi vinculado à turma."
	}
	nativeInfo("Convite criado", msg)
}

func nativeOnlineMentorInvite() {
	name, ok := nativePrompt("CREDENCIAR MENTOR", "Nome do novo mentor", "")
	if !ok {
		return
	}
	email, ok := nativePrompt("CREDENCIAR MENTOR", "E-mail do novo mentor", "")
	if !ok {
		return
	}
	institution, ok := nativePrompt("CREDENCIAR MENTOR", "Instituição", "")
	if !ok {
		return
	}
	inst, ok := nativePrompt("CREDENCIAR MENTOR", "Identificador institucional (opcional)", "")
	if !ok {
		inst = ""
	}
	inv, err := onlineCreateMentorInvitation(name, email, institution, inst)
	if err != nil {
		nativeInfo("Credenciar mentor", err.Error())
		return
	}
	msg := fmt.Sprintf("%s\n%s\n%s\n\nCódigo de mentor: %s\nValidade: 30 dias\nUso único.",
		inv.Name, inv.Email, inv.Institution, inv.Code)
	switch inv.EmailStatus {
	case "sent":
		msg += "\n\n✓ Credencial enviada automaticamente por e-mail."
	case "not_configured":
		msg += "\n\nE-mail automático não está configurado.\nCopie o código e envie ao Mentor manualmente."
	case "failed":
		msg += "\n\n⚠ A credencial foi criada e continua válida, mas o e-mail não pôde ser enviado."
		if strings.TrimSpace(inv.EmailError) != "" {
			msg += "\n\nDetalhe: " + inv.EmailError
		}
	default:
		msg += "\n\nO status do envio de e-mail não foi informado pelo servidor."
	}
	nativeInfo("Credencial criada", msg)
}

func nativeOnlineMentorInvites() {
	invs, err := onlineMentorInvitations()
	if err != nil {
		nativeInfo("Credenciais de mentor", err.Error())
		return
	}
	if len(invs) == 0 {
		nativeInfo("Credenciais de mentor", "Nenhuma credencial emitida por esta conta.")
		return
	}
	var b strings.Builder
	for _, inv := range invs {
		status := "PENDENTE"
		if inv.RevokedAt != "" {
			status = "REVOGADA"
		} else if inv.RedeemedAt != "" {
			status = "ATIVADA"
		} else if inv.ExpiresAt != "" {
			if exp, err := time.Parse(time.RFC3339, inv.ExpiresAt); err == nil && time.Now().After(exp) {
				status = "EXPIRADA"
			}
		}
		emailState := ""
		switch inv.EmailStatus {
		case "sent":
			emailState = " • E-MAIL ENVIADO"
		case "not_configured":
			emailState = " • E-MAIL MANUAL"
		case "failed":
			emailState = " • E-MAIL FALHOU"
		}
		fmt.Fprintf(&b, "%s • %s • %s • %s%s\n", inv.Name, inv.Email, inv.Code, status, emailState)
	}
	nativeInfo("Credenciais de mentor", b.String())
}

func nativeChooseOnlineUser(title string, filter func(OnlineUser) bool) (OnlineUser, bool) {
	users, err := onlineAdminUsers()
	if err != nil {
		nativeInfo(title, err.Error())
		return OnlineUser{}, false
	}
	filtered := []OnlineUser{}
	items := []string{}
	for _, u := range users {
		if filter != nil && !filter(u) {
			continue
		}
		filtered = append(filtered, u)
		role := strings.ToUpper(u.Role)
		if isMentorRole(u.Role) {
			role = "MENTOR"
		} else if u.Role == "admin" {
			role = "ADMIN"
		}
		status := strings.ToUpper(u.Status)
		if status == "" {
			status = "ACTIVE"
		}
		extra := ""
		if u.IsPrimaryAdmin {
			extra += " • PRINCIPAL"
		}
		if isMentorRole(u.Role) && u.CanInviteMentors {
			extra += " • PODE CREDENCIAR"
		}
		items = append(items, fmt.Sprintf("%s — %s — %s%s", u.Name, role, status, extra))
	}
	if len(filtered) == 0 {
		nativeInfo(title, "Nenhum usuário compatível encontrado.")
		return OnlineUser{}, false
	}
	i := nativeChoose(title, "Escolha o usuário", items, 0)
	if i < 0 {
		return OnlineUser{}, false
	}
	return filtered[i], true
}

func nativeOnlineAdminUsers() {
	users, err := onlineAdminUsers()
	if err != nil {
		nativeInfo("USUÁRIOS", err.Error())
		return
	}
	var b strings.Builder
	counts := map[string]int{}
	for _, u := range users {
		counts[u.Role]++
	}
	fmt.Fprintf(&b, "Total: %d\nAdmins: %d • Mentores: %d • Alunos: %d\n\n",
		len(users), counts["admin"], counts["mentor"]+counts["tutor"], counts["aluno"])
	for _, u := range users {
		role := strings.ToUpper(u.Role)
		if isMentorRole(u.Role) {
			role = "MENTOR"
		} else if u.Role == "admin" {
			role = "ADMIN"
		}
		status := strings.ToUpper(u.Status)
		if status == "" {
			status = "ACTIVE"
		}
		flags := ""
		if u.IsPrimaryAdmin {
			flags += " • PRINCIPAL"
		}
		if isMentorRole(u.Role) && u.CanInviteMentors {
			flags += " • CREDENCIA MENTORES"
		}
		fmt.Fprintf(&b, "%s • %s • %s%s\n%s\n\n", u.Name, role, status, flags, u.Email)
	}
	nativeInfo("USUÁRIOS", b.String())
}

func nativeOnlineAdminCreate() {
	name, ok := nativePrompt("CRIAR ADMIN", "Nome do novo Administrador", "")
	if !ok {
		return
	}
	email, ok := nativePrompt("CRIAR ADMIN", "E-mail do novo Administrador", "")
	if !ok {
		return
	}
	u, err := onlineAdminCreateAdmin(name, email, defaultProvisionedPassword)
	if err != nil {
		nativeInfo("CRIAR ADMIN", err.Error())
		return
	}
	nativeInfo("CRIAR ADMIN", fmt.Sprintf("Administrador criado:\n%s\n%s\n\nSenha temporária: %s\nA troca será obrigatória no primeiro acesso. O servidor tentará enviar os dados por e-mail.", u.Name, u.Email, defaultProvisionedPassword))
}

func nativeOnlineAdminStatus() {
	u, ok := nativeChooseOnlineUser("ATIVAR / DESATIVAR", func(u OnlineUser) bool {
		return u.ID != onlineConfig.UserID
	})
	if !ok {
		return
	}
	current := u.Status
	if current == "" {
		current = "active"
	}
	target := "disabled"
	action := "DESATIVAR"
	if current == "disabled" {
		target = "active"
		action = "REATIVAR"
	}
	i := nativeChoose("CONFIRMAR", action+" "+u.Name+"?", []string{"Confirmar", "Cancelar"}, 1)
	if i != 0 {
		return
	}
	updated, err := onlineAdminSetUserStatus(u.ID, target)
	if err != nil {
		nativeInfo("USUÁRIO", err.Error())
		return
	}
	nativeInfo("USUÁRIO", updated.Name+" agora está "+strings.ToUpper(updated.Status)+".")
}

func nativeOnlineAdminMentorPermission() {
	u, ok := nativeChooseOnlineUser("PERMISSÃO DE MENTOR", func(u OnlineUser) bool {
		return isMentorRole(u.Role)
	})
	if !ok {
		return
	}
	allowed := !u.CanInviteMentors
	action := "CONCEDER"
	if !allowed {
		action = "REVOGAR"
	}
	i := nativeChoose("PERMISSÃO DE MENTOR",
		action+" permissão para "+u.Name+" credenciar outros Mentores?",
		[]string{"Confirmar", "Cancelar"}, 1)
	if i != 0 {
		return
	}
	updated, err := onlineAdminSetMentorPermission(u.ID, allowed)
	if err != nil {
		nativeInfo("PERMISSÃO DE MENTOR", err.Error())
		return
	}
	state := "REVOGADA"
	if updated.CanInviteMentors {
		state = "CONCEDIDA"
	}
	nativeInfo("PERMISSÃO DE MENTOR", "Permissão "+state+" para "+updated.Name+".")
}

func nativeOnlineAdminTransfer() {
	u, ok := nativeChooseOnlineUser("TRANSFERIR ADMINISTRAÇÃO PRINCIPAL", func(u OnlineUser) bool {
		return u.Role == "admin" && u.ID != onlineConfig.UserID && activeStatus(u.Status)
	})
	if !ok {
		return
	}
	i := nativeChoose("TRANSFERIR ADMINISTRAÇÃO PRINCIPAL",
		"Transferir a função de Administrador Principal para "+u.Name+"?\nSua conta continuará como Administrador comum.",
		[]string{"TRANSFERIR", "CANCELAR"}, 1)
	if i != 0 {
		return
	}
	target, err := onlineAdminTransferPrimary(u.ID)
	if err != nil {
		nativeInfo("TRANSFERÊNCIA", err.Error())
		return
	}
	nativeInfo("TRANSFERÊNCIA", target.Name+" agora é o Administrador Principal.")
}

func nativeOnlineImportCSV() {
	cl, ok := chooseOnlineClass("IMPORTAR ALUNOS")
	if !ok {
		return
	}
	path, ok := nativePrompt("IMPORTAR CSV", "Caminho do arquivo CSV\nColunas: nome,email,identificador(opcional)", "")
	if !ok {
		return
	}
	rows, err := importStudentsCSV(path)
	if err != nil {
		nativeInfo("Erro ao ler CSV", err.Error())
		return
	}
	if len(rows) == 0 {
		nativeInfo("Importação", "Nenhum aluno válido encontrado.")
		return
	}
	okCount, fail := 0, 0
	var codes strings.Builder
	for _, row := range rows {
		inv, err := onlineCreateInvitation(cl.ID, row[0], row[1], row[2])
		if err != nil {
			fail++
			continue
		}
		okCount++
		fmt.Fprintf(&codes, "%s; %s; %s\n", inv.Name, inv.Email, inv.Code)
	}
	report := fmt.Sprintf("Turma: %s\nConvites processados: %d\nFalhas: %d\n\n%s", cl.Name, okCount, fail, codes.String())
	nativeInfo("Importação concluída", report)
}

func nativeOnlineInvites() {
	cl, ok := chooseOnlineClass("CONVITES")
	if !ok {
		return
	}
	invs, err := onlineInvitations(cl.ID)
	if err != nil {
		nativeInfo("Erro", err.Error())
		return
	}
	if len(invs) == 0 {
		nativeInfo("Convites", "Nenhum convite nesta turma.")
		return
	}
	var b strings.Builder
	for _, inv := range invs {
		status := "PENDENTE"
		if inv.RevokedAt != "" {
			status = "REVOGADO"
		} else if inv.RedeemedAt != "" {
			status = "ATIVADO"
		} else if inv.ExpiresAt != "" {
			if exp, err := time.Parse(time.RFC3339, inv.ExpiresAt); err == nil && time.Now().After(exp) {
				status = "EXPIRADO"
			}
		}
		fmt.Fprintf(&b, "%s • %s • %s • %s\n", inv.Name, inv.Email, inv.Code, status)
	}
	nativeInfo("Convites — "+cl.Name, b.String())
}

func nativeOnlineChangePassword() {
	cur, ok := nativePrompt("ALTERAR SENHA", "Senha atual", "")
	if !ok {
		return
	}
	next, ok := nativePrompt("ALTERAR SENHA", "Nova senha (mínimo 8 caracteres)", "")
	if !ok {
		return
	}
	confirm, ok := nativePrompt("ALTERAR SENHA", "Repita a nova senha", "")
	if !ok {
		return
	}
	if next != confirm {
		nativeInfo("Alterar senha", "As novas senhas não coincidem.")
		return
	}
	if err := onlineChangePassword(cur, next); err != nil {
		nativeInfo("Alterar senha", err.Error())
		return
	}
	nativeInfo("Alterar senha", "Senha alterada com sucesso.")
}

func nativeOnlineRevokeInvite() {
	kind := 1
	if onlineConfig.Role != "admin" {
		i := nativeChoose("REVOGAR CÓDIGO", "Qual tipo de código deseja revogar?",
			[]string{"Convite de aluno", "Credencial de mentor"}, 0)
		if i < 0 {
			return
		}
		kind = i
	}
	code, ok := nativePrompt("REVOGAR CÓDIGO", "Código a revogar", "")
	if !ok {
		return
	}
	var err error
	if kind == 0 {
		err = onlineRevokeInvitation(code)
	} else {
		err = onlineRevokeMentorInvitation(code)
	}
	if err != nil {
		nativeInfo("Revogar código", err.Error())
		return
	}
	nativeInfo("Revogar código", "Código revogado.")
}

func nativeOnlineClasses() {
	cs, err := onlineClasses()
	if err != nil {
		nativeInfo("JED Online", err.Error())
		return
	}
	nativeInfo("Turmas online", onlineClassesText(cs))
}

func nativeOnlineCreateClass() {
	name, ok := nativePrompt("CRIAR TURMA ONLINE", "Nome da turma", "")
	if !ok || strings.TrimSpace(name) == "" {
		return
	}
	items := []string{}
	for _, c := range cenariosBase {
		items = append(items, c.Nome)
	}
	i := nativeChoose("CENÁRIO", "Escolha o cenário inicial", items, 0)
	if i < 0 {
		return
	}
	c, err := onlineCreateClass(name, cenariosBase[i])
	if err != nil {
		nativeInfo("Erro", err.Error())
		return
	}
	nativeInfo("Turma criada", fmt.Sprintf("%s\n\nCódigo para os alunos: %s", c.Name, c.JoinCode))
}

func nativeOnlineCreateStudent() {
	name, ok := nativePrompt("NOVO ALUNO", "Nome do aluno", "")
	if !ok {
		return
	}
	email, ok := nativePrompt("NOVO ALUNO", "E-mail", "")
	if !ok {
		return
	}
	pass, ok := nativePrompt("NOVO ALUNO", "Senha inicial (mínimo 8 caracteres)", "")
	if !ok {
		return
	}
	u, err := onlineCreateStudent(name, email, pass)
	if err != nil {
		nativeInfo("Erro", err.Error())
		return
	}
	nativeInfo("Aluno criado", u.Name+"\n"+u.Email)
}

func nativeOnlineJoin() {
	code, ok := nativePrompt("ENTRAR EM TURMA", "Código da turma", "")
	if !ok {
		return
	}
	c, err := onlineJoinClass(code)
	if err != nil {
		nativeInfo("Erro", err.Error())
		return
	}
	nativeInfo("Turma vinculada", c.Name+"\nCenário: "+c.Scenario.Nome)
}

func nativeOnlineSync() {
	if ns.current == nil {
		nativeInfo("Sincronização", "Abra ou crie uma empresa antes de sincronizar.")
		return
	}
	rc, err := onlineSyncCompany(ns.current)
	if err != nil {
		nativeInfo("Erro de sincronização", err.Error())
		return
	}
	nativeInfo("Sincronizado", fmt.Sprintf("%s\nRevisão online: %d\nAtualizado: %s", rc.Company.Nome, rc.Revision, rc.UpdatedAt))
}

func nativeOnlineMyCompanies() {
	cs, err := onlineMyCompanies()
	if err != nil {
		nativeInfo("Erro", err.Error())
		return
	}
	if len(cs) == 0 {
		nativeInfo("Empresas online", "Nenhuma empresa sincronizada.")
		return
	}
	var b strings.Builder
	for _, c := range cs {
		fmt.Fprintf(&b, "%s • semana %d • caixa %s • rev. %d\n", c.Company.Nome, c.Company.Semana, money(c.Company.Caixa), c.Revision)
	}
	nativeInfo("Empresas online", b.String())
}

func nativeOnlineClassCompanies() {
	cs, err := onlineClasses()
	if err != nil {
		nativeInfo("Erro", err.Error())
		return
	}
	if len(cs) == 0 {
		nativeInfo("Resultados", "Nenhuma turma.")
		return
	}
	items := []string{}
	for _, c := range cs {
		items = append(items, c.Name+" ["+c.JoinCode+"]")
	}
	i := nativeChoose("RESULTADOS", "Escolha a turma", items, 0)
	if i < 0 {
		return
	}
	rows, err := onlineClassCompanies(cs[i].ID)
	if err != nil {
		nativeInfo("Erro", err.Error())
		return
	}
	if len(rows) == 0 {
		nativeInfo("Resultados", "Nenhuma empresa sincronizada nesta turma.")
		return
	}
	var b strings.Builder
	for _, r := range rows {
		fmt.Fprintf(&b, "%s • %s • semana %d • caixa %s • índice %.0f\n", r.Company.Responsavel, r.Company.Nome, r.Company.Semana, money(r.Company.Caixa), score(&r.Company).Total)
	}
	nativeInfo("Resultados — "+cs[i].Name, b.String())
}

func drawCatalog(hdc uintptr, cr rect) {
	drawChrome(hdc, cr, "SELEÇÃO DE NEGÓCIO", "setor → tipo → especialidade")
	left := int32(42)
	top := int32(170)
	if ns.selectedSector < 0 {
		ns.selectedSector = 0
	}
	if ns.selectedSector >= len(ns.cat.Setores) {
		ns.selectedSector = 0
	}
	// sector column
	text(hdc, "01 / SETOR", rect{left, 145, left + 260, 170}, 11, colMuted, DT_LEFT, true)
	for i, s := range ns.cat.Setores {
		r := rect{left, top + int32(i*42), left + 260, top + int32(i*42) + 34}
		c := colLine
		if i == ns.selectedSector {
			c = colCyan
			fill(hdc, r, rgb(235, 250, 251))
		}
		line(hdc, r.Left, r.Bottom, r.Right, r.Bottom, c, 1)
		text(hdc, s.Nome, r, 11, colInk, DT_LEFT, i == ns.selectedSector)
		addHit("sector:"+strconv.Itoa(i), r)
	}
	s := ns.cat.Setores[ns.selectedSector]
	mid := int32(350)
	text(hdc, "02 / TIPO", rect{mid, 145, mid + 300, 170}, 11, colMuted, DT_LEFT, true)
	if ns.selectedType < 0 || ns.selectedType >= len(s.Tipos) {
		ns.selectedType = 0
	}
	for i, t := range s.Tipos {
		r := rect{mid, top + int32(i*52), mid + 310, top + int32(i*52) + 42}
		c := colLine
		if i == ns.selectedType {
			c = colBlue
			fill(hdc, r, rgb(240, 245, 255))
		}
		line(hdc, r.Left, r.Bottom, r.Right, r.Bottom, c, 1)
		text(hdc, t.Nome, r, 12, colInk, DT_LEFT, i == ns.selectedType)
		addHit("type:"+strconv.Itoa(i), r)
	}
	t := s.Tipos[ns.selectedType]
	right := int32(720)
	text(hdc, "03 / ESPECIALIDADE", rect{right, 145, cr.Right - 42, 170}, 11, colMuted, DT_LEFT, true)
	for i, m := range t.Especialidades {
		y := top + int32(i*64)
		if y+54 > cr.Bottom-60 {
			break
		}
		r := rect{right, y, cr.Right - 42, y + 52}
		accent := colCyan
		if m.Regulamentado {
			accent = colMagenta
		}
		button(hdc, "model:"+m.ID, m.Nome, fmt.Sprintf("Preço ref. %s  •  Capacidade %d/sem", money(m.PrecoRef), m.CapacidadeBase), r, accent)
	}
	backButton(hdc, cr)
}

func backButton(hdc uintptr, cr rect) {
	r := rect{42, cr.Bottom - 48, 180, cr.Bottom - 16}
	text(hdc, "← VOLTAR", r, 11, colMuted, DT_LEFT, true)
	addHit("back", r)
}

func navButton(hdc uintptr, id, label, sub string, r rect, active bool) {
	bg := colPanel
	accent := colInk
	if active {
		bg = rgb(234, 250, 251)
		accent = colCyan
	}
	if ns.hover == id {
		if active {
			bg = rgb(230, 247, 248)
		} else {
			bg = rgb(245, 245, 245)
		}
	}
	fill(hdc, r, bg)
	line(hdc, r.Left, r.Top, r.Left, r.Bottom, accent, 4)
	line(hdc, r.Left, r.Bottom, r.Right, r.Bottom, colLine, 1)
	text(hdc, label, rect{r.Left + 16, r.Top + 7, r.Right - 12, r.Top + 24}, 10, colInk, DT_LEFT, true)
	multiText(hdc, sub, rect{r.Left + 16, r.Top + 22, r.Right - 12, r.Bottom - 5}, 8, colMuted, false)
	addHit(id, r)
}

func drawSparkline(hdc uintptr, r rect, e *Empresa) {
	fill(hdc, r, colPanel)
	line(hdc, r.Left, r.Top, r.Right, r.Top, colBlue, 3)
	text(hdc, "RESULTADO • ÚLTIMAS SEMANAS", rect{r.Left + 14, r.Top + 8, r.Right - 14, r.Top + 26}, 9, colMuted, DT_LEFT, true)
	if len(e.Historico) < 2 {
		multiText(hdc, "Conclua ao menos duas semanas para formar a tendência.", rect{r.Left + 14, r.Top + 28, r.Right - 14, r.Bottom - 10}, 10, colMuted, false)
		return
	}
	start := len(e.Historico) - 8
	if start < 0 {
		start = 0
	}
	h := e.Historico[start:]
	minV, maxV := h[0].Resultado, h[0].Resultado
	for _, rr := range h {
		if rr.Resultado < minV {
			minV = rr.Resultado
		}
		if rr.Resultado > maxV {
			maxV = rr.Resultado
		}
	}
	if math.Abs(maxV-minV) < 0.01 {
		maxV = minV + 1
	}
	left, right := r.Left+18, r.Right-18
	top, bottom := r.Top+40, r.Bottom-18
	var px, py int32
	for i, rr := range h {
		x := left
		if len(h) > 1 {
			x = left + int32(float64(right-left)*float64(i)/float64(len(h)-1))
		}
		y := bottom - int32((rr.Resultado-minV)/(maxV-minV)*float64(bottom-top))
		if i > 0 {
			line(hdc, px, py, x, y, colBlue, 2)
		}
		withObj(hdc, brush(colCyan), func() { pEllipse.Call(hdc, uintptr(x-3), uintptr(y-3), uintptr(x+4), uintptr(y+4)) })
		px, py = x, y
	}
}

func alertSummary(e *Empresa) (string, uintptr) {
	if e.Caixa < 0 {
		return "CAIXA NEGATIVO • liquidez crítica", colMagenta
	}
	if sumAccounts(e.ContasPagar) > e.Caixa+sumAccounts(e.ContasReceber) {
		return "ATENÇÃO • compromissos superam recursos disponíveis", colAmber
	}
	if e.UsaInsumos {
		for _, i := range e.Insumos {
			if i.Critico && i.ConsumoPorVenda > 0 && i.Quantidade/i.ConsumoPorVenda < 8 {
				return "ESTOQUE CRÍTICO • " + i.Nome, colMagenta
			}
		}
	}
	if len(e.Historico) > 0 {
		r := e.Historico[len(e.Historico)-1]
		if r.VendasPerdidas > 0 {
			return fmt.Sprintf("OPORTUNIDADE • %d venda(s) perdida(s) na última semana", r.VendasPerdidas), colAmber
		}
	}
	return "OPERAÇÃO ESTÁVEL • acompanhe hipóteses e capital de giro", colCyan
}

func drawDashboard(hdc uintptr, cr rect) {
	e := ns.current
	if e == nil {
		ns.screen = "home"
		drawHome(hdc, cr)
		return
	}
	drawChrome(hdc, cr, e.Nome, fmt.Sprintf("%s / %s / semana %d de %d", e.Setor, e.Especialidade, e.Semana, e.DuracaoSemanas))
	// left nav
	navX := int32(32)
	y := int32(154)
	items := [][3]string{{"overview", "VISÃO GERAL", "status do negócio"}, {"journey", "JORNADA JED", "progresso no JED"}, {"persona", "PERSONA", "perfil do cliente"}, {"lean", "LEAN CANVAS", "modelo e hipóteses"}, {"channels", "CANAIS", "vendas digitais"}, {"tools", "FERRAMENTAS", "ações digitais"}, {"inventory", "INSUMOS", "estoque e compras"}, {"finance", "FINANCEIRO", "caixa e custos"}, {"indicators", "INDICADORES", "métricas do negócio"}, {"tutor", "MODO TUTOR", "turmas e cenários"}}
	for _, it := range items {
		active := (it[0] == "overview")
		navButton(hdc, it[0], it[1], it[2], rect{navX, y, 270, y + 48}, active)
		y += 49
	}
	// central dial
	cx, cy := int32(555), int32(355)
	rad := int32(142)
	s := score(e)
	val := s.Total
	if e.Semana == 0 {
		val = 50
	}
	drawGauge(hdc, cx, cy, rad, val)
	text(hdc, "ÍNDICE DE DESENVOLVIMENTO", rect{cx - 130, cy - 18, cx + 130, cy + 8}, 10, colMuted, DT_CENTER, true)
	text(hdc, fmt.Sprintf("%.0f", val), rect{cx - 110, cy + 4, cx + 110, cy + 70}, 46, colInk, DT_CENTER, true)
	text(hdc, "/100", rect{cx - 70, cy + 65, cx + 70, cy + 90}, 11, colMuted, DT_CENTER, true)
	// metrics right
	rx := int32(770)
	rw := cr.Right - rx - 42
	metricPanel(hdc, rect{rx, 170, rx + rw, 250}, "CAIXA", money(e.Caixa), cashAccent(e.Caixa))
	metricPanel(hdc, rect{rx, 265, rx + rw, 345}, "CLIENTES ATIVOS", strconv.Itoa(e.ClientesAtivos), colInk)
	metricPanel(hdc, rect{rx, 360, rx + rw, 440}, "REPUTAÇÃO", fmt.Sprintf("%.0f / 100", e.Reputacao), colInk)
	est := e.EstoqueValor
	if e.UsaInsumos {
		est = 0
		for _, i := range e.Insumos {
			est += i.Quantidade * i.CustoMedio
		}
	}
	metricPanel(hdc, rect{rx, 455, rx + rw, 535}, "ESTOQUE / INSUMOS", money(est), colInk)
	// bottom actions
	ay := cr.Bottom - 144
	aw := int32(175)
	gap := int32(12)
	ax := int32(310)
	button(hdc, "week", "ENCERRAR SEMANA", "Processar mercado e operação.", rect{ax, ay, ax + aw, ay + 86}, colCyan)
	ax += aw + gap
	button(hdc, "price", digitalModelLabel(e), "Alterar valor unitário.", rect{ax, ay, ax + aw, ay + 86}, colInk)
	ax += aw + gap
	button(hdc, "marketing", "MARKETING", "Ajustar investimento.", rect{ax, ay, ax + aw, ay + 86}, colInk)
	ax += aw + gap
	button(hdc, "promo", "PROMOÇÃO", "Desconto por uma semana.", rect{ax, ay, ax + aw, ay + 86}, colMagenta)
	alertText, alertColor := alertSummary(e)
	fill(hdc, rect{310, 548, cr.Right - 42, 592}, colPanel)
	line(hdc, 310, 548, 310, 592, alertColor, 5)
	text(hdc, alertText, rect{326, 554, cr.Right - 56, 588}, 10, colInk, DT_LEFT, true)
	sparkTop := int32(598)
	if ns.status != "" {
		fill(hdc, rect{310, 598, cr.Right - 42, 632}, colPanel)
		line(hdc, 310, 598, cr.Right-42, 598, colLine, 1)
		multiText(hdc, ns.status, rect{324, 604, cr.Right - 56, 626}, 9, colMuted, false)
		sparkTop = 638
	}
	if ay-sparkTop >= 48 {
		drawSparkline(hdc, rect{310, sparkTop, cr.Right - 42, ay - 10}, e)
	}
	backButton(hdc, cr)
}

func drawGauge(hdc uintptr, cx, cy, rad int32, value float64) {
	withObj(hdc, pen(colLine, 2), func() { pEllipse.Call(hdc, uintptr(cx-rad), uintptr(cy-rad), uintptr(cx+rad), uintptr(cy+rad)) })
	withObj(hdc, pen(colGrid, 10), func() {
		pArc.Call(hdc, uintptr(cx-rad+14), uintptr(cy-rad+14), uintptr(cx+rad-14), uintptr(cy+rad-14), uintptr(cx), uintptr(cy-rad), uintptr(cx-1), uintptr(cy-rad))
	})
	// segmented ring approximation
	segments := 40
	on := int(math.Round(value / 100 * float64(segments)))
	for i := 0; i < segments; i++ {
		a := (-210.0 + float64(i)*240.0/float64(segments-1)) * math.Pi / 180
		a2 := a + 3.2*math.Pi/180
		r1 := float64(rad - 4)
		r2 := float64(rad - 22)
		x1 := cx + int32(math.Cos(a)*r1)
		y1 := cy + int32(math.Sin(a)*r1)
		x2 := cx + int32(math.Cos(a)*r2)
		y2 := cy + int32(math.Sin(a)*r2)
		c := colLine
		if i < on {
			c = colCyan
		}
		if value < 35 && i < on {
			c = colMagenta
		}
		line(hdc, x1, y1, x2, y2, c, 5)
		_ = a2
	}
}

func metricPanel(hdc uintptr, r rect, label, value string, accent uintptr) {
	fill(hdc, r, colPanel)
	line(hdc, r.Left, r.Top, r.Right, r.Top, accent, 4)
	line(hdc, r.Left, r.Bottom, r.Right, r.Bottom, colLine, 1)
	text(hdc, label, rect{r.Left + 18, r.Top + 10, r.Right - 18, r.Top + 32}, 10, colMuted, DT_LEFT, true)
	text(hdc, value, rect{r.Left + 18, r.Top + 30, r.Right - 18, r.Bottom - 10}, 22, colInk, DT_LEFT, true)
}
func cashAccent(v float64) uintptr {
	if v < 0 {
		return colMagenta
	}
	if v < 1000 {
		return colAmber
	}
	return colCyan
}

func drawInventory(hdc uintptr, cr rect) {
	e := ns.current
	if e == nil {
		ns.screen = "home"
		drawHome(hdc, cr)
		return
	}
	drawChrome(hdc, cr, "INSUMOS E FORNECEDORES", e.Nome)
	y := int32(175)
	text(hdc, "ITEM", rect{42, 145, 360, 170}, 10, colMuted, DT_LEFT, true)
	text(hdc, "QUANTIDADE", rect{420, 145, 570, 170}, 10, colMuted, DT_RIGHT, true)
	text(hdc, "VALOR", rect{620, 145, 770, 170}, 10, colMuted, DT_RIGHT, true)
	if len(e.Insumos) == 0 {
		multiText(hdc, "Este negócio não utiliza insumos detalhados. O estoque é controlado de forma agregada.", rect{42, 180, 800, 250}, 13, colMuted, false)
	}
	for idx, i := range e.Insumos {
		if y+46 > cr.Bottom-120 {
			break
		}
		r := rect{42, y, 800, y + 40}
		fill(hdc, r, colPanel)
		line(hdc, 42, y+40, 800, y+40, colLine, 1)
		text(hdc, i.Nome, rect{56, y, 400, y + 40}, 11, colInk, DT_LEFT, i.Critico)
		text(hdc, fmt.Sprintf("%.1f %s", i.Quantidade, i.Unidade), rect{420, y, 570, y + 40}, 11, colInk, DT_RIGHT, false)
		text(hdc, money(i.Quantidade*i.CustoMedio), rect{620, y, 770, y + 40}, 11, colInk, DT_RIGHT, false)
		addHit("input:"+strconv.Itoa(idx), r)
		y += 48
	}
	x := int32(850)
	metricPanel(hdc, rect{x, 175, cr.Right - 42, 260}, "PEDIDOS EM TRÂNSITO", strconv.Itoa(len(e.PedidosInsumos)), colBlue)
	metricPanel(hdc, rect{x, 280, cr.Right - 42, 365}, "CONTAS A PAGAR", money(sumAccounts(e.ContasPagar)), colMagenta)
	button(hdc, "order", "NOVO PEDIDO", "Clique em um insumo à esquerda ou compre o primeiro item crítico.", rect{x, 395, cr.Right - 42, 470}, colCyan)
	button(hdc, "dashboard", "VOLTAR AO PAINEL", "Retornar à visão geral.", rect{x, 485, cr.Right - 42, 560}, colInk)
	backButton(hdc, cr)
}

func drawFinance(hdc uintptr, cr rect) {
	e := ns.current
	if e == nil {
		ns.screen = "home"
		drawHome(hdc, cr)
		return
	}
	drawChrome(hdc, cr, "FINANCEIRO", "liquidez, recebíveis e compromissos")
	metricPanel(hdc, rect{42, 175, 350, 275}, "CAIXA", money(e.Caixa), cashAccent(e.Caixa))
	metricPanel(hdc, rect{370, 175, 678, 275}, "A RECEBER", money(sumAccounts(e.ContasReceber)), colCyan)
	metricPanel(hdc, rect{698, 175, 1006, 275}, "A PAGAR", money(sumAccounts(e.ContasPagar)), colMagenta)
	ind := indicators(e)
	resultAccum := 0.0
	if ind != nil {
		resultAccum = ind.Resultado
	}
	metricPanel(hdc, rect{1026, 175, cr.Right - 42, 275}, "RESULTADO ACUMULADO", money(resultAccum), cashAccent(resultAccum))
	y := int32(330)
	text(hdc, "PRÓXIMOS MOVIMENTOS", rect{42, 300, 600, 326}, 12, colMuted, DT_LEFT, true)
	for _, c := range e.ContasReceber {
		if y > cr.Bottom-80 {
			break
		}
		text(hdc, fmt.Sprintf("RECEBER  S%02d  %s", c.Semana, money(c.Valor)), rect{52, y, 500, y + 30}, 11, colGreen, DT_LEFT, true)
		text(hdc, c.Descricao, rect{520, y, cr.Right - 52, y + 30}, 10, colMuted, DT_LEFT, false)
		y += 36
	}
	for _, c := range e.ContasPagar {
		if y > cr.Bottom-80 {
			break
		}
		text(hdc, fmt.Sprintf("PAGAR    S%02d  %s", c.Semana, money(c.Valor)), rect{52, y, 500, y + 30}, 11, colMagenta, DT_LEFT, true)
		text(hdc, c.Descricao, rect{520, y, cr.Right - 52, y + 30}, 10, colMuted, DT_LEFT, false)
		y += 36
	}
	button(hdc, "dashboard", "VOLTAR AO PAINEL", "Visão geral da empresa.", rect{cr.Right - 310, cr.Bottom - 110, cr.Right - 42, cr.Bottom - 48}, colInk)
	backButton(hdc, cr)
}

func drawLean(hdc uintptr, cr rect) {
	e := ns.current
	if e == nil {
		ns.screen = "home"
		drawHome(hdc, cr)
		return
	}
	drawChrome(hdc, cr, "LEAN CANVAS", "mapa estratégico da empresa")
	boxes := []struct{ title, body, id string }{
		{"PROBLEMA", e.Canvas.Problema, "lean:problema"}, {"SOLUÇÃO", e.Canvas.Solucao, "lean:solucao"}, {"OFERTA DE VALOR", e.Canvas.PropostaValor, "lean:oferta"},
		{"VANTAGEM DIFERENCIAL", e.Canvas.Vantagem, "lean:vantagem"}, {"SEGMENTO DE CLIENTES", strings.Join(e.Canvas.Segmentos, ", "), "lean:segmentos"}, {"MÉTRICAS", strings.Join(e.Canvas.Metricas, ", "), "lean:metricas"},
		{"CANAIS", strings.Join(e.Canvas.Canais, ", "), "lean:canais"}, {"ESTRUTURA DE CUSTOS", e.Canvas.CustosNotas, "lean:custos"}, {"FONTES DE RECEITAS", e.Canvas.ReceitaModelo, "lean:receitas"},
	}
	x0, y0 := int32(42), int32(170)
	gap := int32(16)
	w := (cr.Right - 84 - gap*2) / 3
	h := int32(150)
	for i, b := range boxes {
		col := i % 3
		row := i / 3
		r := rect{x0 + int32(col)*(w+gap), y0 + int32(row)*(h+gap), x0 + int32(col)*(w+gap) + w, y0 + int32(row)*(h+gap) + h}
		fill(hdc, r, colPanel)
		line(hdc, r.Left, r.Top, r.Right, r.Top, colInk, 3)
		text(hdc, b.title, rect{r.Left + 14, r.Top + 10, r.Right - 14, r.Top + 34}, 10, colMuted, DT_LEFT, true)
		multiText(hdc, b.body, rect{r.Left + 14, r.Top + 42, r.Right - 14, r.Bottom - 12}, 11, colInk, false)
		addHit(b.id, r)
	}
	text(hdc, "CLIQUE EM QUALQUER BLOCO PARA REVISAR • o Canvas é um mapa vivo", rect{42, cr.Bottom - 70, cr.Right - 330, cr.Bottom - 30}, 10, colMuted, DT_LEFT, true)
	button(hdc, "dashboard", "VOLTAR AO PAINEL", "Visão geral da empresa.", rect{cr.Right - 310, cr.Bottom - 78, cr.Right - 42, cr.Bottom - 20}, colInk)
	backButton(hdc, cr)
}

func drawPersona(hdc uintptr, cr rect) {
	e := ns.current
	if e == nil {
		ns.screen = "home"
		drawHome(hdc, cr)
		return
	}
	drawChrome(hdc, cr, "CANVAS PERSONA", "comportamentos, necessidades, desejos, desafios e motivações")
	items := []struct{ title, body, id string }{
		{"NOME / IDENTIDADE", e.Persona.Nome, "persona:nome"},
		{"DEMOGRAFIA", e.Persona.Demografia, "persona:demografia"},
		{"ROTINAS", e.Persona.Rotinas, "persona:rotinas"},
		{"OBJETIVOS", e.Persona.Objetivos, "persona:objetivos"},
		{"DESAFIOS", e.Persona.Desafios, "persona:desafios"},
		{"MOTIVADORES", e.Persona.Motivadores, "persona:motivadores"},
		{"OBJEÇÕES", e.Persona.Objecoes, "persona:objecoes"},
		{"CITAÇÕES", e.Persona.Citacoes, "persona:citacoes"},
		{"PALAVRAS-CHAVE", e.Persona.PalavrasChave, "persona:palavras"},
	}
	x0, y0 := int32(42), int32(165)
	gap := int32(14)
	w := (cr.Right - 84 - gap*2) / 3
	h := int32(150)
	for i, it := range items {
		col, row := i%3, i/3
		r := rect{x0 + int32(col)*(w+gap), y0 + int32(row)*(h+gap), x0 + int32(col)*(w+gap) + w, y0 + int32(row)*(h+gap) + h}
		fill(hdc, r, colPanel)
		line(hdc, r.Left, r.Top, r.Right, r.Top, colInk, 3)
		text(hdc, it.title, rect{r.Left + 14, r.Top + 10, r.Right - 14, r.Top + 34}, 10, colMuted, DT_LEFT, true)
		multiText(hdc, it.body, rect{r.Left + 14, r.Top + 42, r.Right - 14, r.Bottom - 12}, 11, colInk, false)
		addHit(it.id, r)
	}
	pc := personaCompleteness(e.Persona) * 100
	text(hdc, fmt.Sprintf("COMPLETUDE DA PERSONA  %.0f%%  •  clique em qualquer bloco para editar", pc), rect{42, cr.Bottom - 70, cr.Right - 330, cr.Bottom - 30}, 10, colMuted, DT_LEFT, true)
	button(hdc, "dashboard", "VOLTAR AO PAINEL", "Visão geral da empresa.", rect{cr.Right - 310, cr.Bottom - 78, cr.Right - 42, cr.Bottom - 20}, colInk)
	backButton(hdc, cr)
}

func drawChannels(hdc uintptr, cr rect) {
	e := ns.current
	if e == nil {
		ns.screen = "home"
		drawHome(hdc, cr)
		return
	}
	drawChrome(hdc, cr, "CANAIS DIGITAIS", "escolha os canais conforme a Persona e o modelo de negócio")
	y := int32(165)
	for _, c := range canaisDigitais {
		selected := contains(e.CanaisDigitais, c.ID)
		r := rect{42, y, cr.Right - 42, y + 64}
		fill(hdc, r, colPanel)
		accent := colInk
		state := "INATIVO"
		if selected {
			accent = colCyan
			state = "ATIVO"
		}
		line(hdc, r.Left, r.Top, r.Left, r.Bottom, accent, 5)
		line(hdc, r.Left, r.Bottom, r.Right, r.Bottom, colLine, 1)
		text(hdc, c.Nome, rect{60, y, 300, y + 30}, 12, colInk, DT_LEFT, true)
		text(hdc, c.Descricao, rect{310, y, cr.Right - 210, y + 30}, 10, colMuted, DT_LEFT, false)
		text(hdc, state, rect{cr.Right - 190, y, cr.Right - 60, y + 30}, 10, accent, DT_RIGHT, true)
		text(hdc, fmt.Sprintf("efeito simulado: alcance %.2fx  conversão %.2fx  recorrência %.2fx", c.Alcance, c.Conversao, c.Recorrencia), rect{60, y + 30, cr.Right - 60, y + 58}, 9, colMuted, DT_LEFT, false)
		addHit("channel:"+c.ID, r)
		y += 72
		if y > cr.Bottom-105 {
			break
		}
	}
	if len(e.Historico) > 0 {
		r := e.Historico[len(e.Historico)-1]
		text(hdc, fmt.Sprintf("ÚLTIMO FUNIL  alcance %d  →  interações %d  →  leads %d  →  novos clientes %d", r.Alcance, r.Interacoes, r.Leads, r.NovosClientes), rect{42, cr.Bottom - 92, cr.Right - 330, cr.Bottom - 52}, 10, colInk, DT_LEFT, true)
	}
	button(hdc, "dashboard", "VOLTAR AO PAINEL", "Visão geral da empresa.", rect{cr.Right - 310, cr.Bottom - 78, cr.Right - 42, cr.Bottom - 20}, colInk)
	backButton(hdc, cr)
}

func drawTools(hdc uintptr, cr rect) {
	e := ns.current
	if e == nil {
		ns.screen = "home"
		drawHome(hdc, cr)
		return
	}
	drawChrome(hdc, cr, "FERRAMENTAS DIGITAIS", "práticas estudadas nos kits JED • efeitos são parâmetros da simulação")
	cols := int32(2)
	gap := int32(18)
	x0 := int32(42)
	y0 := int32(170)
	w := (cr.Right - 84 - gap) / cols
	h := int32(142)
	for i, f := range ferramentasDigitais {
		col := int32(i) % cols
		row := int32(i) / cols
		r := rect{x0 + col*(w+gap), y0 + row*(h+gap), x0 + col*(w+gap) + w, y0 + row*(h+gap) + h}
		active := hasDigitalTool(e, f.ID)
		bg := colPanel
		accent := colInk
		state := "INATIVA"
		if active {
			bg = rgb(247, 247, 247)
			accent = colCyan
			state = "ATIVA"
		}
		if ns.hover == "tool:"+f.ID {
			bg = rgb(242, 242, 242)
		}
		fill(hdc, r, bg)
		line(hdc, r.Left, r.Top, r.Right, r.Top, accent, 4)
		line(hdc, r.Left, r.Bottom, r.Right, r.Bottom, colLine, 1)
		text(hdc, f.Nome, rect{r.Left + 16, r.Top + 10, r.Right - 115, r.Top + 34}, 12, colInk, DT_LEFT, true)
		text(hdc, state, rect{r.Right - 105, r.Top + 10, r.Right - 16, r.Top + 34}, 9, accent, DT_RIGHT, true)
		multiText(hdc, f.Descricao, rect{r.Left + 16, r.Top + 40, r.Right - 16, r.Bottom - 38}, 10, colMuted, false)
		cost := "sem custo adicional"
		if f.CustoSemanal > 0 {
			cost = "custo simulado: " + money(f.CustoSemanal) + "/semana"
		} else if f.ID == "trafego_pago" {
			cost = "usa a verba definida em Marketing"
		}
		text(hdc, strings.ToUpper(cost), rect{r.Left + 16, r.Bottom - 32, r.Right - 16, r.Bottom - 10}, 9, colMuted, DT_LEFT, true)
		addHit("tool:"+f.ID, r)
	}
	cost := digitalToolCost(e)
	text(hdc, fmt.Sprintf("CUSTO SEMANAL DAS FERRAMENTAS: %s", money(cost)), rect{42, cr.Bottom - 72, 560, cr.Bottom - 34}, 10, colMuted, DT_LEFT, true)
	text(hdc, "Clique em uma ferramenta para ativar/desativar. Tráfego pago depende da verba de Marketing.", rect{560, cr.Bottom - 72, cr.Right - 330, cr.Bottom - 34}, 9, colMuted, DT_LEFT, false)
	button(hdc, "dashboard", "VOLTAR AO PAINEL", "Visão geral da empresa.", rect{cr.Right - 310, cr.Bottom - 78, cr.Right - 42, cr.Bottom - 20}, colInk)
	backButton(hdc, cr)
}

func drawJourney(hdc uintptr, cr rect) {
	e := ns.current
	if e == nil {
		ns.screen = "home"
		drawHome(hdc, cr)
		return
	}
	drawChrome(hdc, cr, "JORNADA JED", "trilha pedagógica em quatro passos")
	current := journeyStep(e)
	steps := []struct{ title, sub string }{
		{"PASSO 1 • SUA IDEIA DE NEGÓCIO", "Estruture a ideia, construa Lean Canvas e Persona."},
		{"PASSO 2 • SEU NEGÓCIO NA INTERNET", "Prepare produto/serviço e defina presença e canais digitais."},
		{"PASSO 3 • VENDA MAIS NA INTERNET", "Teste canais, conteúdo, marketplace e investimento em divulgação."},
		{"PASSO 4 • FERRAMENTAS DE APOIO", "Use métricas, design, UX e ferramentas de IA para aperfeiçoar decisões."},
	}
	y := int32(175)
	for i, st := range steps {
		idx := i + 1
		r := rect{90, y, cr.Right - 90, y + 108}
		fill(hdc, r, colPanel)
		accent := colLine
		status := "BLOQUEADO"
		if idx < current {
			accent = colInk
			status = "CONCLUÍDO / EM REVISÃO"
		}
		if idx == current {
			accent = colCyan
			status = "PASSO ATUAL"
		}
		line(hdc, r.Left, r.Top, r.Left, r.Bottom, accent, 7)
		text(hdc, st.title, rect{118, y + 12, cr.Right - 260, y + 40}, 15, colInk, DT_LEFT, true)
		multiText(hdc, st.sub, rect{118, y + 47, cr.Right - 260, y + 91}, 10, colMuted, false)
		text(hdc, status, rect{cr.Right - 245, y + 12, cr.Right - 112, y + 40}, 9, accent, DT_RIGHT, true)
		y += 124
	}
	text(hdc, "A jornada organiza a simulação; Canvas e Persona permanecem mapas vivos e podem ser revistos durante todo o percurso.", rect{90, cr.Bottom - 82, cr.Right - 350, cr.Bottom - 28}, 10, colMuted, DT_LEFT, false)
	button(hdc, "dashboard", "VOLTAR AO PAINEL", "Visão geral da empresa.", rect{cr.Right - 310, cr.Bottom - 78, cr.Right - 42, cr.Bottom - 20}, colInk)
	backButton(hdc, cr)
}

func drawIndicators(hdc uintptr, cr rect) {
	e := ns.current
	if e == nil {
		ns.screen = "home"
		drawHome(hdc, cr)
		return
	}
	drawChrome(hdc, cr, "INDICADORES", "desempenho acumulado")
	ind := indicators(e)
	if ind == nil {
		ind = &Indicadores{Caixa: e.Caixa, ClientesAtivos: e.ClientesAtivos}
	}
	s := score(e)
	drawGauge(hdc, 255, 380, 150, s.Total)
	text(hdc, "ÍNDICE DE DESENVOLVIMENTO", rect{100, 555, 410, 585}, 11, colMuted, DT_CENTER, true)
	labels := []struct {
		name   string
		v, max float64
		c      uintptr
	}{{"FINANCEIRO", s.Financeiro, 25, colCyan}, {"MERCADO", s.Mercado, 20, colBlue}, {"OPERAÇÃO", s.Operacao, 15, colInk}, {"HIPÓTESES", s.Hipoteses, 25, colAmber}, {"GESTÃO", s.Gestao, 15, colMagenta}}
	y := int32(190)
	for _, it := range labels {
		drawBar(hdc, 500, y, cr.Right-60, 34, it.name, it.v, it.max, it.c)
		y += 70
	}
	metricPanel(hdc, rect{500, 555, 760, 645}, "RECEITA", money(ind.Receita), colCyan)
	metricPanel(hdc, rect{780, 555, 1040, 645}, "RESULTADO", money(ind.Resultado), cashAccent(ind.Resultado))
	metricPanel(hdc, rect{1060, 555, cr.Right - 42, 645}, "CONVERSÃO", fmt.Sprintf("%.1f%%", ind.ConversaoAcumuladaPct), colBlue)
	button(hdc, "dashboard", "VOLTAR AO PAINEL", "Visão geral da empresa.", rect{cr.Right - 310, cr.Bottom - 78, cr.Right - 42, cr.Bottom - 20}, colInk)
	backButton(hdc, cr)
}
func drawBar(hdc uintptr, x, y, right, h int32, label string, v, max float64, c uintptr) {
	text(hdc, label, rect{x, y, right, y + 20}, 10, colMuted, DT_LEFT, true)
	rr := rect{x, y + 26, right, y + 38}
	fill(hdc, rr, colGrid)
	ratio := 0.0
	if max > 0 {
		ratio = math.Max(0, math.Min(1, v/max))
	}
	fill(hdc, rect{x, y + 26, x + int32(float64(right-x)*ratio), y + 38}, c)
	text(hdc, fmt.Sprintf("%.1f / %.0f", v, max), rect{x, y + 42, right, y + 65}, 10, colInk, DT_RIGHT, true)
}

func drawTutor(hdc uintptr, cr rect) {
	drawChrome(hdc, cr, "MODO TUTOR", "cenários, turmas e laboratório")
	classes := listClasses()
	scenarios := listScenarios()
	metricPanel(hdc, rect{42, 175, 330, 270}, "TURMAS", strconv.Itoa(len(classes)), colCyan)
	metricPanel(hdc, rect{350, 175, 638, 270}, "CENÁRIOS", strconv.Itoa(len(scenarios)), colBlue)
	metricPanel(hdc, rect{658, 175, 946, 270}, "CATÁLOGO", fmt.Sprintf("%d modelos", catalogCount(ns.cat)), colInk)
	y := int32(330)
	button(hdc, "newScenario", "NOVO CENÁRIO", "Criar ambiente econômico para uma turma.", rect{42, y, 420, y + 82}, colCyan)
	button(hdc, "newClass", "NOVA TURMA", "Vincular turma a um cenário.", rect{440, y, 818, y + 82}, colBlue)
	button(hdc, "balance", "LABORATÓRIO", "Executar calibração automática.", rect{838, y, cr.Right - 42, y + 82}, colInk)
	y += 105
	text(hdc, "TURMAS CADASTRADAS", rect{42, y, 450, y + 26}, 11, colMuted, DT_LEFT, true)
	y += 34
	for i, t := range classes {
		if y > cr.Bottom-100 {
			break
		}
		r := rect{42, y, cr.Right - 42, y + 46}
		fill(hdc, r, colPanel)
		line(hdc, r.Left, r.Bottom, r.Right, r.Bottom, colLine, 1)
		text(hdc, t.Nome, rect{56, y, r.Right - 450, y + 46}, 11, colInk, DT_LEFT, true)
		text(hdc, t.Cenario.Nome, rect{r.Right - 430, y, r.Right - 180, y + 46}, 10, colMuted, DT_LEFT, false)
		text(hdc, "COMPARAR →", rect{r.Right - 170, y, r.Right - 24, y + 46}, 10, colBlue, DT_RIGHT, true)
		addHit("compare:"+strconv.Itoa(i), r)
		y += 52
	}
	if ns.tutorStatus != "" {
		multiText(hdc, ns.tutorStatus, rect{42, cr.Bottom - 90, cr.Right - 260, cr.Bottom - 20}, 10, colMuted, false)
	}
	button(hdc, "home", "MENU PRINCIPAL", "Retornar ao início.", rect{cr.Right - 235, cr.Bottom - 82, cr.Right - 42, cr.Bottom - 22}, colInk)
}

func drawSaved(hdc uintptr, cr rect) {
	drawChrome(hdc, cr, "ABRIR SIMULAÇÃO", "empresas salvas neste computador")
	files := savedCompanyFiles()
	y := int32(170)
	if len(files) == 0 {
		multiText(hdc, "Nenhuma empresa salva foi encontrada. Crie um novo empreendimento primeiro.", rect{42, 180, 800, 250}, 13, colMuted, false)
	}
	for i, p := range files {
		if y+54 > cr.Bottom-70 {
			break
		}
		name := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
		button(hdc, "file:"+strconv.Itoa(i), strings.ReplaceAll(name, "_", " "), p, rect{42, y, cr.Right - 42, y + 52}, colCyan)
		y += 64
	}
	backButton(hdc, cr)
}

func savedCompanyFiles() []string {
	var out []string
	roots := []string{filepath.Join(dataDir(), "empresas"), classesDir()}
	for _, root := range roots {
		_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err == nil && info != nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(p), ".json") && filepath.Base(p) != "turma.json" {
				out = append(out, p)
			}
			return nil
		})
	}
	return out
}

func handleHit(id string) {
	switch {
	case id == "online":
		if onlineConfig.Token != "" {
			if err := onlineValidateSession(); err != nil {
				onlineConfig.Token = ""
				saveOnlineConfig()
			}
		}
		ns.screen = "online"
	case id == "online:login":
		nativeOnlineLogin()
		ns.screen = "online"
	case id == "online:registerstudent" || id == "online:registermentor" || id == "online:activate":
		nativeInfo("Cadastro", "Contas são cadastradas exclusivamente por Administradores.")
		ns.screen = "online"
	case id == "online:server":
		v, ok := nativePrompt("SERVIDOR JED", "Endereço do servidor", onlineConfig.ServerURL)
		if ok {
			onlineConfig.ServerURL = normalizeServerURL(v)
			saveOnlineConfig()
		}
	case id == "online:logout":
		onlineLogout()
		ns.screen = "online"
	case id == "online:password":
		nativeOnlineChangePassword()
	case id == "online:revoke":
		nativeOnlineRevokeInvite()
	case id == "online:classes":
		nativeOnlineClasses()
	case id == "online:createclass":
		nativeOnlineCreateClass()
	case id == "online:invite":
		nativeOnlineInvite()
	case id == "online:mentorinvite":
		nativeOnlineMentorInvite()
	case id == "online:mentorinvites":
		nativeOnlineMentorInvites()
	case id == "online:adminusers":
		nativeOnlineAdminUsers()
	case id == "online:admincreate":
		nativeOnlineAdminCreate()
	case id == "online:adminstatus":
		nativeOnlineAdminStatus()
	case id == "online:adminmentorperm":
		nativeOnlineAdminMentorPermission()
	case id == "online:admintransfer":
		nativeOnlineAdminTransfer()
	case id == "online:importcsv":
		nativeOnlineImportCSV()
	case id == "online:invites":
		nativeOnlineInvites()
	case id == "online:join":
		nativeOnlineJoin()
	case id == "online:sync":
		nativeOnlineSync()
	case id == "online:mycompanies":
		nativeOnlineMyCompanies()
	case id == "online:classcompanies":
		nativeOnlineClassCompanies()
	case id == "new":
		ns.screen = "catalog"
		ns.selectedSector = 0
		ns.selectedType = 0
	case id == "load":
		ns.screen = "saved"
	case id == "home":
		ns.screen = "home"
	case id == "back":
		switch ns.screen {
		case "dashboard", "catalog", "saved", "online":
			ns.screen = "home"
		default:
			if ns.current != nil {
				ns.screen = "dashboard"
			} else {
				ns.screen = "home"
			}
		}
	case id == "exit":
		pDestroyWindow.Call(ns.hwnd)
	case id == "tutorial":
		nativeInfo("Tutorial JED", `PASSO 1 — IDEIA, LEAN CANVAS E PERSONA

Estruture o problema, a solução, a oferta de valor e o perfil do cliente.

PASSO 2 — NEGÓCIO NA INTERNET

Escolha canais coerentes com a Persona e prepare sua operação digital.

PASSO 3 — VENDA MAIS

Acompanhe alcance, interações, leads, conversão, clientes e resultado.

PASSO 4 — APOIO E APRENDIZADO

Use Social Media, Copywriting, Design, Tráfego Pago, UI/UX e IA conforme a necessidade do negócio.
Compare alcance, interações, leads, conversão e resultado antes e depois das decisões.

O motor financeiro continua ativo por baixo de toda a jornada.`)
	case id == "tutor":
		ns.screen = "tutor"
	case id == "dashboard" || id == "overview":
		ns.screen = "dashboard"
	case id == "inventory":
		ns.screen = "inventory"
	case id == "finance":
		ns.screen = "finance"
	case id == "lean":
		ns.screen = "lean"
	case id == "persona":
		ns.screen = "persona"
	case id == "channels":
		ns.screen = "channels"
	case id == "tools":
		ns.screen = "tools"
	case id == "journey":
		ns.screen = "journey"
	case id == "indicators":
		ns.screen = "indicators"
	case strings.HasPrefix(id, "sector:"):
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "sector:"))
		ns.selectedSector = n
		ns.selectedType = 0
	case strings.HasPrefix(id, "type:"):
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "type:"))
		ns.selectedType = n
	case strings.HasPrefix(id, "model:"):
		mid := strings.TrimPrefix(id, "model:")
		if m, ok := findModel(ns.cat, mid); ok {
			createNativeCompany(m)
		}
	case strings.HasPrefix(id, "file:"):
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "file:"))
		f := savedCompanyFiles()
		if n >= 0 && n < len(f) {
			if e, err := loadCompany(f[n]); err == nil {
				ns.current = e
				ns.screen = "dashboard"
				ns.status = "SIMULAÇÃO CARREGADA • " + e.Nome
			} else {
				nativeInfo("Erro", err.Error())
			}
		}
	case id == "week":
		if ns.current != nil {
			_, _ = backupCompany(ns.current, "antes-semana")
			r := processWeek(ns.current)
			ns.last = r
			_, _ = saveCompany(ns.current)
			ns.status = fmt.Sprintf("SEMANA %d • vendas %d • resultado %s • caixa %s", r.Semana, r.Vendas, money(r.Resultado), money(r.Caixa))
			if r.Evento != "" {
				ns.status += " • evento registrado"
			}
			if r.Caixa < 0 || r.VendasPerdidas > 0 || r.Evento != "" {
				msg := fmt.Sprintf("Semana %d concluída.\n\nVendas: %d\nResultado: %s\nCaixa: %s", r.Semana, r.Vendas, money(r.Resultado), money(r.Caixa))
				if r.VendasPerdidas > 0 {
					msg += fmt.Sprintf("\nVendas perdidas: %d", r.VendasPerdidas)
				}
				if r.Evento != "" {
					msg += "\n\nEvento: " + r.Evento
				}
				nativeInfo("Checkpoint semanal", msg)
			}
		}
	case id == "price":
		if ns.current != nil {
			if v, ok := nativePromptFloat("ALTERAR VALOR", digitalModelLabel(ns.current), ns.current.Preco); ok {
				ns.current.Preco = v
				_, _ = saveCompany(ns.current)
				ns.status = "PREÇO ATUALIZADO • " + money(v)
			}
		}
	case id == "marketing":
		if ns.current != nil {
			if v, ok := nativePromptFloat("MARKETING", "Investimento semanal em marketing", ns.current.MarketingSemanal); ok {
				ns.current.MarketingSemanal = v
				_, _ = saveCompany(ns.current)
				ns.status = "MARKETING ATUALIZADO • " + money(v)
			}
		}
	case id == "promo":
		if ns.current != nil {
			if v, ok := nativePromptFloat("PROMOÇÃO", "Desconto para a próxima semana (%)", ns.current.PromocaoDesconto); ok {
				if v < 0 {
					v = 0
				}
				if v > 35 {
					v = 35
				}
				ns.current.PromocaoDesconto = v
				_, _ = saveCompany(ns.current)
				ns.status = fmt.Sprintf("PROMOÇÃO PROGRAMADA • %.1f%%", v)
			}
		}
	case strings.HasPrefix(id, "lean:"):
		if ns.current != nil {
			field := strings.TrimPrefix(id, "lean:")
			var label, cur string
			switch field {
			case "problema":
				label, cur = "Problema", ns.current.Canvas.Problema
			case "solucao":
				label, cur = "Solução", ns.current.Canvas.Solucao
			case "oferta":
				label, cur = "Oferta de Valor", ns.current.Canvas.PropostaValor
			case "vantagem":
				label, cur = "Vantagem Diferencial", ns.current.Canvas.Vantagem
			case "segmentos":
				label, cur = "Segmento de Clientes (separe por vírgulas)", strings.Join(ns.current.Canvas.Segmentos, ", ")
			case "metricas":
				label, cur = "Métricas (separe por vírgulas)", strings.Join(ns.current.Canvas.Metricas, ", ")
			case "canais":
				label, cur = "Canais do Lean Canvas (separe por vírgulas)", strings.Join(ns.current.Canvas.Canais, ", ")
			case "custos":
				label, cur = "Estrutura de Custos", ns.current.Canvas.CustosNotas
			case "receitas":
				label, cur = "Fontes de Receitas", ns.current.Canvas.ReceitaModelo
			}
			if v, ok := nativePrompt("LEAN CANVAS", label, cur); ok {
				split := func(x string) []string {
					out := []string{}
					for _, p := range strings.Split(x, ",") {
						p = strings.TrimSpace(p)
						if p != "" {
							out = append(out, p)
						}
					}
					return out
				}
				switch field {
				case "problema":
					ns.current.Canvas.Problema = v
				case "solucao":
					ns.current.Canvas.Solucao = v
				case "oferta":
					ns.current.Canvas.PropostaValor = v
				case "vantagem":
					ns.current.Canvas.Vantagem = v
				case "segmentos":
					ns.current.Canvas.Segmentos = split(v)
				case "metricas":
					ns.current.Canvas.Metricas = split(v)
				case "canais":
					ns.current.Canvas.Canais = split(v)
				case "custos":
					ns.current.Canvas.CustosNotas = v
				case "receitas":
					ns.current.Canvas.ReceitaModelo = v
				}
				_, _ = saveCompany(ns.current)
				ns.status = "LEAN CANVAS REVISADO"
			}
		}
	case strings.HasPrefix(id, "persona:"):
		if ns.current != nil {
			field := strings.TrimPrefix(id, "persona:")
			var label, cur string
			switch field {
			case "nome":
				label, cur = "Nome/identidade da Persona", ns.current.Persona.Nome
			case "demografia":
				label, cur = "Demografia", ns.current.Persona.Demografia
			case "rotinas":
				label, cur = "Rotinas", ns.current.Persona.Rotinas
			case "objetivos":
				label, cur = "Objetivos", ns.current.Persona.Objetivos
			case "desafios":
				label, cur = "Desafios", ns.current.Persona.Desafios
			case "motivadores":
				label, cur = "Motivadores", ns.current.Persona.Motivadores
			case "objecoes":
				label, cur = "Objeções", ns.current.Persona.Objecoes
			case "citacoes":
				label, cur = "Citações / frases típicas", ns.current.Persona.Citacoes
			case "palavras":
				label, cur = "Palavras-chave", ns.current.Persona.PalavrasChave
			}
			if v, ok := nativePrompt("CANVAS PERSONA", label, cur); ok {
				switch field {
				case "nome":
					ns.current.Persona.Nome = v
				case "demografia":
					ns.current.Persona.Demografia = v
				case "rotinas":
					ns.current.Persona.Rotinas = v
				case "objetivos":
					ns.current.Persona.Objetivos = v
				case "desafios":
					ns.current.Persona.Desafios = v
				case "motivadores":
					ns.current.Persona.Motivadores = v
				case "objecoes":
					ns.current.Persona.Objecoes = v
				case "citacoes":
					ns.current.Persona.Citacoes = v
				case "palavras":
					ns.current.Persona.PalavrasChave = v
				}
				_, _ = saveCompany(ns.current)
				ns.status = "PERSONA ATUALIZADA"
			}
		}
	case strings.HasPrefix(id, "channel:"):
		if ns.current != nil {
			cid := strings.TrimPrefix(id, "channel:")
			if contains(ns.current.CanaisDigitais, cid) {
				out := []string{}
				for _, v := range ns.current.CanaisDigitais {
					if v != cid {
						out = append(out, v)
					}
				}
				ns.current.CanaisDigitais = out
			} else {
				ns.current.CanaisDigitais = append(ns.current.CanaisDigitais, cid)
			}
			_, _ = saveCompany(ns.current)
			ns.status = "CANAIS DIGITAIS ATUALIZADOS"
		}
	case strings.HasPrefix(id, "tool:"):
		if ns.current != nil {
			tid := strings.TrimPrefix(id, "tool:")
			if contains(ns.current.FerramentasDigitais, tid) {
				out := []string{}
				for _, v := range ns.current.FerramentasDigitais {
					if v != tid {
						out = append(out, v)
					}
				}
				ns.current.FerramentasDigitais = out
			} else {
				ns.current.FerramentasDigitais = append(ns.current.FerramentasDigitais, tid)
			}
			_, _ = saveCompany(ns.current)
			ns.status = "FERRAMENTAS DIGITAIS ATUALIZADAS"
		}
	case id == "order":
		nativeQuickOrder()
	case strings.HasPrefix(id, "input:"):
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "input:"))
		nativeOrderForIndex(n)
	case id == "newScenario":
		nativeNewScenario()
	case id == "newClass":
		nativeNewClass()
	case id == "balance":
		nativeRunBalance()
	case strings.HasPrefix(id, "compare:"):
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "compare:"))
		cls := listClasses()
		if n >= 0 && n < len(cls) {
			nativeCompareClass(cls[n])
		}
	}
	invalidate()
}

func createNativeCompany(m Modelo) {
	name, ok := nativePrompt("NOVO EMPREENDIMENTO", "Nome da empresa", m.Nome)
	if !ok {
		return
	}
	resp, ok := nativePrompt("NOVO EMPREENDIMENTO", "Nome do aluno ou equipe", "Equipe")
	if !ok {
		return
	}
	capi, ok := nativePromptFloat("CAPITAL INICIAL", "Capital próprio disponível", 20000)
	if !ok {
		return
	}
	diff := nativeChoose("DIFICULDADE", "Escolha o nível", []string{"Iniciante", "Intermediário", "Avançado"}, 1)
	if diff < 0 {
		return
	}
	d := []string{"iniciante", "intermediario", "avancado"}[diff]
	profile := supplyProfileFor(m, ns.sc)
	ins := []InsumoEstoque{}
	estVal := 0.0
	for _, sp := range profile.Insumos {
		q := sp.EstoqueInicial
		ins = append(ins, InsumoEstoque{ID: sp.ID, Nome: sp.Nome, Unidade: sp.Unidade, Quantidade: q, CustoMedio: sp.CustoBase, CustoReferencia: sp.CustoBase, ConsumoPorVenda: sp.ConsumoPorVenda, ValidadeSemanas: sp.ValidadeSemanas, PerdaSemanal: sp.PerdaSemanal, Critico: sp.Critico})
		estVal += q * sp.CustoBase
	}
	equip := m.Equipamentos
	licenses := 500.0
	launch := 600.0
	cash := capi - equip - licenses - estVal - launch
	if cash < 0 {
		cash = math.Max(1000, capi*.20)
	}
	e := &Empresa{Nome: name, Responsavel: resp, ModeloBase: m.Nome, Categoria: m.SetorID, ModeloID: m.ID, Setor: m.SetorNome, TipoNegocio: m.TipoNome, Especialidade: m.Nome, PerecibilidadeSemanal: m.PerecibilidadeSemanal, Regulamentado: m.Regulamentado, CustoRegulatorioMensal: m.CustoRegulatorioMensal, ReputacaoImportancia: m.ReputacaoImportancia, DeliveryAfinidade: m.DeliveryAfinidade, Dificuldade: d, CapitalProprio: capi, Caixa: cash, Preco: m.PrecoRef, PrecoReferencia: m.PrecoRef, CustoUnitario: m.CustoUnitario, AlcanceBase: m.AlcanceBase, ConversaoBase: m.ConversaoBase, RecorrenciaBase: m.RecorrenciaBase, CapacidadeBase: m.CapacidadeBase, SazonalidadeAmplitude: m.Sazonalidade, SazonalidadeFase: m.Fase, UsaEstoque: m.Estoque, EstoqueUnidades: m.EstoqueInicialUn, EstoqueValor: estVal, CustoMedioEstoque: m.CustoUnitario, UsaInsumos: len(ins) > 0, Insumos: ins, Operacao: "hibrida", Imovel: "alugado", AluguelMensal: m.Aluguel, QualidadeLocalizacao: "media", Delivery: m.DeliveryAfinidade > 1.05, DeliveryTipo: "misto", Funcionarios: m.Funcionarios, SalarioMedio: 1800, MarketingSemanal: 150, MeiosPagamento: []string{"pix", "cartao"}, TaxaCartao: 2.5, PrazoCartaoSemanas: 1, PrazoFornecedorMax: 2, Cenario: "Mercado estável", CenarioAlcance: 1, CenarioConversao: 1, CenarioOscilacao: .08, DuracaoSemanas: 24, ConcorrenciaNivel: "media", ConcorrenciaIndice: 1, Reputacao: 50, Canvas: LeanCanvas{Problema: "Validar necessidade do cliente", Segmentos: []string{"geral"}, PropostaValor: "qualidade", Solucao: m.Nome, Canais: []string{"redes_sociais"}, ReceitaModelo: "venda_unica", CustosNotas: "Custos operacionais e aquisição", Metricas: []string{"alcance", "interações", "leads", "vendas", "caixa", "conversão"}, Vantagem: "A desenvolver"}, Persona: Persona{Nome: "Cliente principal", Demografia: "A definir", Rotinas: "A definir", Objetivos: "A definir", Desafios: "A definir", Motivadores: "A definir", Objecoes: "A definir", Citacoes: "A definir", PalavrasChave: "A definir"}, CanaisDigitais: []string{}, Hipoteses: Hipoteses{VendasSemanais: math.Max(5, float64(m.AlcanceBase)*m.ConversaoBase), FaturamentoSemanal: math.Max(5, float64(m.AlcanceBase)*m.ConversaoBase) * m.PrecoRef, EsperaResultadoPositivo: true, ConversaoPct: m.ConversaoBase * 100, RecorrenciaPct: m.RecorrenciaBase * 100}, InvestimentosIniciais: map[string]float64{"equipamentos": equip, "equipamentos_referencia": equip, "licencas": licenses, "estoque_inicial": estVal, "marketing_lancamento": launch, "capital_giro_inicial": cash}, VersaoDados: version}
	_, _ = saveCompany(e)
	ns.current = e
	ns.screen = "persona"
	ns.status = "EMPRESA CRIADA • defina a Persona, revise o Lean Canvas e escolha canais e ferramentas"
}

func nativeQuickOrder() {
	if ns.current == nil {
		return
	}
	if len(ns.current.Insumos) == 0 {
		nativeInfo("Compras", "Este negócio não possui insumos detalhados.")
		return
	}
	idx := 0
	for i, x := range ns.current.Insumos {
		if x.Critico {
			idx = i
			break
		}
	}
	nativeOrderForIndex(idx)
}
func nativeOrderForIndex(idx int) {
	e := ns.current
	if e == nil || idx < 0 || idx >= len(e.Insumos) {
		return
	}
	it := e.Insumos[idx]
	q, ok := nativePromptFloat("NOVO PEDIDO", it.Nome+" — quantidade ("+it.Unidade+")", math.Max(1, it.ConsumoPorVenda*20))
	if !ok {
		return
	}
	supIdx := nativeChoose("FORNECEDOR", "Escolha o fornecedor", supplierNames(ns.sc), 1)
	if supIdx < 0 || supIdx >= len(ns.sc.Fornecedores) {
		return
	}
	f := ns.sc.Fornecedores[supIdx]
	ok2, cost, msg := placeInputOrder(e, it.ID, q, f, minInt(1, f.PrazoPagamentoMax))
	if ok2 {
		_, _ = saveCompany(e)
		ns.status = "PEDIDO REGISTRADO • " + msg + " • " + money(cost)
	} else {
		nativeInfo("Pedido não realizado", msg)
	}
}
func supplierNames(c CatalogoInsumos) []string {
	r := []string{}
	for _, f := range c.Fornecedores {
		r = append(r, f.Nome)
	}
	return r
}

func nativeNewScenario() {
	name, ok := nativePrompt("NOVO CENÁRIO", "Nome do cenário", "Cenário da turma")
	if !ok {
		return
	}
	dur, ok := nativePromptFloat("NOVO CENÁRIO", "Duração em semanas", 24)
	if !ok {
		return
	}
	c := Cenario{Nome: name, Alcance: 1, Conversao: 1, Oscilacao: .08, Duracao: int(dur), ConcorrenciaNivel: "media", ConcorrenciaIndice: 1, Dificuldade: "intermediario"}
	p, err := saveScenario(c)
	if err != nil {
		nativeInfo("Erro", err.Error())
		return
	}
	ns.tutorStatus = "CENÁRIO CRIADO • " + p
}
func nativeNewClass() {
	scs := listScenarios()
	if len(scs) == 0 {
		nativeInfo("Turmas", "Crie primeiro um cenário.")
		return
	}
	names := []string{}
	for _, c := range scs {
		names = append(names, c.Nome)
	}
	idx := nativeChoose("NOVA TURMA", "Escolha o cenário", names, 0)
	if idx < 0 {
		return
	}
	name, ok := nativePrompt("NOVA TURMA", "Nome da turma", "Turma JED")
	if !ok {
		return
	}
	tutor, ok := nativePrompt("NOVA TURMA", "Nome do tutor", "Tutor")
	if !ok {
		return
	}
	t := Turma{ID: slug(name), Nome: name, Tutor: tutor, Cenario: scs[idx]}
	p, err := saveClass(t)
	if err != nil {
		nativeInfo("Erro", err.Error())
		return
	}
	ns.tutorStatus = "TURMA CRIADA • " + p
}
func nativeRunBalance() {
	ns.tutorStatus = "LABORATÓRIO EM EXECUÇÃO..."
	invalidate()
	rows, p, err := runBalanceLab(ns.cat, ns.sc, 1, "intermediario")
	if err != nil {
		nativeInfo("Laboratório", err.Error())
		return
	}
	ns.tutorStatus = fmt.Sprintf("LABORATÓRIO CONCLUÍDO • %d combinações • %s", len(rows), p)
}
func nativeCompareClass(t Turma) {
	p, rows, err := compareClass(t.ID)
	if err != nil {
		nativeInfo("Comparativo", err.Error())
		return
	}
	msg := fmt.Sprintf("Turma: %s\nEmpresas comparadas: %d\n\nCSV: %s", t.Nome, len(rows), p)
	if len(rows) > 0 {
		msg += "\n\nLíder atual: " + strings.Join(rows[0], " | ")
	}
	nativeInfo("Comparativo da turma", msg)
}

func nativeInfo(title, msg string) {
	pMessageBoxW.Call(ns.hwnd, uintptr(unsafe.Pointer(ptr(msg))), uintptr(unsafe.Pointer(ptr(title))), 0x40)
}

// Simple modal prompt implemented with a temporary native window and child controls.
var promptResult string
var promptDone, promptOK bool
var promptEdit uintptr
var promptSecretMode bool

const promptClass = "JEDPromptOrbit"

func nativePrompt(title, label, def string) (string, bool) {
	promptResult = def
	promptDone = false
	promptOK = false
	hInst, _, _ := pGetModuleHandleW.Call(0)
	cn := ptr(promptClass)
	cursor, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
	bigIcon, smallIcon := appIcons()
	wc := wndClassEx{CbSize: uint32(unsafe.Sizeof(wndClassEx{})), LpfnWndProc: syscall.NewCallback(promptWndProc), HInstance: hInst, HIcon: bigIcon, HCursor: cursor, LpszClassName: cn, HIconSm: smallIcon}
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	hwnd, _, _ := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(cn)), uintptr(unsafe.Pointer(ptr(title))), 0x00C80000|WS_VISIBLE, uintptr(520), uintptr(280), uintptr(520), uintptr(220), ns.hwnd, 0, hInst, 0)
	if bigIcon != 0 {
		pSendMessageW.Call(hwnd, WM_SETICON, ICON_BIG, bigIcon)
	}
	if smallIcon != 0 {
		pSendMessageW.Call(hwnd, WM_SETICON, ICON_SMALL, smallIcon)
	}
	if hwnd == 0 {
		return "", false
	}
	pEnableWindow.Call(ns.hwnd, 0)
	// static, edit, buttons
	pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(ptr("STATIC"))), uintptr(unsafe.Pointer(ptr(label))), WS_CHILD|WS_VISIBLE, 24, 24, 460, 28, hwnd, 0, hInst, 0)
	editStyle := uintptr(WS_CHILD | WS_VISIBLE | WS_BORDER | WS_TABSTOP | ES_AUTOHSCROLL)
	if promptSecretMode {
		editStyle |= 0x0020 // ES_PASSWORD
	}
	edit, _, _ := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(ptr("EDIT"))), uintptr(unsafe.Pointer(ptr(def))), editStyle, 24, 60, 460, 30, hwnd, 1001, hInst, 0)
	promptEdit = edit
	pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(ptr("BUTTON"))), uintptr(unsafe.Pointer(ptr("CONFIRMAR"))), WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 264, 118, 105, 34, hwnd, 1, hInst, 0)
	pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(ptr("BUTTON"))), uintptr(unsafe.Pointer(ptr("CANCELAR"))), WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 379, 118, 105, 34, hwnd, 2, hInst, 0)
	pSetFocus.Call(edit)
	var m msg
	for !promptDone {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	pEnableWindow.Call(ns.hwnd, 1)
	pSetFocus.Call(ns.hwnd)
	return promptResult, promptOK
}
func promptWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case WM_COMMAND:
		id := int32(wParam & 0xffff)
		if id == 1 {
			buf := make([]uint16, 1024)
			getText := modUser32.NewProc("GetWindowTextW")
			getText.Call(promptEdit, uintptr(unsafe.Pointer(&buf[0])), 1024)
			promptResult = syscall.UTF16ToString(buf)
			promptOK = true
			promptDone = true
			pDestroyWindow.Call(hwnd)
			return 0
		}
		if id == 2 {
			promptDone = true
			promptOK = false
			pDestroyWindow.Call(hwnd)
			return 0
		}
	case WM_CLOSE:
		promptDone = true
		promptOK = false
		pDestroyWindow.Call(hwnd)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return r
}
func nativePromptSecret(title, label string) (string, bool) {
	promptSecretMode = true
	defer func() { promptSecretMode = false }()
	return nativePrompt(title, label, "")
}

func nativePromptFloat(title, label string, def float64) (float64, bool) {
	s, ok := nativePrompt(title, label, strconv.FormatFloat(def, 'f', 2, 64))
	if !ok {
		return 0, false
	}
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", ".")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		nativeInfo("Valor inválido", "Digite um número válido.")
		return 0, false
	}
	return v, true
}
func nativeChoose(title, label string, items []string, def int) int {
	if len(items) == 0 {
		return -1
	}
	msg := label + "\n\n"
	for i, it := range items {
		msg += fmt.Sprintf("%d — %s\n", i+1, it)
	}
	msg += "\nDigite o número da opção."
	s, ok := nativePrompt(title, msg, strconv.Itoa(def+1))
	if !ok {
		return -1
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 1 || n > len(items) {
		nativeInfo("Opção inválida", "Escolha um número da lista.")
		return -1
	}
	return n - 1
}

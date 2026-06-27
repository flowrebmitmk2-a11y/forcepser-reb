package voiceroid2

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"syscall"

	"github.com/oov/forcepser/fairy/internal"
	"github.com/zzl/go-win32api/win32"
)

var (
	mainWindowSaveButtonCaptions = []string{"音声保存", "名前を付けて保存", "音声ファイルを保存"}
	fileMenuItemCaptions         = []string{"ファイル"}
	saveMenuItemCaptions         = []string{"音声ファイルを保存"}
	menuAccessKeyPattern         = regexp.MustCompile(`\((?:&|_)?[^)]\)`)
	ignoredCharacterNames        = map[string]struct{}{
		"システム": {},
		"最小化":  {},
		"最大化":  {},
		"閉じる":  {},
		"ファイル": {},
		"編集":   {},
		"テキスト": {},
		"マスター": {},
		"ボイス":  {},
		"フレーズ": {},
		"単語":   {},
		"表示":   {},
		"ツール":  {},
		"ヘルプ":  {},
		"標準":   {},
		"ユーザー": {},
		"再生":   {},
		"停止":   {},
		"先頭":   {},
		"末尾":   {},
		"音声保存": {},
		"再生時間": {},
	}
)

type mainWindow struct {
	uia    *internal.UIAutomation
	window *internal.Element
	save   *internal.Element
}

type characterNameCandidate struct {
	name  string
	count int
}

func (mw *mainWindow) Release() {
	if mw.save != nil {
		mw.save.Release()
		mw.save = nil
	}
	if mw.window != nil {
		mw.window.Release()
		mw.window = nil
	}
}

func normalizeMenuCaption(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexRune(s, '\t'); idx >= 0 {
		s = s[:idx]
	}
	s = menuAccessKeyPattern.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&", "")
	s = strings.TrimRight(s, ".…")
	return strings.TrimSpace(s)
}

func menuCaptionMatches(caption string, candidates []string) bool {
	normalized := normalizeMenuCaption(caption)
	for _, candidate := range candidates {
		if normalized == normalizeMenuCaption(candidate) {
			return true
		}
	}
	return false
}

func menuCaptionHasPrefix(caption string, candidates []string) bool {
	normalized := normalizeMenuCaption(caption)
	for _, candidate := range candidates {
		if strings.HasPrefix(normalized, normalizeMenuCaption(candidate)) {
			return true
		}
	}
	return false
}

func shouldIgnoreCharacterName(name string) bool {
	if name == "" {
		return true
	}
	if _, found := ignoredCharacterNames[name]; found {
		return true
	}
	if strings.HasSuffix(name, "文字") {
		return true
	}
	for _, r := range name {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func chooseCharacterName(candidates []characterNameCandidate) string {
	if len(candidates) == 0 {
		return "unknown"
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].count != candidates[j].count {
			return candidates[i].count > candidates[j].count
		}
		if len([]rune(candidates[i].name)) != len([]rune(candidates[j].name)) {
			return len([]rune(candidates[i].name)) > len([]rune(candidates[j].name))
		}
		return candidates[i].name < candidates[j].name
	})
	return candidates[0].name
}

func getElementStringProperty(elem *internal.Element, propertyID int32) string {
	s, err := elem.GetCurrentPropertyStringValue(propertyID)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

func readElementText(elem *internal.Element) string {
	candidates := []func() (string, error){
		elem.GetTextViaTextPattern,
		elem.GetTextViaValuePattern,
		func() (string, error) {
			hwnd, err := elem.GetNativeWindowHandle()
			if err != nil {
				return "", err
			}
			return internal.GetWindowText(hwnd), nil
		},
	}
	for _, candidate := range candidates {
		text, err := candidate()
		if err != nil {
			continue
		}
		text = strings.TrimSpace(text)
		if text != "" {
			return text
		}
	}
	return ""
}

func (mw *mainWindow) enumerateDescendants() (*internal.Elements, error) {
	trueCond, err := mw.uia.CreateTrueCondition()
	if err != nil {
		return nil, fmt.Errorf("failed to create true condition: %w", err)
	}
	defer trueCond.Release()

	elems, err := mw.window.FindAll(win32.TreeScope_Descendants, trueCond)
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate main window elements: %w", err)
	}
	return elems, nil
}

func (mw *mainWindow) resolveCharacterName() string {
	elems, err := mw.enumerateDescendants()
	if err != nil {
		return "unknown"
	}
	defer elems.Release()

	counts := map[string]int{}
	for i := 0; i < elems.Len; i++ {
		elem, err := elems.Get(i)
		if err != nil {
			continue
		}
		name, err := elem.GetName()
		elem.Release()
		if err != nil {
			continue
		}
		name = normalizeMenuCaption(name)
		if shouldIgnoreCharacterName(name) {
			continue
		}
		counts[name]++
	}

	candidates := make([]characterNameCandidate, 0, len(counts))
	for name, count := range counts {
		candidates = append(candidates, characterNameCandidate{name: name, count: count})
	}
	return chooseCharacterName(candidates)
}

func (mw *mainWindow) resolveText() string {
	elems, err := mw.enumerateDescendants()
	if err != nil {
		return ""
	}
	defer elems.Release()

	for i := 0; i < elems.Len; i++ {
		elem, err := elems.Get(i)
		if err != nil {
			continue
		}
		automationID := getElementStringProperty(elem, win32.UIA_AutomationIdPropertyId)
		className := getElementStringProperty(elem, win32.UIA_ClassNamePropertyId)
		text := ""
		switch {
		case automationID == "TextBox", className == "TextBox", automationID == "c", className == "TextEditView":
			logElementDiagnostic(fmt.Sprintf("text candidate[%d]", i), elem)
			text = readElementText(elem)
		}
		elem.Release()
		if text != "" {
			diagf("voiceroid2 diag: selected main text candidate index=%d text_len=%d text_preview=%q", i, len([]rune(text)), previewText(text))
			return text
		}
	}
	diagf("voiceroid2 diag: main text candidate not found")
	return ""
}

func getMenuString(menu win32.HMENU, pos uint32) string {
	buf := make([]uint16, 256)
	n := win32.GetMenuStringW(menu, pos, &buf[0], int32(len(buf)), win32.MF_BYPOSITION)
	if n <= 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:n])
}

func findMenuItemID(menu win32.HMENU, captions []string) (uint32, error) {
	count, err := win32.GetMenuItemCount(menu)
	if count < 0 {
		return 0, fmt.Errorf("GetMenuItemCount failed: %w", err)
	}
	for i := int32(0); i < count; i++ {
		state := win32.GetMenuState(menu, uint32(i), win32.MF_BYPOSITION)
		if state&uint32(win32.MF_SEPARATOR) != 0 {
			continue
		}
		caption := getMenuString(menu, uint32(i))
		if !menuCaptionHasPrefix(caption, captions) {
			continue
		}
		id := win32.GetMenuItemID(menu, i)
		if id == 0xFFFFFFFF {
			continue
		}
		diagf("voiceroid2 diag: matched menu item caption=%q id=%d", caption, id)
		return id, nil
	}
	return 0, fmt.Errorf("menu item not found")
}

func findSubMenu(menu win32.HMENU, captions []string) (win32.HMENU, error) {
	count, err := win32.GetMenuItemCount(menu)
	if count < 0 {
		return 0, fmt.Errorf("GetMenuItemCount failed: %w", err)
	}
	for i := int32(0); i < count; i++ {
		caption := getMenuString(menu, uint32(i))
		if !menuCaptionHasPrefix(caption, captions) {
			continue
		}
		sub := win32.GetSubMenu(menu, i)
		if sub != 0 {
			diagf("voiceroid2 diag: matched submenu caption=%q", caption)
			return sub, nil
		}
	}
	return 0, fmt.Errorf("submenu not found")
}

func invokeSaveViaMenu(hwnd win32.HWND) error {
	menu := win32.GetMenu(hwnd)
	if menu == 0 {
		return fmt.Errorf("window menu not found")
	}
	fileMenu, err := findSubMenu(menu, fileMenuItemCaptions)
	if err != nil {
		return fmt.Errorf("failed to find file menu: %w", err)
	}
	id, err := findMenuItemID(fileMenu, saveMenuItemCaptions)
	if err != nil {
		return fmt.Errorf("failed to find save menu item in file menu: %w", err)
	}
	diagf("voiceroid2 diag: invoking save via menu command id=%d", id)
	win32.SendMessage(hwnd, win32.WM_COMMAND, win32.WPARAM(id), 0)
	return nil
}

func isClickableControlType(ctrlType int32) bool {
	switch ctrlType {
	case win32.UIA_ButtonControlTypeId, win32.UIA_MenuItemControlTypeId, win32.UIA_HyperlinkControlTypeId, win32.UIA_CustomControlTypeId:
		return true
	default:
		return false
	}
}

func findSaveButton(uia *internal.UIAutomation, window *internal.Element) (*internal.Element, error) {
	trueCond, err := uia.CreateTrueCondition()
	if err != nil {
		return nil, fmt.Errorf("failed to create true condition: %w", err)
	}
	defer trueCond.Release()

	elems, err := window.FindAll(win32.TreeScope_Descendants, trueCond)
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate main window elements: %w", err)
	}
	defer elems.Release()

	var lastClickable *internal.Element
	for i := 0; i < elems.Len; i++ {
		elem, err := elems.Get(i)
		if err != nil {
			if lastClickable != nil {
				lastClickable.Release()
			}
			return nil, fmt.Errorf("failed to get main window element: %w", err)
		}

		ctrlType, ctrlErr := elem.GetControlType()
		if ctrlErr == nil && isClickableControlType(ctrlType) {
			if lastClickable != nil {
				lastClickable.Release()
			}
			elem.AddRef()
			lastClickable = elem
		}

		name, err := elem.GetName()
		if err != nil || !menuCaptionHasPrefix(name, mainWindowSaveButtonCaptions) {
			elem.Release()
			continue
		}

		diagf("voiceroid2 diag: matched save caption index=%d raw_name=%q", i, name)
		logElementDiagnostic("save caption match", elem)
		if ctrlErr == nil && isClickableControlType(ctrlType) {
			if lastClickable != nil && lastClickable != elem {
				lastClickable.Release()
			}
			logElementDiagnostic("save action direct match", elem)
			return elem, nil
		}
		if lastClickable != nil {
			logElementDiagnostic("save action previous clickable", lastClickable)
			elem.Release()
			return lastClickable, nil
		}
		elem.Release()
	}
	if lastClickable != nil {
		lastClickable.Release()
	}
	return nil, fmt.Errorf("save action element not found")
}

func invokeElement(elem *internal.Element, window win32.HWND) error {
	logElementDiagnostic("invoke target", elem)
	invokeErr := elem.Invoke()
	if invokeErr == nil {
		diagf("voiceroid2 diag: invoke path succeeded via InvokePattern")
		return nil
	}
	diagf("voiceroid2 diag: invoke path InvokePattern failed: %v", invokeErr)
	defaultErr := elem.DoDefaultAction()
	if defaultErr == nil {
		diagf("voiceroid2 diag: invoke path succeeded via LegacyIAccessible default action")
		return nil
	}
	diagf("voiceroid2 diag: invoke path LegacyIAccessible default action failed: %v", defaultErr)
	mouseErr := elem.LeftClickViaMouseClick(window)
	if mouseErr == nil {
		diagf("voiceroid2 diag: invoke path succeeded via mouse click")
		return nil
	}
	diagf("voiceroid2 diag: invoke path mouse click failed: %v", mouseErr)
	return fmt.Errorf("invoke failed (%v), default action failed (%v), and mouse click failed (%w)", invokeErr, defaultErr, mouseErr)
}

func newMainWindow(uia *internal.UIAutomation, hwnd win32.HWND) (*mainWindow, error) {
	window, err := uia.ElementFromHandle(hwnd)
	if err != nil {
		return nil, fmt.Errorf("failed to create element from native window handle: %w", err)
	}
	defer window.Release()

	r := &mainWindow{uia: uia}
	window.AddRef()
	r.window = window

	save, err := findSaveButton(uia, window)
	if err == nil {
		defer save.Release()
		save.AddRef()
		r.save = save
		diagln("voiceroid2 diag: cached main window save action element")
	} else {
		diagf("voiceroid2 diag: failed to cache main window save action element: %v", err)
	}
	return r, nil
}

func (mw *mainWindow) invokeSave(hwnd win32.HWND) error {
	win32.SetForegroundWindow(hwnd)

	var saveErr error
	if mw.save != nil {
		diagln("voiceroid2 diag: trying main window save action element")
		saveErr = invokeElement(mw.save, hwnd)
		if saveErr == nil {
			return nil
		}
		diagf("voiceroid2 diag: main window save action failed: %v", saveErr)
	}

	diagln("voiceroid2 diag: falling back to menu save action")
	menuErr := invokeSaveViaMenu(hwnd)
	if menuErr == nil {
		return nil
	}
	diagf("voiceroid2 diag: menu save action failed: %v", menuErr)
	logMainWindowDiagnostics(mw.uia, mw.window)
	if saveErr != nil {
		return fmt.Errorf("failed to invoke save button (%v) and menu (%w)", saveErr, menuErr)
	}
	return menuErr
}

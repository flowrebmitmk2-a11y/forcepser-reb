package voiceroid2

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"github.com/oov/forcepser/fairy/internal"
	"github.com/zzl/go-win32api/win32"
)

const (
	saveDialogFileNameComboID = 1148
	saveDialogFileNameEditID  = 1152
)

type saveDialog struct {
	window   *internal.Element
	edit     *internal.Element
	editHWND win32.HWND
	button   *internal.Element
}

func (sd *saveDialog) Release() {
	if sd.window != nil {
		sd.window.Release()
		sd.window = nil
	}
	if sd.edit != nil {
		sd.edit.Release()
		sd.edit = nil
	}
	if sd.button != nil {
		sd.button.Release()
		sd.button = nil
	}
}

func windowClassName(hwnd win32.HWND) string {
	if hwnd == 0 {
		return ""
	}
	buf := make([]uint16, 256)
	win32.GetClassNameW(hwnd, &buf[0], int32(len(buf)))
	return syscall.UTF16ToString(buf)
}

func logWindowHandleDiagnostic(prefix string, hwnd win32.HWND) {
	if hwnd == 0 {
		diagf("voiceroid2 diag: %s hwnd=0", prefix)
		return
	}
	diagf("voiceroid2 diag: %s hwnd=%d class=%q text=%q", prefix, hwnd, windowClassName(hwnd), previewText(internal.GetWindowText(hwnd)))
}

func findChildWindowByClass(parent win32.HWND, className string) win32.HWND {
	if parent == 0 {
		return 0
	}
	classPtr, err := syscall.UTF16PtrFromString(className)
	if err != nil {
		return 0
	}
	hwnd, _ := win32.FindWindowExW(parent, 0, classPtr, nil)
	return hwnd
}

func findSaveDialogEditHandle(dialogHWND win32.HWND) (win32.HWND, error) {
	if dialogHWND == 0 {
		return 0, fmt.Errorf("dialog hwnd is zero")
	}

	editHWND, _ := win32.GetDlgItem(dialogHWND, saveDialogFileNameEditID)
	if editHWND != 0 {
		return editHWND, nil
	}

	comboHWND, _ := win32.GetDlgItem(dialogHWND, saveDialogFileNameComboID)
	if comboHWND == 0 {
		return 0, fmt.Errorf("GetDlgItem failed for file name controls")
	}

	for _, hwnd := range []win32.HWND{
		comboHWND,
		findChildWindowByClass(comboHWND, "ComboBox"),
		findChildWindowByClass(comboHWND, "Edit"),
	} {
		if hwnd == 0 {
			continue
		}
		if windowClassName(hwnd) == "Edit" {
			return hwnd, nil
		}
		child := findChildWindowByClass(hwnd, "Edit")
		if child != 0 {
			return child, nil
		}
	}
	return 0, fmt.Errorf("file name edit handle not found")
}

func setWindowText(hwnd win32.HWND, text string) error {
	ptr, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return fmt.Errorf("failed to encode text: %w", err)
	}
	result, _ := win32.SendMessageW(hwnd, win32.WM_SETTEXT, 0, uintptr(unsafe.Pointer(ptr)))
	if result == 0 {
		return fmt.Errorf("WM_SETTEXT failed")
	}
	return nil
}

func logSaveDialogEditState(prefix string, elem *internal.Element, hwnd win32.HWND) {
	if elem != nil {
		logElementDiagnostic(prefix+" ui", elem)
		text, err := elem.GetTextViaValuePattern()
		logTextProbe(prefix+" ui via ValuePattern", text, err)
		text, err = elem.GetTextViaTextPattern()
		logTextProbe(prefix+" ui via TextPattern", text, err)
		nativeHWND, err := elem.GetNativeWindowHandle()
		if err != nil {
			diagf("voiceroid2 diag: %s ui native handle lookup failed: %v", prefix, err)
		} else {
			logWindowHandleDiagnostic(prefix+" ui native", nativeHWND)
		}
	}
	logWindowHandleDiagnostic(prefix+" hwnd", hwnd)
}

func (sd *saveDialog) SetFileName(filename string) error {
	filename = strings.TrimSpace(filename)
	diagf("voiceroid2 diag: attempting save dialog filename=%q", filename)
	if filename == "" {
		return fmt.Errorf("filename is empty")
	}

	logSaveDialogEditState("save dialog edit before set", sd.edit, sd.editHWND)

	var nativeErr error
	if sd.editHWND != 0 {
		nativeErr = setWindowText(sd.editHWND, filename)
		if nativeErr == nil {
			diagf("voiceroid2 diag: save dialog filename set via WM_SETTEXT")
			logSaveDialogEditState("save dialog edit after WM_SETTEXT", sd.edit, sd.editHWND)
			return nil
		}
		diagf("voiceroid2 diag: WM_SETTEXT failed: %v", nativeErr)
	}

	if sd.edit == nil {
		if nativeErr != nil {
			return fmt.Errorf("failed to set save dialog file name natively: %w", nativeErr)
		}
		return fmt.Errorf("save dialog edit control is unavailable")
	}

	valueErr := sd.edit.SetTextViaValuePattern(filename)
	if valueErr == nil {
		diagf("voiceroid2 diag: save dialog filename set via ValuePattern")
		logSaveDialogEditState("save dialog edit after ValuePattern", sd.edit, sd.editHWND)
		return nil
	}
	diagf("voiceroid2 diag: ValuePattern failed: %v", valueErr)

	legacyErr := sd.edit.SetTextViaLegacyIAccessiblePattern(filename)
	if legacyErr == nil {
		diagf("voiceroid2 diag: save dialog filename set via LegacyIAccessible")
		logSaveDialogEditState("save dialog edit after LegacyIAccessible", sd.edit, sd.editHWND)
		return nil
	}
	diagf("voiceroid2 diag: LegacyIAccessible failed: %v", legacyErr)

	hwnd, hwndErr := sd.edit.GetNativeWindowHandle()
	if hwndErr == nil {
		wmCharErr := sd.edit.SetTextViaWMChar(hwnd, filename)
		if wmCharErr == nil {
			diagf("voiceroid2 diag: save dialog filename set via WM_CHAR")
			logSaveDialogEditState("save dialog edit after WM_CHAR", sd.edit, sd.editHWND)
			return nil
		}
		diagf("voiceroid2 diag: WM_CHAR failed: %v", wmCharErr)
		logSaveDialogEditState("save dialog edit after failed WM_CHAR", sd.edit, sd.editHWND)
		return fmt.Errorf("native path failed (%v), value pattern failed (%v), legacy accessibility failed (%v), and wm_char failed (%w)", nativeErr, valueErr, legacyErr, wmCharErr)
	}

	diagf("voiceroid2 diag: ui native handle lookup failed: %v", hwndErr)
	logSaveDialogEditState("save dialog edit after failed ui native handle lookup", sd.edit, sd.editHWND)
	return fmt.Errorf("native path failed (%v), value pattern failed (%v), legacy accessibility failed (%v), and ui native handle lookup failed (%w)", nativeErr, valueErr, legacyErr, hwndErr)
}

func findLegacyDescendantControl(uia *internal.UIAutomation, window *internal.Element, controlType int32, controlID int) (*internal.Element, error) {
	ctrlCond, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, controlType)
	if err != nil {
		return nil, fmt.Errorf("failed to create control type condition: %w", err)
	}
	defer ctrlCond.Release()

	idCond, err := uia.CreateStringPropertyConditionEx(win32.UIA_AutomationIdPropertyId, fmt.Sprint(controlID), 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create id condition: %w", err)
	}
	defer idCond.Release()

	cond, err := uia.CreateAndCondition(ctrlCond, idCond)
	if err != nil {
		return nil, fmt.Errorf("failed to create and condition: %w", err)
	}
	defer cond.Release()

	elem, err := window.FindFirst(win32.TreeScope_Descendants, cond)
	if err != nil {
		return nil, fmt.Errorf("element not found: %w", err)
	}
	return elem, nil
}

func isSaveDialogEditCandidate(elem *internal.Element) bool {
	ctrlType, err := elem.GetControlType()
	if err != nil || ctrlType != win32.UIA_EditControlTypeId {
		return false
	}
	automationID := getElementStringProperty(elem, win32.UIA_AutomationIdPropertyId)
	if strings.HasPrefix(automationID, "System.") {
		return false
	}
	return true
}

func findSaveDialogEdit(uia *internal.UIAutomation, window *internal.Element) (*internal.Element, error) {
	trueCond, err := uia.CreateTrueCondition()
	if err != nil {
		return nil, fmt.Errorf("failed to create true condition: %w", err)
	}
	defer trueCond.Release()

	elems, err := window.FindAll(win32.TreeScope_Descendants, trueCond)
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate dialog elements: %w", err)
	}
	defer elems.Release()

	for i := 0; i < elems.Len; i++ {
		elem, err := elems.Get(i)
		if err != nil {
			return nil, fmt.Errorf("failed to get dialog element: %w", err)
		}
		if isSaveDialogEditCandidate(elem) {
			diagf("voiceroid2 diag: found save dialog edit control at index=%d", i)
			logElementDiagnostic("save dialog edit candidate", elem)
			return elem, nil
		}
		elem.Release()
	}
	logSaveDialogDiagnostics(uia, window)
	return nil, fmt.Errorf("save file name edit control not found")
}

func findSaveDialog(uia *internal.UIAutomation, pid win32.DWORD, mainWindow win32.HWND) (*saveDialog, error) {
	windowHandle := internal.FindWindow(0, saveDialogClass, "", uint32(pid), func(h win32.HWND) bool {
		return h != mainWindow
	})
	if windowHandle == 0 {
		return nil, fmt.Errorf("save dialog not found")
	}
	diagf("voiceroid2 diag: found save dialog hwnd=%d", windowHandle)

	nativeEditHWND, nativeEditErr := findSaveDialogEditHandle(windowHandle)
	if nativeEditErr != nil {
		diagf("voiceroid2 diag: failed to resolve save dialog native edit handle: %v", nativeEditErr)
	} else {
		logWindowHandleDiagnostic("save dialog native edit", nativeEditHWND)
	}

	elem, err := uia.ElementFromHandle(windowHandle)
	if err != nil {
		return nil, fmt.Errorf("failed to get save dialog: %w", err)
	}
	defer elem.Release()

	cndFramework, err := uia.CreateStringPropertyConditionEx(win32.UIA_FrameworkIdPropertyId, saveDialogFramework, 0)
	if err != nil {
		return nil, fmt.Errorf("CreateStringPropertyCondition failed: %w", err)
	}
	defer cndFramework.Release()

	dialogElem, err := elem.FindFirst(win32.TreeScope_Element, cndFramework)
	if err != nil {
		return nil, fmt.Errorf("save dialog framework not matched: %w", err)
	}
	defer dialogElem.Release()
	logElementDiagnostic("save dialog root", dialogElem)
	logSaveDialogDiagnostics(uia, dialogElem)

	var editElem *internal.Element
	editElem, err = findSaveDialogEdit(uia, dialogElem)
	if err != nil {
		if nativeEditHWND == 0 {
			return nil, fmt.Errorf("edit control not found in save dialog: %w", err)
		}
		diagf("voiceroid2 diag: proceeding without UIA save dialog edit: %v", err)
	} else {
		defer editElem.Release()
	}

	buttonElem, err := findLegacyDescendantControl(uia, dialogElem, win32.UIA_ButtonControlTypeId, saveDialogButtonID)
	if err != nil {
		return nil, fmt.Errorf("button control not found in save dialog: %w", err)
	}
	defer buttonElem.Release()
	diagln("voiceroid2 diag: found save dialog confirm button")
	logElementDiagnostic("save dialog confirm button", buttonElem)

	dialogElem.AddRef()
	buttonElem.AddRef()
	if editElem != nil {
		editElem.AddRef()
	}
	return &saveDialog{
		window:   dialogElem,
		edit:     editElem,
		editHWND: nativeEditHWND,
		button:   buttonElem,
	}, nil
}

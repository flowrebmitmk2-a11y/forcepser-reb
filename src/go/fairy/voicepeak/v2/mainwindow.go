package voicepeak

import (
	"fmt"
	"strings"

	"github.com/oov/forcepser/fairy/internal"

	"github.com/zzl/go-win32api/win32"
)

type mainWindow struct {
	window *internal.Element
	combo  *internal.Element
	edit   *internal.Element
	button *internal.Element
}

func (mw *mainWindow) Release() {
	if mw.window != nil {
		mw.window.Release()
		mw.window = nil
	}
	if mw.combo != nil {
		mw.combo.Release()
		mw.combo = nil
	}
	if mw.edit != nil {
		mw.edit.Release()
		mw.edit = nil
	}
	if mw.button != nil {
		mw.button.Release()
		mw.button = nil
	}
}

type mainWindowControlInfo struct {
	sourceIndex int
	controlType int32
	name        string
}

func isMainWindowRelevantControlType(controlType int32) bool {
	return controlType == win32.UIA_ButtonControlTypeId ||
		controlType == win32.UIA_EditControlTypeId ||
		controlType == win32.UIA_TextControlTypeId
}

func isMainWindowCharacterNameControlType(controlType int32) bool {
	return controlType == win32.UIA_ButtonControlTypeId ||
		controlType == win32.UIA_TextControlTypeId
}

func findMainWindowControlIndexesOfType(controls []mainWindowControlInfo, editControlType int32, preferredSourceIndex int) (comboIndex int, buttonIndex int, editIndex int, ok bool) {
	for i := range controls {
		if controls[i].controlType != editControlType {
			continue
		}
		if preferredSourceIndex >= 0 && controls[i].sourceIndex != preferredSourceIndex {
			continue
		}

		for buttonCandidate := i - 1; buttonCandidate >= 1; buttonCandidate-- {
			if controls[buttonCandidate].controlType != win32.UIA_ButtonControlTypeId || controls[buttonCandidate].name != mainWindowIconButtonName {
				continue
			}

			fallbackComboCandidate := -1
			for comboCandidate := buttonCandidate - 1; comboCandidate >= 0; comboCandidate-- {
				if !isMainWindowCharacterNameControlType(controls[comboCandidate].controlType) || controls[comboCandidate].name == mainWindowIconButtonName {
					continue
				}
				if fallbackComboCandidate == -1 {
					fallbackComboCandidate = comboCandidate
				}
				if strings.TrimSpace(controls[comboCandidate].name) == "" {
					continue
				}

				return comboCandidate, buttonCandidate, i, true
			}
			if fallbackComboCandidate != -1 {
				return fallbackComboCandidate, buttonCandidate, i, true
			}
		}
	}
	return 0, 0, 0, false
}

func findMainWindowControlIndexes(controls []mainWindowControlInfo, preferredSourceIndex int) (comboIndex int, buttonIndex int, editIndex int, ok bool) {
	if preferredSourceIndex >= 0 {
		comboIndex, buttonIndex, editIndex, ok = findMainWindowControlIndexesOfType(controls, win32.UIA_EditControlTypeId, preferredSourceIndex)
		if ok {
			return comboIndex, buttonIndex, editIndex, true
		}
		comboIndex, buttonIndex, editIndex, ok = findMainWindowControlIndexesOfType(controls, win32.UIA_TextControlTypeId, preferredSourceIndex)
		if ok {
			return comboIndex, buttonIndex, editIndex, true
		}
	}

	comboIndex, buttonIndex, editIndex, ok = findMainWindowControlIndexesOfType(controls, win32.UIA_EditControlTypeId, -1)
	if ok {
		return comboIndex, buttonIndex, editIndex, true
	}
	return findMainWindowControlIndexesOfType(controls, win32.UIA_TextControlTypeId, -1)
}

func setMainWindowControlsFromIndexes(elems *internal.Elements, comboIndex int, buttonIndex int, editIndex int, out *mainWindow) error {
	comboElem, err := elems.Get(comboIndex)
	if err != nil {
		return fmt.Errorf("failed to get combo box element: %w", err)
	}
	defer comboElem.Release()

	buttonElem, err := elems.Get(buttonIndex)
	if err != nil {
		return fmt.Errorf("failed to get image button element: %w", err)
	}
	defer buttonElem.Release()

	editElem, err := elems.Get(editIndex)
	if err != nil {
		return fmt.Errorf("failed to get edit element: %w", err)
	}
	defer editElem.Release()

	comboElem.AddRef()
	out.combo = comboElem
	buttonElem.AddRef()
	out.button = buttonElem
	editElem.AddRef()
	out.edit = editElem
	return nil
}

func findFocusedSourceIndex(uia *internal.UIAutomation, elems *internal.Elements) (int, error) {
	focused, err := uia.GetFocusedElement()
	if err != nil {
		return -1, nil
	}
	defer focused.Release()

	for i := 0; i < elems.Len; i++ {
		elem, err := elems.Get(i)
		if err != nil {
			return -1, fmt.Errorf("failed to get main window element: %w", err)
		}
		same, err := uia.CompareElements(elem, focused)
		elem.Release()
		if err != nil {
			return -1, fmt.Errorf("failed to compare focused element: %w", err)
		}
		if same {
			return i, nil
		}
	}
	return -1, nil
}

func findMainWindowControls(uia *internal.UIAutomation, elems *internal.Elements, out *mainWindow) error {
	controls := make([]mainWindowControlInfo, 0, elems.Len)
	for i := 0; i < elems.Len; i++ {
		elem, err := elems.Get(i)
		if err != nil {
			return fmt.Errorf("failed to get main window element: %w", err)
		}

		ctrlType, err := elem.GetControlType()
		if err != nil {
			elem.Release()
			return fmt.Errorf("failed to get main window control type: %w", err)
		}
		if !isMainWindowRelevantControlType(ctrlType) {
			elem.Release()
			continue
		}

		info := mainWindowControlInfo{
			sourceIndex: i,
			controlType: ctrlType,
		}
		if ctrlType == win32.UIA_ButtonControlTypeId || ctrlType == win32.UIA_TextControlTypeId {
			info.name, err = elem.GetName()
			if err != nil {
				elem.Release()
				return fmt.Errorf("failed to get main window control name: %w", err)
			}
		}
		controls = append(controls, info)
		elem.Release()
	}

	preferredSourceIndex, err := findFocusedSourceIndex(uia, elems)
	if err != nil {
		return err
	}

	comboIndex, buttonIndex, editIndex, ok := findMainWindowControlIndexes(controls, preferredSourceIndex)
	if !ok {
		return internal.ErrElementNotFound
	}
	return setMainWindowControlsFromIndexes(
		elems,
		controls[comboIndex].sourceIndex,
		controls[buttonIndex].sourceIndex,
		controls[editIndex].sourceIndex,
		out,
	)
}

func findMainWindowElement(uia *internal.UIAutomation, hwnd win32.HWND) (*internal.Element, error) {
	var conds []*win32.IUIAutomationCondition
	cndCtrl, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, win32.UIA_WindowControlTypeId)
	if err != nil {
		return nil, fmt.Errorf("failed to create control type condition: %w", err)
	}
	defer cndCtrl.Release()
	conds = append(conds, cndCtrl)

	cndName, err := uia.CreateStringPropertyConditionEx(win32.UIA_NamePropertyId, mainWindowName, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create name condition: %w", err)
	}
	defer cndName.Release()
	conds = append(conds, cndName)

	cndFramework, err := uia.CreateStringPropertyConditionEx(win32.UIA_FrameworkIdPropertyId, mainWindowFramework, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create framework condition: %w", err)
	}
	defer cndFramework.Release()
	conds = append(conds, cndFramework)

	cond, err := uia.CreateAndCondition(conds...)
	if err != nil {
		return nil, fmt.Errorf("failed to create and condition: %w", err)
	}
	defer cond.Release()

	var window *internal.Element
	if hwnd != 0 && hwnd != win32.INVALID_HANDLE_VALUE {
		elem, err := uia.ElementFromHandle(hwnd)
		if err != nil {
			return nil, fmt.Errorf("failed to create element from native window handle: %w", err)
		}
		defer elem.Release()
		window, err = elem.FindFirst(win32.TreeScope_Element, cond)
		if err != nil {
			return nil, fmt.Errorf("FindFirst failed: %w", err)
		}
	} else {
		window, err = uia.FindTopElement(cond)
		if err != nil {
			return nil, fmt.Errorf("FindFirst failed: %w", err)
		}
	}
	return window, nil
}

func newMainWindow(uia *internal.UIAutomation, hwnd win32.HWND) (*mainWindow, error) {
	window, err := findMainWindowElement(uia, hwnd)
	if err != nil {
		return nil, err
	}
	defer window.Release()

	r := mainWindow{
		window: window,
	}
	scopes := []win32.TreeScope{
		win32.TreeScope_Children,
		win32.TreeScope_Descendants,
	}
	for _, scope := range scopes {
		cndTrue, err := uia.CreateTrueCondition()
		if err != nil {
			return nil, fmt.Errorf("failed to create true condition: %w", err)
		}

		elems, err := window.FindAll(scope, cndTrue)
		cndTrue.Release()
		if err != nil {
			if err == internal.ErrElementNotFound {
				continue
			}
			return nil, fmt.Errorf("failed to get window elements: %w", err)
		}

		err = findMainWindowControls(uia, elems, &r)
		elems.Release()
		if err == nil {
			break
		}
		if err != nil && err != internal.ErrElementNotFound {
			return nil, err
		}
	}
	if r.combo == nil || r.edit == nil || r.button == nil {
		return nil, fmt.Errorf("main window controls not found in children or descendants")
	}
	defer r.combo.Release()
	defer r.edit.Release()
	defer r.button.Release()

	window.AddRef()
	r.combo.AddRef()
	r.edit.AddRef()
	r.button.AddRef()
	return &r, nil
}

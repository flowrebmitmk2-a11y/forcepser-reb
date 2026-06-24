package voicepeak

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oov/forcepser/fairy"
	"github.com/oov/forcepser/fairy/internal"
	"github.com/zzl/go-win32api/win32"
)

var (
	exeName = "voicepeak.exe"

	mainWindowName           = "Voicepeak"
	mainWindowFramework      = "JUCE"
	mainWindowIconButtonName = "ImageIconButton"

	exportDialogTitles           = []string{"Export Settings", "出力設定", "导出设置"}
	exportDialogFilenameLabels   = []string{"Save as", "ファイル名", "文件名"}
	exportDialogNamingRuleLabels = []string{"Enable Naming Rule", "命名規則を有効", "启用命名规则"}
	exportDialogButtonCaptions   = []string{"Export", "出力", "导出"}

	blockExportMenuItemCaptions = []string{"Export Block", "ブロックの出力", "导出对话框"}

	folderSelectDialogClass     = "#32770"
	folderSelectDialogFramework = "Win32"
	folderSelectDialogEditID    = 1152
	folderSelectDialogButtonID  = 1

	windowCreationTimeout       = 5 * time.Second
	windowCreationCheckInterval = 40 * time.Millisecond
	fileCreationTimeout         = 1 * time.Minute
	fileCreationCheckInterval   = 100 * time.Millisecond
)

type voicepeak struct{}

var legacyFairyCall bool

func New() fairy.Fairy {
	return &voicepeak{}
}

func sendBlockExportShortcut(hwnd win32.HWND) error {
	// Release modifiers first so a held hotkey does not turn F9 into another shortcut.
	_, err := internal.SendInput([]internal.Input{
		{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_CONTROL, Flags: internal.KEYEVENTF_KEYUP}},
		{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_SHIFT, Flags: internal.KEYEVENTF_KEYUP}},
		{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_MENU, Flags: internal.KEYEVENTF_KEYUP}},
		{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_RMENU, Flags: internal.KEYEVENTF_KEYUP}},
	})
	if err != nil {
		return fmt.Errorf("failed to send input: %w", err)
	}

	win32.SendMessage(hwnd, win32.WM_KEYDOWN, win32.WPARAM(win32.VK_F9), 1)
	win32.SendMessage(hwnd, win32.WM_KEYUP, win32.WPARAM(win32.VK_F9), 0xc0000001)
	return nil
}

func waitBlockExportDialog(uia *internal.UIAutomation, pid win32.DWORD, hwnd win32.HWND) (*blockExportDialog, error) {
	var blockExportDialog *blockExportDialog
	var err error
	for deadLine := time.Now().Add(windowCreationTimeout); ; time.Sleep(windowCreationCheckInterval) {
		if time.Now().After(deadLine) {
			return nil, fmt.Errorf("waiting for block export dialog creation timed out: %w", err)
		}
		blockExportDialog, err = findBlockExportDialog(uia, pid, hwnd, legacyFairyCall)
		if err != nil {
			continue
		}
		return blockExportDialog, nil
	}
}

func resolveCharacterName(mainWindow *mainWindow) (string, error) {
	candidates := []func() (string, error){
		mainWindow.combo.GetName,
		mainWindow.combo.GetTextViaValuePattern,
		func() (string, error) {
			return mainWindow.combo.GetCurrentPropertyStringValue(win32.UIA_ValueValuePropertyId)
		},
	}
	for _, candidate := range candidates {
		name, err := candidate()
		if err != nil {
			continue
		}
		name = strings.TrimSpace(name)
		if name != "" {
			return name, nil
		}
	}
	if legacyFairyCall {
		return "", fmt.Errorf("character name is empty")
	}
	return "unknown", nil
}

func SetLegacyFairyCall(v bool) {
	legacyFairyCall = v
}

func (vp *voicepeak) IsTarget(hwnd win32.HWND, exePath string) bool {
	return filepath.Base(exePath) == exeName
}

func (vp *voicepeak) TestedProgram() string {
	return "VOICEPEAK 1.2.22"
}

func buildExportDir(namer func(name, text string) (string, error)) (string, error) {
	wavPath, err := namer("", "")
	if err != nil {
		return "", fmt.Errorf("failed to build export path: %w", err)
	}
	return filepath.Dir(wavPath), nil
}

func (vp *voicepeak) Execute(hwnd win32.HWND, namer func(name, text string) (string, error)) error {
	var pid uint32
	win32.GetWindowThreadProcessId(hwnd, &pid)

	uia, err := internal.New()
	if err != nil {
		return fmt.Errorf("failed to create IUIAutomation: %w", err)
	}

	exportDir, err := buildExportDir(namer)
	if err != nil {
		return err
	}

	var mainWindow *mainWindow
	if legacyFairyCall {
		mainWindow, err = newMainWindow(uia, hwnd)
		if err != nil {
			return fmt.Errorf("main window not found: %w", err)
		}
		defer mainWindow.Release()

		err = mainWindow.window.SetEnable(false)
		if err != nil {
			return fmt.Errorf("failed to disable window")
		}
		defer mainWindow.window.SetEnable(true)
	}

	name := ""
	text := ""

	if legacyFairyCall {
		name, err = resolveCharacterName(mainWindow)
		if err != nil {
			return fmt.Errorf("failed to get character name: %w", err)
		}

		text, err = mainWindow.edit.GetTextViaTextPattern()
		if err != nil {
			return fmt.Errorf("failed to get text: %w", err)
		}
		if text == "" {
			return fmt.Errorf("text is empty")
		}

		// get current caret position
		tr, err := mainWindow.edit.GetFirstSelection()
		if err != nil {
			return fmt.Errorf("failed to get edit text pattern: %w", err)
		}
		if tr != nil {
			// restore focus and caret on exit, if available
			defer func() {
				mainWindow.edit.SetFocus()
				tr.Select()
				tr.Release()
			}()
		}

		wavPath, err := namer(name, text)
		if err != nil {
			return fmt.Errorf("failed to build filename: %w", err)
		}
		_, err = os.Stat(wavPath)
		if err == nil {
			return fmt.Errorf("file %v already exists: %w", wavPath, os.ErrExist)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to test file %v: %w", wavPath, err)
		}
	}

	err = sendBlockExportShortcut(hwnd)
	if err != nil {
		return fmt.Errorf("failed to trigger block export shortcut: %w", err)
	}

	blockExportDialog, err := waitBlockExportDialog(uia, pid, hwnd)
	if err != nil {
		return err
	}
	defer blockExportDialog.Release()

	var wavPath string
	if legacyFairyCall {
		wavPath, err = namer(name, text)
		if err != nil {
			return fmt.Errorf("failed to build filename: %w", err)
		}

		// set export file name
		behwnd, err := blockExportDialog.window.GetNativeWindowHandle()
		if err != nil {
			return fmt.Errorf("failed to get dialog window handle: %w", err)
		}
		err = blockExportDialog.edit.SetTextViaWMChar(behwnd, filepath.Base(changeExt(wavPath, "")))
		if err != nil {
			return fmt.Errorf("failed to set text: %w", err)
		}

		// disable naming rule if enabled
		chk, err := blockExportDialog.namingRuleCheckBox.GetCurrentPropertyStringValue(win32.UIA_ValueValuePropertyId)
		if err != nil {
			return fmt.Errorf("failed to get checkbox state: %w", err)
		}
		if chk == "On" {
			err = blockExportDialog.namingRuleCheckBox.Invoke()
			if err != nil {
				return fmt.Errorf("failed to set checkbox state: %w", err)
			}
		}
	}

	// click export button
	err = blockExportDialog.button.Invoke()
	if err != nil {
		return fmt.Errorf("failed to click export button: %w", err)
	}

	// find folder select dialog
	var folderSelectDialog *folderSelectDialog
	for deadLine := time.Now().Add(windowCreationTimeout); ; time.Sleep(windowCreationCheckInterval) {
		if time.Now().After(deadLine) {
			return fmt.Errorf("waiting for folder select dialog creation timed out: %w", err)
		}
		folderSelectDialog, err = findFolderSelectDialog(uia, pid, hwnd)
		if err != nil {
			continue
		}
		break
	}
	defer folderSelectDialog.Release()

	// input export folder
	err = folderSelectDialog.edit.SetTextViaValuePattern(exportDir)
	if err != nil {
		return fmt.Errorf("failed to input export folder: %w", err)
	}

	if legacyFairyCall {
		mainWindow.window.SetEnable(true)
	}

	// click button
	err = folderSelectDialog.button.Invoke()
	if err != nil {
		return fmt.Errorf("failed to click select button: %w", err)
	}

	if !legacyFairyCall {
		return nil
	}

	// wait file creation
	for deadLine := time.Now().Add(fileCreationTimeout); ; time.Sleep(fileCreationCheckInterval) {
		if time.Now().After(deadLine) {
			return fmt.Errorf("waiting for file creation timed out: %w", os.ErrDeadlineExceeded)
		}
		f, err := os.OpenFile(wavPath, 0666, fs.FileMode(os.O_RDWR|os.O_APPEND))
		if err == nil {
			f.Close()
			break
		}
	}

	// create text file
	f, err := os.Create(changeExt(wavPath, ".txt"))
	if err != nil {
		return fmt.Errorf("failed to create text file: %w", err)
	}
	defer f.Close()

	_, err = f.Write([]byte{0xef, 0xbb, 0xbf})
	if err != nil {
		return fmt.Errorf("failed to write UTF-8 BOM: %w", err)
	}

	_, err = f.WriteString(text)
	if err != nil {
		return fmt.Errorf("failed to write text: %w", err)
	}

	return nil
}

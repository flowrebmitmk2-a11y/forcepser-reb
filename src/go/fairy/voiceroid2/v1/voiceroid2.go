package voiceroid2

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/oov/forcepser/fairy"
	"github.com/oov/forcepser/fairy/internal"
	"github.com/zzl/go-win32api/win32"
)

var (
	exeName = "VoiceroidEditor.exe"

	defaultSaveFileNamePrefix = "ボイロ2"
	saveFileNamePrefix        = defaultSaveFileNamePrefix

	saveDialogClass     = "#32770"
	saveDialogFramework = "Win32"
	saveDialogButtonID  = 1

	windowCreationTimeout       = 5 * time.Second
	windowCreationCheckInterval = 40 * time.Millisecond
)

type voiceroid2 struct{}

func New() fairy.Fairy {
	return &voiceroid2{}
}

func (vr *voiceroid2) IsTarget(hwnd win32.HWND, exePath string) bool {
	return filepath.Base(exePath) == exeName
}

func (vr *voiceroid2) TestedProgram() string {
	return "VOICEROID2"
}

func SetSaveFileNamePrefix(prefix string) {
	if prefix == "" {
		saveFileNamePrefix = defaultSaveFileNamePrefix
		return
	}
	saveFileNamePrefix = prefix
}

func buildSaveFileName() string {
	return fmt.Sprintf("%s_%d.wav", saveFileNamePrefix, time.Now().UnixMilli())
}

func (vr *voiceroid2) Execute(hwnd win32.HWND, namer func(name, text string) (string, error)) error {
	var pid uint32
	win32.GetWindowThreadProcessId(hwnd, &pid)
	diagf("voiceroid2 diag: execute start hwnd=%d pid=%d", hwnd, pid)

	uia, err := internal.New()
	if err != nil {
		return fmt.Errorf("failed to create IUIAutomation: %w", err)
	}

	mainWindow, err := newMainWindow(uia, hwnd)
	if err != nil {
		return fmt.Errorf("main window not found: %w", err)
	}
	defer mainWindow.Release()

	name := mainWindow.resolveCharacterName()
	text := mainWindow.resolveText()
	filename := buildSaveFileName()
	logResolvedState(name, text, filename)

	diagln("voiceroid2 diag: triggering save action")
	if err := mainWindow.invokeSave(hwnd); err != nil {
		return fmt.Errorf("failed to trigger save action: %w", err)
	}

	diagf("voiceroid2 diag: waiting for save dialog timeout=%s interval=%s", windowCreationTimeout, windowCreationCheckInterval)
	var saveDialog *saveDialog
	for deadLine := time.Now().Add(windowCreationTimeout); ; time.Sleep(windowCreationCheckInterval) {
		if time.Now().After(deadLine) {
			return fmt.Errorf("waiting for save dialog creation timed out: %w", err)
		}
		saveDialog, err = findSaveDialog(uia, win32.DWORD(pid), hwnd)
		if err != nil {
			continue
		}
		break
	}
	defer saveDialog.Release()
	diagln("voiceroid2 diag: save dialog is ready")

	if err := saveDialog.SetFileName(filename); err != nil {
		return fmt.Errorf("failed to input save file name: %w", err)
	}

	dialogHWND, hwndErr := saveDialog.window.GetNativeWindowHandle()
	if hwndErr != nil {
		return fmt.Errorf("failed to get save dialog window handle: %w", hwndErr)
	}
	diagf("voiceroid2 diag: clicking save dialog confirm button hwnd=%d", dialogHWND)
	if err := invokeElement(saveDialog.button, dialogHWND); err != nil {
		return fmt.Errorf("failed to click save button in dialog: %w", err)
	}

	win32.SetForegroundWindow(hwnd)
	diagln("voiceroid2 diag: execute completed")
	return nil
}

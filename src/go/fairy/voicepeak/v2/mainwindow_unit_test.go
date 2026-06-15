package voicepeak

import (
	"testing"

	"github.com/zzl/go-win32api/win32"
)

func TestFindMainWindowControlIndexes(t *testing.T) {
	t.Run("finds sequence with extra irrelevant controls removed by filtering", func(t *testing.T) {
		controls := []mainWindowControlInfo{
			{sourceIndex: 1, controlType: win32.UIA_ButtonControlTypeId, name: "Narrator"},
			{sourceIndex: 4, controlType: win32.UIA_ButtonControlTypeId, name: mainWindowIconButtonName},
			{sourceIndex: 9, controlType: win32.UIA_EditControlTypeId},
		}

		comboIndex, buttonIndex, editIndex, ok := findMainWindowControlIndexes(controls, -1)
		if !ok {
			t.Fatal("expected controls to be found")
		}
		if comboIndex != 0 || buttonIndex != 1 || editIndex != 2 {
			t.Fatalf("unexpected indexes: combo=%d button=%d edit=%d", comboIndex, buttonIndex, editIndex)
		}
	})

	t.Run("accepts text control as editor", func(t *testing.T) {
		controls := []mainWindowControlInfo{
			{sourceIndex: 0, controlType: win32.UIA_ButtonControlTypeId, name: "Speaker"},
			{sourceIndex: 1, controlType: win32.UIA_ButtonControlTypeId, name: mainWindowIconButtonName},
			{sourceIndex: 2, controlType: win32.UIA_TextControlTypeId},
		}

		_, _, _, ok := findMainWindowControlIndexes(controls, -1)
		if !ok {
			t.Fatal("expected text control to be accepted")
		}
	})

	t.Run("accepts extra relevant controls inserted by newer UI", func(t *testing.T) {
		controls := []mainWindowControlInfo{
			{sourceIndex: 0, controlType: win32.UIA_ButtonControlTypeId, name: "Speaker"},
			{sourceIndex: 1, controlType: win32.UIA_TextControlTypeId},
			{sourceIndex: 2, controlType: win32.UIA_ButtonControlTypeId, name: mainWindowIconButtonName},
			{sourceIndex: 3, controlType: win32.UIA_TextControlTypeId},
			{sourceIndex: 4, controlType: win32.UIA_EditControlTypeId},
		}

		comboIndex, buttonIndex, editIndex, ok := findMainWindowControlIndexes(controls, -1)
		if !ok {
			t.Fatal("expected controls to be found")
		}
		if comboIndex != 0 || buttonIndex != 2 || editIndex != 4 {
			t.Fatalf("unexpected indexes: combo=%d button=%d edit=%d", comboIndex, buttonIndex, editIndex)
		}
	})

	t.Run("rejects unrelated button order", func(t *testing.T) {
		controls := []mainWindowControlInfo{
			{sourceIndex: 0, controlType: win32.UIA_ButtonControlTypeId, name: mainWindowIconButtonName},
			{sourceIndex: 1, controlType: win32.UIA_ButtonControlTypeId, name: "Speaker"},
			{sourceIndex: 2, controlType: win32.UIA_EditControlTypeId},
		}

		_, _, _, ok := findMainWindowControlIndexes(controls, -1)
		if ok {
			t.Fatal("expected invalid order to be rejected")
		}
	})

	t.Run("prefers focused edit when multiple candidates exist", func(t *testing.T) {
		controls := []mainWindowControlInfo{
			{sourceIndex: 0, controlType: win32.UIA_ButtonControlTypeId, name: "Speaker A"},
			{sourceIndex: 1, controlType: win32.UIA_ButtonControlTypeId, name: mainWindowIconButtonName},
			{sourceIndex: 2, controlType: win32.UIA_EditControlTypeId},
			{sourceIndex: 3, controlType: win32.UIA_ButtonControlTypeId, name: "Speaker B"},
			{sourceIndex: 4, controlType: win32.UIA_ButtonControlTypeId, name: mainWindowIconButtonName},
			{sourceIndex: 5, controlType: win32.UIA_EditControlTypeId},
		}

		comboIndex, buttonIndex, editIndex, ok := findMainWindowControlIndexes(controls, 5)
		if !ok {
			t.Fatal("expected focused controls to be found")
		}
		if comboIndex != 3 || buttonIndex != 4 || editIndex != 5 {
			t.Fatalf("unexpected indexes: combo=%d button=%d edit=%d", comboIndex, buttonIndex, editIndex)
		}
	})
}

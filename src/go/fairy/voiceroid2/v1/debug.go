package voiceroid2

import (
	"fmt"
	"log"
	"strings"

	"github.com/oov/forcepser/fairy/internal"
	"github.com/zzl/go-win32api/win32"
)

const (
	voiceroid2Debug       = false
	maxDiagnosticElements = 80
)

func diagf(format string, args ...any) {
	if !voiceroid2Debug {
		return
	}
	log.Printf(format, args...)
}

func diagln(args ...any) {
	if !voiceroid2Debug {
		return
	}
	log.Println(args...)
}
func diagnosticString(elem *internal.Element, propertyID int32) string {
	s, err := elem.GetCurrentPropertyStringValue(propertyID)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

func diagnosticInt32(elem *internal.Element, propertyID int32) int32 {
	v, err := elem.GetCurrentPropertyInt32Value(propertyID)
	if err != nil {
		return 0
	}
	return v
}

func previewText(text string) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", " "))
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	const maxRunes = 40
	runes := []rune(text)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "..."
	}
	return text
}

func logResolvedState(name string, text string, filename string) {
	diagf("voiceroid2 diag: resolved name=%q text_len=%d text_preview=%q filename=%q", name, len([]rune(text)), previewText(text), filename)
}

func logTextProbe(prefix string, text string, err error) {
	if err != nil {
		diagf("voiceroid2 diag: %s probe failed: %v", prefix, err)
		return
	}
	diagf("voiceroid2 diag: %s probe text_len=%d text_preview=%q", prefix, len([]rune(text)), previewText(text))
}

func logElementDiagnostic(prefix string, elem *internal.Element) {
	name, _ := elem.GetName()
	className := diagnosticString(elem, win32.UIA_ClassNamePropertyId)
	automationID := diagnosticString(elem, win32.UIA_AutomationIdPropertyId)
	framework := diagnosticString(elem, win32.UIA_FrameworkIdPropertyId)
	ctrlType, _ := elem.GetControlType()
	invoke := diagnosticInt32(elem, win32.UIA_IsInvokePatternAvailablePropertyId)
	legacy := diagnosticInt32(elem, win32.UIA_IsLegacyIAccessiblePatternAvailablePropertyId)
	value := diagnosticInt32(elem, win32.UIA_IsValuePatternAvailablePropertyId)
	diagf("voiceroid2 diag: %s name=%q class=%q automationId=%q framework=%q controlType=%d invoke=%d legacy=%d value=%d", prefix, name, className, automationID, framework, ctrlType, invoke, legacy, value)
}

func logWindowDiagnostics(uia *internal.UIAutomation, window *internal.Element, label string) {
	if uia == nil || window == nil {
		return
	}
	diagf("voiceroid2 diag: dumping %s descendants", label)
	trueCond, err := uia.CreateTrueCondition()
	if err != nil {
		diagf("voiceroid2 diag: failed to create true condition: %v", err)
		return
	}
	defer trueCond.Release()

	elems, err := window.FindAll(win32.TreeScope_Descendants, trueCond)
	if err != nil {
		diagf("voiceroid2 diag: failed to enumerate descendants: %v", err)
		return
	}
	defer elems.Release()

	limit := elems.Len
	if limit > maxDiagnosticElements {
		limit = maxDiagnosticElements
	}
	for i := 0; i < limit; i++ {
		elem, err := elems.Get(i)
		if err != nil {
			diagf("voiceroid2 diag: failed to get descendant %d: %v", i, err)
			continue
		}
		logElementDiagnostic(fmt.Sprintf("descendant[%d]", i), elem)
		elem.Release()
	}
	if elems.Len > limit {
		diagf("voiceroid2 diag: omitted %d descendants", elems.Len-limit)
	}
}

func logMainWindowDiagnostics(uia *internal.UIAutomation, window *internal.Element) {
	logWindowDiagnostics(uia, window, "main window")
}

func logSaveDialogDiagnostics(uia *internal.UIAutomation, window *internal.Element) {
	logWindowDiagnostics(uia, window, "save dialog")
}

package jobs

import "fmt"

const (
	LogGlyphOK   = "✓"
	LogGlyphSkip = "↷"
	LogGlyphFail = "✗"
	LogGlyphInfo = "·"
	LogGlyphWarn = "!"
)

func LogOK(msg string) string   { return LogGlyphOK + " " + msg }
func LogSkip(msg string) string { return LogGlyphSkip + " " + msg }
func LogFail(msg string) string { return LogGlyphFail + " " + msg }
func LogInfo(msg string) string { return LogGlyphInfo + " " + msg }
func LogWarn(msg string) string { return LogGlyphWarn + " " + msg }

func LogInfof(format string, args ...any) string {
	return LogInfo(fmt.Sprintf(format, args...))
}

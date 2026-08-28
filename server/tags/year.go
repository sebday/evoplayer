package tags

// YearFromText extracts a calendar year from free text (title, filename stem, etc.).
func YearFromText(text string) int {
	return yearFromFilename(text)
}

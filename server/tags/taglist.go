package tags

import "strings"

// TagKey groups "Space Jazz" with "space jazz". "spacejazz" stays separate.
func TagKey(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	return spaceRe.ReplaceAllString(name, " ")
}

// ParseTagList splits a tags field on semicolons or commas and keeps the first spelling of each tag.
func ParseTagList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ';' || r == ','
	})
	var out []string
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(spaceRe.ReplaceAllString(part, " "))
		if part == "" {
			continue
		}
		key := TagKey(part)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, part)
	}
	return out
}

// FormatTagList writes tags the way they are stored in ID3.
func FormatTagList(list []string) string {
	return strings.Join(ParseTagList(strings.Join(list, "; ")), "; ")
}

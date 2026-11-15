package secrets

import "testing"

func TestMatchSoundcloudOAuth(t *testing.T) {
	legacy := "2-305776-1459234-abcdefghij"
	jwt := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	cases := map[string]string{
		"noise " + legacy + " tail": legacy,
		jwt:                         jwt,
		"prefix\x00" + jwt + "\n":   jwt,
		jwt + " " + legacy:          jwt,
		"":                          "",
		"not-a-token":               "",
	}
	for in, want := range cases {
		if got := matchSoundcloudOAuth(in); got != want {
			t.Errorf("matchSoundcloudOAuth(%q) = %q, want %q", in, got, want)
		}
	}
}

package soundcloud

import "strings"

func SetsURL(user string) string {
	user = strings.TrimSpace(user)
	if user == "" {
		user = defaultUser
	}
	return "https://soundcloud.com/" + user + "/sets"
}

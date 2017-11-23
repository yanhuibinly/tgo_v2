package util

import "strings"

func StringIsEmpty(data string) bool {
	return strings.Trim(data, " ") == ""
}

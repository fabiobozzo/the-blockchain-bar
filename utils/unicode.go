package utils

import (
	"fmt"
	"strconv"
	"strings"
)

func Unicode(s string) string {
	r, _ := strconv.ParseInt(strings.TrimPrefix(s, "\\U"), 16, 32)

	return fmt.Sprint(r)
}

package debug

import (
	"os"
	"strconv"
	"strings"
)

// EnvVars dumps env vars trimming tail part for security.
func EnvVars(trimAfter int) map[string]string {
	vars := make(map[string]string)

	for _, v := range os.Environ() {
		vv := strings.SplitN(v, "=", 2)
		head := vv[1]

		if len(head) > trimAfter {
			head = head[0:trimAfter] + "...[" + strconv.Itoa(len(head)) + "]"
		}

		vars[vv[0]] = head
	}

	return vars
}

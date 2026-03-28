package tools

import "strconv"

func argInt(args map[string]string, key string, def int) int {
	s := args[key]
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

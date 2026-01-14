package ui

import "strings"

type entryItem struct {
	path     string
	name     string
	isDir    bool
	isParent bool
	selected bool
}

func filterEntries(entries []entryItem, filter string) []entryItem {
	if filter == "" {
		return entries
	}
	needle := strings.ToLower(filter)
	out := make([]entryItem, 0)
	for _, e := range entries {
		if e.isParent {
			out = append(out, e)
			continue
		}
		if strings.Contains(strings.ToLower(e.name), needle) {
			out = append(out, e)
		}
	}
	return out
}

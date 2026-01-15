package core

import "strings"

type EntryItem struct {
	Path     string
	Name     string
	IsDir    bool
	IsParent bool
	Selected bool
}

func FilterEntries(entries []EntryItem, filter string) []EntryItem {
	if filter == "" {
		return entries
	}
	needle := strings.ToLower(filter)
	out := make([]EntryItem, 0)
	for _, e := range entries {
		if e.IsParent {
			out = append(out, e)
			continue
		}
		if strings.Contains(strings.ToLower(e.Name), needle) {
			out = append(out, e)
		}
	}
	return out
}

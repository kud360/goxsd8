package xmltree

import "strings"

// pseudoAttr returns the value of an XML declaration's name pseudo-attribute
// (encoding, standalone), or "" when the declaration omits it. inst is the
// declaration's content — everything between "<?xml" and "?>".
func pseudoAttr(inst, name string) string {
	rest := inst
	for {
		at := strings.Index(rest, name)
		if at < 0 {
			return ""
		}
		rest = rest[at+len(name):]
		value, ok := pseudoAttrValue(rest)
		if ok {
			return value
		}
	}
}

// pseudoAttrValue reads the quoted value of a pseudo-attribute whose name has
// just been consumed, reporting false when what follows is not "= 'value'".
func pseudoAttrValue(rest string) (string, bool) {
	rest = strings.TrimLeft(rest, " \t\r\n")
	if !strings.HasPrefix(rest, "=") {
		return "", false
	}
	rest = strings.TrimLeft(rest[1:], " \t\r\n")
	if rest == "" {
		return "", false
	}
	quote := rest[0]
	if quote != '\'' && quote != '"' {
		return "", false
	}
	end := strings.IndexByte(rest[1:], quote)
	if end < 0 {
		return "", false
	}
	return rest[1 : 1+end], true
}

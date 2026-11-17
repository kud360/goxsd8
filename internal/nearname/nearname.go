package nearname

// Name is an expanded name: Space is the namespace name, empty for the absent
// one, and Local the local name. The zero Name is an anonymous component.
type Name struct{ Space, Local string }

// Hint is the clause naming the candidate nearest missing, or "" when none
// qualifies (package doc).
func Hint(missing Name, candidates []Name, render func(Name) string) string {
	want := []rune(missing.Local)
	best, bestDist := -1, min(2, max(len(want), 3)/3)+1
	for i, c := range candidates {
		if c == (Name{}) || c.Space != missing.Space {
			continue
		}
		if d := distance(want, []rune(c.Local)); d < bestDist {
			best, bestDist = i, d
		}
	}
	if best < 0 {
		return ""
	}
	return "; did you mean " + render(candidates[best]) + "?"
}

// distance is the Levenshtein distance between a and b, every edit costing 1,
// over one row of the dynamic-programming table.
func distance(a, b []rune) int {
	row := make([]int, len(b)+1)
	for j := range row {
		row[j] = j
	}
	for i := 1; i <= len(a); i++ {
		diag := row[0]
		row[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			diag, row[j] = row[j], min(row[j]+1, row[j-1]+1, diag+cost)
		}
	}
	return row[len(b)]
}

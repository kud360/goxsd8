module github.com/kud360/goxsd8

go 1.26

tool (
	github.com/kud360/goxsd8/tools/fetchspecs
	github.com/kud360/goxsd8/tools/hfnextract
	github.com/kud360/goxsd8/tools/rulecat
	github.com/kud360/goxsd8/tools/spec2md
	github.com/kud360/goxsd8/tools/typespecgen
)

require golang.org/x/net v0.56.0

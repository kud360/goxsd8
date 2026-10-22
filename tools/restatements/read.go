package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kud360/goxsd8/tools/internal/commentpara"
)

// usageFile and usageConst name the one text read that is not a comment:
// cmd/goxsd8's usage, the CLI's contract. renderFile holds usage rendered as
// its package doc's "# Usage" section (cmd/goxsd8's TestDocRendersUsage).
const (
	usageFile  = "cmd/goxsd8/main.go"
	usageConst = "usage"
	renderFile = "cmd/goxsd8/doc.go"
)

// sentence is one sentence of a tree's prose: the slash-separated path of its
// file relative to the tree's root, the line it starts on, its words joined by
// single spaces, and whether it is an owner's (the package doc names them).
type sentence struct {
	file  string
	line  int
	text  string
	owner bool
}

// word is one whitespace-delimited word of prose and the source line it sits on.
type word struct {
	text string
	line int
}

// readTree returns the sentences of every Go file under root: the comments',
// file by file in path order and in source order within a file, then usage's.
// A renderFile comment sentence spelling a usage sentence word for word is
// usage rendered and is dropped, so that text is read once, at usage.
func readTree(root string) ([]sentence, error) {
	files, err := commentpara.Files([]string{filepath.Join(root, "...")})
	if err != nil {
		return nil, err
	}
	var comments, usage []sentence
	for _, path := range files {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil, fmt.Errorf("relativizing %s to %s: %w", path, root, err)
		}
		c, u, err := readFile(path, filepath.ToSlash(rel))
		if err != nil {
			return nil, err
		}
		comments = append(comments, c...)
		usage = append(usage, u...)
	}

	rendered := make(map[string]bool, len(usage))
	for _, s := range usage {
		rendered[s.text] = true
	}
	out := make([]sentence, 0, len(comments)+len(usage))
	for _, s := range comments {
		if s.file != renderFile || !rendered[s.text] {
			out = append(out, s)
		}
	}
	return append(out, usage...), nil
}

// readFile returns one file's comment sentences and, for usageFile, usage's,
// rel being the file's path as sentence.file spells it.
func readFile(path, rel string) (comments, usage []sentence, err error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", path, err)
	}
	lines := strings.Split(string(src), "\n")
	if commentpara.IsGenerated(lines) {
		return nil, nil, nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	if rel == usageFile {
		usage, err = usageSentences(fset, file)
		if err != nil {
			return nil, nil, fmt.Errorf("reading %s's %s: %w", rel, usageConst, err)
		}
	}

	owners := ownerDocs(file, filepath.Base(rel))
	for _, group := range file.Comments {
		for _, para := range commentpara.Paragraphs(fset, group, lines) {
			if para[0].IsVerbatim() {
				continue
			}
			var words []word
			for _, l := range para {
				words = appendWords(words, l.Body, l.Num)
			}
			comments = append(comments, sentences(words, rel, owners[group])...)
		}
	}
	return comments, usage, nil
}

// ownerDocs returns the comment groups of file that are owners' — every one in
// a doc.go, otherwise the package clause's doc and the doc comment of each
// exported identifier: a func, a method of an exported type, a type, a const
// or var, a struct field or interface method of an exported type, and a
// declaration group declaring any of those. A test file has none. The map is
// a lookup index (STYLE D2's permitted use).
func ownerDocs(file *ast.File, name string) map[*ast.CommentGroup]bool {
	docs := make(map[*ast.CommentGroup]bool)
	if strings.HasSuffix(name, "_test.go") {
		return docs
	}
	if name == "doc.go" {
		for _, g := range file.Comments {
			docs[g] = true
		}
		return docs
	}
	add := func(g *ast.CommentGroup) {
		if g != nil {
			docs[g] = true
		}
	}
	add(file.Doc)
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.IsExported() && (d.Recv == nil || len(d.Recv.List) == 0 || ast.IsExported(recvName(d.Recv.List[0].Type))) {
				add(d.Doc)
			}
		case *ast.GenDecl:
			if specDocs(d.Specs, add) {
				add(d.Doc)
			}
		}
	}
	return docs
}

// specDocs adds the doc comment of every spec declaring an exported name, and
// of the exported fields and methods of an exported struct or interface type,
// and reports whether any spec declared one.
func specDocs(specs []ast.Spec, add func(*ast.CommentGroup)) bool {
	exported := false
	for _, spec := range specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			if !s.Name.IsExported() {
				continue
			}
			exported = true
			add(s.Doc)
			memberDocs(s.Type, add)
		case *ast.ValueSpec:
			for _, n := range s.Names {
				if n.IsExported() {
					exported = true
					add(s.Doc)
					break
				}
			}
		}
	}
	return exported
}

// memberDocs adds the doc comment of every exported field of a struct type
// and every exported method of an interface type.
func memberDocs(typ ast.Expr, add func(*ast.CommentGroup)) {
	var fields *ast.FieldList
	switch t := typ.(type) {
	case *ast.StructType:
		fields = t.Fields
	case *ast.InterfaceType:
		fields = t.Methods
	}
	if fields == nil {
		return
	}
	for _, f := range fields.List {
		for _, n := range f.Names {
			if n.IsExported() {
				add(f.Doc)
				break
			}
		}
	}
}

// recvName is the name of a method receiver's base type, its pointer and type
// arguments stripped.
func recvName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return recvName(t.X)
	case *ast.IndexExpr:
		return recvName(t.X)
	case *ast.IndexListExpr:
		return recvName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}

// usageSentences returns the sentences of file's usage constant, owners all.
// Its prose sits at column 0 and, under a subcommand's synopsis line at column
// 2, at column 6; a synopsis line, and a line indented past 6 (an example),
// is read as no sentence and ends the one before it, as a blank line does.
// A file declaring no usage constant has none.
func usageSentences(fset *token.FileSet, file *ast.File) ([]sentence, error) {
	lit := usageLiteral(file)
	if lit == nil {
		return nil, nil
	}
	text, err := strconv.Unquote(lit.Value)
	if err != nil {
		return nil, fmt.Errorf("unquoting: %w", err)
	}
	first := fset.Position(lit.ValuePos).Line
	var out []sentence
	var words []word
	flush := func() {
		out = append(out, sentences(words, usageFile, true)...)
		words = nil
	}
	for i, line := range strings.Split(text, "\n") {
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if strings.TrimSpace(line) == "" || indent == 2 || indent > 6 {
			flush()
			continue
		}
		// A raw string's lines are its source lines; an interpreted one's
		// escaped newlines are not, and all of it is located at its first.
		num := first
		if strings.HasPrefix(lit.Value, "`") {
			num += i
		}
		words = appendWords(words, line, num)
	}
	flush()
	return out, nil
}

// usageLiteral finds the string literal file's usage constant is declared
// with, or nil.
func usageLiteral(file *ast.File) *ast.BasicLit {
	for _, decl := range file.Decls {
		d, ok := decl.(*ast.GenDecl)
		if !ok || d.Tok != token.CONST {
			continue
		}
		for _, spec := range d.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, n := range vs.Names {
				if n.Name != usageConst || i >= len(vs.Values) {
					continue
				}
				if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					return lit
				}
			}
		}
	}
	return nil
}

// appendWords appends the words of one line of prose, all on line num.
func appendWords(words []word, line string, num int) []word {
	for _, w := range strings.Fields(line) {
		words = append(words, word{text: w, line: num})
	}
	return words
}

// abbreviations end in a full stop and end no sentence.
var abbreviations = map[string]bool{"e.g.": true, "i.e.": true, "cf.": true, "etc.": true, "vs.": true}

// sentences cuts a paragraph's words into sentences: one ends at a word whose
// last character, closing quotes, brackets and parentheses aside, is '.', '?'
// or '!', unless the word is an abbreviation or nothing but full stops (an
// ellipsis). A paragraph's last words are a sentence whatever they end with.
func sentences(words []word, file string, owner bool) []sentence {
	var out []sentence
	start := 0
	for i, w := range words {
		if i < len(words)-1 && !endsSentence(w.text) {
			continue
		}
		texts := make([]string, 0, i+1-start)
		for _, x := range words[start : i+1] {
			texts = append(texts, x.text)
		}
		out = append(out, sentence{file: file, line: words[start].line, text: strings.Join(texts, " "), owner: owner})
		start = i + 1
	}
	return out
}

// endsSentence reports whether a word closes a sentence, as sentences states.
func endsSentence(w string) bool {
	core := strings.TrimRight(w, `)]"'”’`+"`")
	if core == "" || strings.Trim(core, ".") == "" {
		return false
	}
	if abbreviations[strings.ToLower(strings.TrimLeft(core, `(["'“‘`))] {
		return false
	}
	last, _ := utf8.DecodeLastRuneInString(core)
	return last == '.' || last == '?' || last == '!'
}

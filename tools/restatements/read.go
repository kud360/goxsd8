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

// usageFile and usageConst name the one text read that is not a comment, and
// renderFile the file holding it rendered.
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

// ident is an exported identifier of a package of a tree: the slash-separated
// directory of its package relative to the tree's root, "" for the root, and
// its name, a member's qualified by its type's ("Result.Err").
type ident struct {
	dir  string
	name string
}

// readTree returns the sentences of every Go file under root: the comments',
// file by file in path order and in source order within a file, less usage's
// rendering as the package doc states, then usage's. It returns too the
// sentences of each exported identifier's doc outside package main, in
// source order: a lookup index (STYLE D2's permitted use of a map).
func readTree(root string) ([]sentence, map[ident][]sentence, error) {
	files, err := commentpara.Files([]string{filepath.Join(root, "...")})
	if err != nil {
		return nil, nil, err
	}
	var comments, usage []sentence
	idents := make(map[ident][]sentence)
	for _, path := range files {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil, nil, fmt.Errorf("relativizing %s to %s: %w", path, root, err)
		}
		rel = filepath.ToSlash(rel)
		c, u, docs, err := readFile(path, rel)
		if err != nil {
			return nil, nil, err
		}
		comments = append(comments, c...)
		usage = append(usage, u...)
		for _, d := range docs {
			k := ident{dir: dirOf(rel), name: d.name}
			idents[k] = append(idents[k], d.sentences...)
		}
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
	return append(out, usage...), idents, nil
}

// dirOf is the slash-separated directory of a file named by its
// slash-separated path relative to a tree's root, "" for the root.
func dirOf(rel string) string {
	i := strings.LastIndex(rel, "/")
	if i < 0 {
		return ""
	}
	return rel[:i]
}

// identDoc is the sentences of one exported identifier's doc, the identifier
// named as ident.name spells it.
type identDoc struct {
	name      string
	sentences []sentence
}

// readFile returns one file's comment sentences, for usageFile usage's, and,
// in a package other than main, the doc of each exported identifier it
// declares, in source order, rel being the file's path as sentence.file
// spells it.
func readFile(path, rel string) (comments, usage []sentence, docs []identDoc, err error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("reading %s: %w", path, err)
	}
	lines := strings.Split(string(src), "\n")
	if commentpara.IsGenerated(lines) {
		return nil, nil, nil, nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	if rel == usageFile {
		usage, err = usageSentences(fset, file)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("reading %s's %s: %w", rel, usageConst, err)
		}
	}

	owners := ownerDocs(file, filepath.Base(rel))
	byGroup := make(map[*ast.CommentGroup][]sentence)
	for _, group := range file.Comments {
		_, owner := owners[group]
		for _, para := range commentpara.Paragraphs(fset, group, lines) {
			if para[0].IsVerbatim() {
				continue
			}
			var words []word
			for _, l := range para {
				words = appendWords(words, l.Body, l.Num)
			}
			ss := sentences(words, rel, owner)
			comments = append(comments, ss...)
			byGroup[group] = append(byGroup[group], ss...)
		}
	}
	if file.Name.Name == "main" {
		return comments, usage, nil, nil
	}
	for _, group := range file.Comments {
		for _, name := range owners[group] {
			docs = append(docs, identDoc{name: name, sentences: byGroup[group]})
		}
	}
	return comments, usage, docs, nil
}

// ownerDocs returns the comment groups of file that are owners', the package
// doc naming them, file being named name, each with the exported identifiers
// it documents as ident.name spells them, none for the package clause's doc
// or a doc.go comment documenting no declaration. An exported identifier is a
// func, a method of an exported type, a type, a const or var, or a struct
// field or interface method of an exported type, and a declaration group
// declaring any of those has its doc read too, documenting each of those
// names whose spec has no doc of its own. A test file has none. The map is a
// lookup index (STYLE D2's permitted use).
func ownerDocs(file *ast.File, name string) map[*ast.CommentGroup][]string {
	docs := make(map[*ast.CommentGroup][]string)
	if strings.HasSuffix(name, "_test.go") {
		return docs
	}
	if name == "doc.go" {
		for _, g := range file.Comments {
			docs[g] = nil
		}
	}
	add := func(g *ast.CommentGroup, names ...string) {
		if g != nil {
			docs[g] = append(docs[g], names...)
		}
	}
	add(file.Doc)
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if !d.Name.IsExported() {
				continue
			}
			if d.Recv == nil || len(d.Recv.List) == 0 {
				add(d.Doc, d.Name.Name)
				continue
			}
			if recv := recvName(d.Recv.List[0].Type); ast.IsExported(recv) {
				add(d.Doc, recv+"."+d.Name.Name)
			}
		case *ast.GenDecl:
			if exported, undocumented := specDocs(d.Specs, add); exported {
				add(d.Doc, undocumented...)
			}
		}
	}
	return docs
}

// specDocs adds the doc comment of every spec declaring an exported name, and
// of the exported fields and methods of an exported struct or interface type,
// each with the names it documents. It reports whether any spec declared an
// exported name, and returns the exported names whose spec has no doc.
func specDocs(specs []ast.Spec, add func(*ast.CommentGroup, ...string)) (exported bool, undocumented []string) {
	for _, spec := range specs {
		var names []string
		var doc *ast.CommentGroup
		switch s := spec.(type) {
		case *ast.TypeSpec:
			if !s.Name.IsExported() {
				continue
			}
			names, doc = []string{s.Name.Name}, s.Doc
			memberDocs(s.Type, s.Name.Name, add)
		case *ast.ValueSpec:
			for _, n := range s.Names {
				if n.IsExported() {
					names = append(names, n.Name)
				}
			}
			doc = s.Doc
		}
		if len(names) == 0 {
			continue
		}
		exported = true
		add(doc, names...)
		if doc == nil {
			undocumented = append(undocumented, names...)
		}
	}
	return exported, undocumented
}

// memberDocs adds the doc comment of every exported field of a struct type
// and every exported method of an interface type, the type being named typ,
// each with the member names it documents qualified by typ.
func memberDocs(expr ast.Expr, typ string, add func(*ast.CommentGroup, ...string)) {
	var fields *ast.FieldList
	switch t := expr.(type) {
	case *ast.StructType:
		fields = t.Fields
	case *ast.InterfaceType:
		fields = t.Methods
	}
	if fields == nil {
		return
	}
	for _, f := range fields.List {
		var names []string
		for _, n := range f.Names {
			if n.IsExported() {
				names = append(names, typ+"."+n.Name)
			}
		}
		if len(names) > 0 {
			add(f.Doc, names...)
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

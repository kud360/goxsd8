package xsd

// AppInfo is one <appinfo> child of an Annotation (Structures §3.15.1/
// §3.15.2): an optional {source} (an anyURI) and the element's opaque raw
// content, preserved verbatim and never interpreted — §3.15.1: "The mapping
// defined in this specification from XML representations to components does
// not apply to XML elements contained within an <annotation> element; such
// elements do not correspond to components." Content is the serialized child
// subtree ({any}* per §3.15.2), an opaque string this package only carries;
// the parser owns however it serializes or extracts it.
//
// Construct only through NewAppInfo. AppInfo is immutable after construction.
type AppInfo struct {
	source    string
	hasSource bool
	content   string
}

// NewAppInfo builds an AppInfo. A nil source means the {source} attribute is
// absent; a non-nil source (including a pointer to "") means it is present,
// because "" is a legal anyURI and cannot double as an absence sentinel.
func NewAppInfo(source *string, content string) AppInfo {
	a := AppInfo{content: content}
	if source != nil {
		a.source, a.hasSource = *source, true
	}
	return a
}

// Source returns the {source} attribute (an anyURI); the second result is
// false when it is absent, in which case the first result is not meaningful.
func (a AppInfo) Source() (string, bool) {
	return a.source, a.hasSource
}

// Content returns the opaque raw content of the <appinfo>, preserved verbatim.
func (a AppInfo) Content() string {
	return a.content
}

// Documentation is one <documentation> child of an Annotation (Structures
// §3.15.1/§3.15.2): an optional {source} (an anyURI), an optional xml:lang
// (a language), and the element's opaque raw content, preserved verbatim
// under the same opacity discipline as AppInfo. Content is the serialized
// child subtree ({any}* per §3.15.2), an opaque string this package only
// carries.
//
// Construct only through NewDocumentation. Documentation is immutable after
// construction. It is a distinct type from AppInfo — <appinfo> structurally
// cannot carry xml:lang (§3.15.2), so unifying the two would make an illegal
// state representable.
type Documentation struct {
	source    string
	hasSource bool
	lang      string
	hasLang   bool
	content   string
}

// NewDocumentation builds a Documentation. A nil source or lang means the
// corresponding attribute is absent; a non-nil pointer (including to "") means
// it is present, because "" is a legal attribute value and cannot double as an
// absence sentinel.
func NewDocumentation(source, lang *string, content string) Documentation {
	d := Documentation{content: content}
	if source != nil {
		d.source, d.hasSource = *source, true
	}
	if lang != nil {
		d.lang, d.hasLang = *lang, true
	}
	return d
}

// Source returns the {source} attribute (an anyURI); the second result is
// false when it is absent, in which case the first result is not meaningful.
func (d Documentation) Source() (string, bool) {
	return d.source, d.hasSource
}

// Lang returns the xml:lang attribute (a language); the second result is false
// when it is absent, in which case the first result is not meaningful.
func (d Documentation) Lang() (string, bool) {
	return d.lang, d.hasLang
}

// Content returns the opaque raw content of the <documentation>, preserved
// verbatim.
func (d Documentation) Content() string {
	return d.content
}

// Attr is one entry of an Annotation's flattened {attributes} set (Structures
// §3.15.1): a name (as an expanded QName) and its string value. Provenance
// among possibly several enclosing elements is intentionally not preserved —
// the spec's own wording: the attributes "do not indicate which element
// information item ... was their parent."
//
// Construct only through NewAttr. Attr is immutable after construction.
type Attr struct {
	name  QName
	value string
}

// NewAttr builds an Attr with the given expanded name and value.
func NewAttr(name QName, value string) Attr {
	return Attr{name: name, value: value}
}

// Name returns the attribute's expanded name.
func (a Attr) Name() QName {
	return a.name
}

// Value returns the attribute's string value.
func (a Attr) Value() string {
	return a.value
}

// Annotation is the Annotation component (Structures §3.15.1): a bare
// Component — not an Annotated Component, so it carries no {annotations} of
// its own (it is the annotation). Its properties are {application information}
// (the <appinfo> items), {user information} (the <documentation> items), and
// {attributes} (the flattened attribute set). §3.15.3/§3.15.4 define no
// constraint (both "None as such"; §3.15.6's Annotation Correct,
// an-props-correct, is the generic tableau SCC), so construction has no
// rejectable state and NewAnnotation returns no error.
//
// All three collections are held in document order (STYLE D2). {attributes}
// is, per §3.15.1, a flattened unordered set; it is stored in document order
// for determinism only, and that order carries no spec significance.
//
// Construct only through NewAnnotation. Annotation is immutable after
// construction.
type Annotation struct {
	appInfo       []AppInfo
	documentation []Documentation
	attributes    []Attr
}

// NewAnnotation builds an Annotation. Each incoming slice is copied; the
// caller's backing arrays are not aliased. Empty or nil inputs are held as nil.
func NewAnnotation(appInfo []AppInfo, documentation []Documentation, attributes []Attr) Annotation {
	a := Annotation{}
	if len(appInfo) > 0 {
		a.appInfo = append([]AppInfo(nil), appInfo...)
	}
	if len(documentation) > 0 {
		a.documentation = append([]Documentation(nil), documentation...)
	}
	if len(attributes) > 0 {
		a.attributes = append([]Attr(nil), attributes...)
	}
	return a
}

// AppInfo returns the {application information} property (the <appinfo> items)
// in document order. It returns a copy: mutating the result does not affect a.
// An empty {application information} yields nil.
func (a Annotation) AppInfo() []AppInfo {
	if len(a.appInfo) == 0 {
		return nil
	}
	return append([]AppInfo(nil), a.appInfo...)
}

// Documentation returns the {user information} property (the <documentation>
// items) in document order, each carrying its own {source} and xml:lang and
// its readable Content. It returns a copy: mutating the result does not affect
// a. An empty {user information} yields nil.
func (a Annotation) Documentation() []Documentation {
	if len(a.documentation) == 0 {
		return nil
	}
	return append([]Documentation(nil), a.documentation...)
}

// Attributes returns the {attributes} property in document order. It returns a
// copy: mutating the result does not affect a. An empty {attributes} yields
// nil.
func (a Annotation) Attributes() []Attr {
	if len(a.attributes) == 0 {
		return nil
	}
	return append([]Attr(nil), a.attributes...)
}

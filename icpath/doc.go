// Package icpath owns the RESTRICTED path subset an identity constraint's
// {selector} and {fields} are written in — the ·selector subset· of §3.11.6.2
// (c-selector-xpath) and the ·field subset· of §3.11.6.3 (c-fields-xpaths) —
// and nothing wider. It is deliberately NOT a bridge to the XPath 2.0
// evaluator: the productions below are a path grammar over the child and
// attribute axes with no predicates, no functions and no other axis, so
// evaluating them directly is exact where a fail-open delegation to a general
// engine would be a guess.
//
//	[1] Selector   ::= Path ( '|' Path )*
//	[2] Path       ::= ('.' '//')? Step ( '/' Step )*
//	[3] Step       ::= '.' | NameTest
//	[4] NameTest   ::= QName | '*' | NCName ':*'
//	[5] token      ::= '.' | '/' | '//' | '|' | '@' | NameTest
//	[6] whitespace ::= S
//	[7] Path       ::= ('.' '//')? ( Step '/' )* ( Step | '@' NameTest )   (fields only)
//
// Production [7] is the whole of the fields/selector difference: only a field's
// FINAL step may name an attribute. [5] and [6] are §3.11.6.2's lexical
// productions, which §3.11.6.3 repeats verbatim for fields — "whitespace may be
// freely added within patterns before or after any token", and "when tokenizing,
// the longest possible token is always returned".
//
// # Why this is not part of xpath
//
// The grammar above is not a stage of XPath 2.0 and shares no production with
// it, so hosting it in xpath would put one package's name on two unrelated
// grammars. xpath serves conditional type assignment and assertions;
// validate/doc.go states that identity-constraint paths are evaluated "directly
// and never through the XPath engine", and that stays true with this package
// carrying them.
//
// It is its own package rather than validate's private file because the two
// Schema Component Constraints over this grammar are charged at schema ASSEMBLY
// (parser) while the paths themselves are evaluated at instance time
// (validate). One grammar with two consumers in different layers is the shape
// xpath already has for §3.12.6's ta-Test grammar (STYLE T4).
//
// # The matcher
//
// [CompileSelector] and [CompileField] turn one [xsd.XPathExpression] property
// record into an [Expr], and [Expr.Start], [Expr.Self] and [Live.Advance] carry
// one down a descent.
//
// Evaluation is incremental, because the walk that drives it is streaming: an
// [Expr] never sees a tree. [Expr.Start] opens an expression at its context node,
// [Live.Advance] carries the live step indices one level down as the descent
// enters a child, and a path whose last step matches reports the selection at
// that child. The leading `.//` is the only unbounded construct, and it is
// handled by re-seeding step 0 at every level rather than by remembering a depth.
//
// The compiled path/step tree is not exported and never will be: the compiler
// establishes invariants the matcher depends on — `.` self steps already
// stripped, prefixes already resolved against the record's {namespace bindings} —
// and a consumer holding the tree could violate each of them. [Selection]
// answers about what one level selects; it hands back no NameTest.
//
// # The two Schema Component Constraints
//
// [SelectorViolation] and [FieldViolation] are the assembler's entry points, and
// they charge FOUR shapes: an unbound prefix, a predicate, an attribute step
// before a field's final step, and an attribute named anywhere in a selector.
// Every other {expression} is nil there — above all one this package simply
// cannot read — because clause 2 of each SCC is a disjunction whose second arm
// ("an XPath expression involving the child axis whose abbreviated form is as
// given above") no recognizer of clause 2.1's BNF can see. "Does not match
// production [1]" is therefore never evidence of a violation, and
// [SelectorViolation]'s doc carries the argument for each of the four.
//
// THE PREDICATES IT RECOGNIZES are the ones whose whole {expression} lexes as
// tokens of production [5] plus the two brackets: `a[b]` and `a[@b]` are
// charged, `a[1]` and `a[b='c']` are not, because a digit and a quote open no
// token and a stream this package cannot read whole is declined whatever it
// holds. Under-charging is a rejection the processor can still make at validate
// time; over-charging rejects a conforming schema before any instance exists.
//
// One sub-shape inside those four is declined rather than charged, under
// shapeFault's own GAP marker: an '@' with no NameTest after it, in a FIELD.
//
// An unbound prefix is the one charged shape with a vocabulary of its own:
// err:XPST0081 travels as the wrapped cause under the SCC charge, reached with
// one errors.Unwrap and read with [xsderr.RuleOf]. The other three have no
// second vocabulary and wrap nothing, so the SCC's rule is never on two layers
// of one error (STYLE E2).
package icpath

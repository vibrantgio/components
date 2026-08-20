// The reading sample: the prose surface, rendered by the real markdown
// renderer rather than mocked up out of labels.
//
// A theme is judged on running text as much as on controls — heading ladder,
// link colour, code chip against the body, table rules, checkbox marks — and
// none of that shows on a page of widgets. The sample below is chosen to put
// every one of those on screen at once.
//
// Code gets a sample of its own, for the same reason and one more: a syntax
// palette is a dozen decisions — keyword against string against number
// against comment — and a fence with two statements in it shows one of them.
// The specimen below shows all of them at once, so the question "does this
// palette work" has an answer that can be read off one screen rather than
// assembled from several.
package inventory

import (
	"gioui.org/layout"

	"github.com/vibrantgio/markdown"
	"github.com/vibrantgio/markdown/highlight"
	"github.com/vibrantgio/theme/tokens"
)

// readingSample exercises, in order: the heading ladder, a paragraph carrying
// links and inline code chips, a bullet list, an ordered list, a task list
// with both states, a table, a blockquote, a rule, and a fenced code block
// whose longest line overflows so the horizontal scroll shows.
const readingSample = "" +
	"# Reading sample\n" +
	"\n" +
	"## What a document surface has to carry\n" +
	"\n" +
	"A theme is not finished until running prose reads on it. This paragraph\n" +
	"carries a [link to gioui.org](https://gioui.org), a second one to the\n" +
	"[design system](https://github.com/vibrantgio), an inline `code chip`,\n" +
	"and some **bold** and *italic* and ~~struck~~ runs to set the weight\n" +
	"against the body.\n" +
	"\n" +
	"### Heading three, where the ladder starts to flatten\n" +
	"\n" +
	"- A bullet list item\n" +
	"- A second one, with `code` inside it\n" +
	"- A third\n" +
	"\n" +
	"1. An ordered item\n" +
	"2. The one after it\n" +
	"\n" +
	"#### Heading four\n" +
	"\n" +
	"- [x] A task that is done\n" +
	"- [ ] A task that is not\n" +
	"- [x] Another finished one\n" +
	"\n" +
	"| Family   | Kind      | Sections |\n" +
	"| -------- | --------- | -------: |\n" +
	"| Button   | Component |        1 |\n" +
	"| Card     | Pattern   |        1 |\n" +
	"| Markdown | Document  |        1 |\n" +
	"\n" +
	"> A blockquote stands off the page on its own bar, and is where a\n" +
	"> surface's quiet text colour gets tested.\n" +
	"\n" +
	"---\n" +
	"\n" +
	"```go\n" +
	"// The fenced block's longest line overflows on purpose, so the code area's horizontal scroll and its edge dissolve both show.\n" +
	"doc := markdown.NewDocument(markdown.Parse(source))\n" +
	"doc.LayoutColumn(gtx, shaper, markdown.FromTokens(colors, typography))\n" +
	"```\n"

// codeSample is the syntax specimen: one excerpt carrying every kind of run a
// palette colours differently, so the whole set of decisions is legible at a
// glance rather than a fence at a time.
//
// Line comments and a doc comment; the keywords a body is built from and the
// declaring ones above them; a package-level constant beside a struct's
// fields; strings both plain and with verbs in them; integers, a float and a
// duration; the built-in types, a named one, and a pointer to it; and calls
// through three different packages, so a function name is on screen next to
// the type it returns.
//
// It is a real excerpt rather than a list of token types on purpose. What a
// syntax palette has to survive is code as it is actually shaped — a run of
// keywords at the head of a line, a string wedged between two calls — and a
// specimen arranged for the palette's convenience would flatter it.
//
// The comments are kept short and few, which is the one way this excerpt is
// unlike the code it stands for. A comment is the widest run on any line it
// is on, and a specimen written with the doc comments real code deserves puts
// the comment colour over most of the plate — leaving the reader judging a
// palette by the one ink it spends least effort on, with the keywords and
// strings the choice actually turns on crowded into the margins.
const codeSample = "" +
	"```go\n" +
	"// Package berth assigns vessels to the quay they fit on.\n" +
	"package berth\n" +
	"\n" +
	"import (\n" +
	"\t\"fmt\"\n" +
	"\t\"time\"\n" +
	")\n" +
	"\n" +
	"const clearance = 1.05 // room beyond a vessel's own length\n" +
	"\n" +
	"// A Berth is one place along the quay.\n" +
	"type Berth struct {\n" +
	"\tName   string\n" +
	"\tLength float64\n" +
	"\tFree   bool\n" +
	"\tUntil  time.Time\n" +
	"}\n" +
	"\n" +
	"// Assign puts v in the first berth long enough to hold it.\n" +
	"func (q *Quay) Assign(v Vessel) (*Berth, error) {\n" +
	"\tfor i := range q.berths {\n" +
	"\t\tb := &q.berths[i]\n" +
	"\t\tif !b.Free || b.Length < v.Length*clearance {\n" +
	"\t\t\tcontinue\n" +
	"\t\t}\n" +
	"\t\tb.Free, b.Until = false, time.Now().Add(72*time.Hour)\n" +
	"\t\tq.log.Printf(\"berth %s taken (%.1f m)\", b.Name, v.Length)\n" +
	"\t\treturn b, nil\n" +
	"\t}\n" +
	"\treturn nil, fmt.Errorf(\"no berth for %q at %.1f m\", v.Name, v.Length)\n" +
	"}\n" +
	"```\n"

// Reading returns the prose sections: documents laid out by the real markdown
// renderer rather than mocked up out of labels.
//
// The code specimen comes last, and last on the whole column with it. This
// surface is for judging a theme, and a plate of syntax at the top of it takes
// the attention the palette, the components and the compositions are there to
// get — a page that opens on code is a page about code. Put at the end it is
// still the whole specimen, reached by whoever went looking for it; and a
// caller with a reason to put it in front of somebody has [ItemIndex] to scroll
// there, which costs one call and no reordering.
func (inv *Inventory) Reading(c tokens.ColorTokens) []Section {
	return []Section{{
		Name:   "markdown-reading",
		Title:  "Markdown — headings, links, chips, lists, tasks, a table, a quote and a code fence",
		Height: 966,
		Body:   inv.readingBody(c),
	}, {
		Name:   codeSectionName,
		Title:  "Markdown — a fenced code block, in the chosen syntax palette as its author drew it",
		Height: 613,
		Body:   inv.codeBody(c),
	}}
}

// codeSectionName identifies the syntax specimen's section. It is a constant
// because it is also the answer to "which row is the code on" — see
// [Inventory.ItemIndex].
const codeSectionName = "markdown-code"

// CodeSectionName is the name of the section the syntax specimen is drawn in,
// for a caller that wants to put it in front of somebody rather than wait for
// them to scroll to it.
func CodeSectionName() string { return codeSectionName }

func (inv *Inventory) codeBody(c tokens.ColorTokens) layout.Widget {
	style := markdown.FromTokens(c, tokens.DefaultTypography)
	inv.wear(&style, c)
	// No measure cap, unlike the prose sample. Prose is capped because a long
	// line of it is hard to read; code is not prose — a line wrapped or
	// scrolled out of sight is a line the palette cannot be judged on — and a
	// capped plate leaves its own section's heading running past it on one
	// side, which reads as a panel that failed to fill rather than as a
	// measure somebody chose.
	return func(gtx layout.Context) layout.Dimensions {
		return inv.code.LayoutColumn(gtx, inv.shaper, style)
	}
}

// SetCodeBase names one syntax palette the code in these sections is drawn
// in, under both appearances — one of the highlighting package's base names.
// The empty string, which is the value an [Inventory] starts with, means that
// package's own default; so does a name it does not recognise.
//
// It is a setting on the inventory rather than an argument to a section
// because it is not a fact about a palette: which member of a base's pair
// reaches the fence is the appearance's to say, and the tokens are what a
// section is a function of.
func (inv *Inventory) SetCodeBase(name string) {
	inv.SetCodeBases(highlight.BasePair{Light: name, Dark: name})
}

// SetCodeBases names a syntax palette per appearance, for a caller whose
// person has chosen one for each: the light member draws the code while the
// tokens are the light ones and the dark member while they are the dark ones,
// so a change of appearance is a change of palette rather than a counterpart
// somebody never picked. An unset member means the highlighting package's own
// default for that appearance, as does one it does not recognise.
func (inv *Inventory) SetCodeBases(p highlight.BasePair) { inv.codeBases = p }

// wear puts the chosen palette on st's fenced code: the appearance c
// describes picks the member, and that member's own ground, inks and body
// colour are what a block is drawn with. A name that no longer resolves falls
// back to the default for its appearance rather than failing the page.
func (inv *Inventory) wear(st *markdown.Style, c tokens.ColorTokens) {
	highlight.WearPair(st, highlight.BasePair{
		Light: highlight.BaseOrDefault(inv.codeBases.Light),
		Dark:  highlight.BaseOrDefault(inv.codeBases.Dark),
	}, c)
}

func (inv *Inventory) readingBody(c tokens.ColorTokens) layout.Widget {
	style := markdown.FromTokens(c, tokens.DefaultTypography)
	// The chosen base, worn whole: the fence takes the ground its author drew
	// their inks on, and the inks as they were drawn. Everything around it —
	// the page, the prose, the chip an inline span sits on — stays the theme's,
	// which is what makes the two judgeable side by side on one screen.
	inv.wear(&style, c)
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(560))
		// LayoutColumn rather than Layout: the document is embedded in a
		// column that scrolls already, and its own viewport would fight the
		// outer one.
		return inv.doc.LayoutColumn(gtx, inv.shaper, style)
	}
}

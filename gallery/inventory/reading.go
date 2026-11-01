// The reading sample: the prose surface, rendered by the real markdown
// renderer rather than mocked up out of labels.
//
// A theme is judged on running text as much as on controls — the heading
// steps, link colour, code chip against the body, table rules, checkbox marks
// — and none of that shows on a page of components. The sample below is chosen
// to put every one of those on screen at once.
//
// Its fenced block is the syntax specimen. A syntax highlighter style is a
// dozen decisions — keyword against string against number against comment —
// and the excerpt in the fence shows all of them on one screen.
package inventory

import (
	"gioui.org/layout"

	"github.com/vibrantgio/components/scrollbar"
	"github.com/vibrantgio/markdown"
	"github.com/vibrantgio/markdown/highlight"
	"github.com/vibrantgio/theme/tokens"
)

// readingSample exercises, in order: the heading steps, a paragraph carrying
// links and inline code chips, a bullet list, an ordered list, a task list
// with both states, a table, a blockquote, a rule, and a fenced code block.
//
// The fence is the last block, and the block above it is not a heading:
// [readingBlocks] depends on both.
//
// The fence holds one excerpt with every kind of run a style colours
// differently. Line comments and a doc comment; the keywords a body is built
// from and the declaring ones above them; a package-level constant beside a
// struct's fields; strings both plain and with verbs in them; integers, a
// float and a duration; the built-in types, a named one, and a pointer to it;
// and calls through three different packages.
//
// It is a real excerpt rather than a list of token types. A highlighter style
// has to survive code as it is shaped: a run of keywords at the head of a
// line, a string between two calls.
//
// The comments are short and few. A comment is the widest run on its line,
// and long comments put the comment colour over most of the block. The
// keywords and strings the choice of style turns on then get the margins.
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
	"### Heading three, where the steps start to flatten\n" +
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
	"> surface's muted text colour gets tested.\n" +
	"\n" +
	"---\n" +
	"\n" +
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

// readingBlocks parses [readingSample] and returns its prose blocks and its
// fence, the last block, apart.
//
// The two parts are laid out as two documents because one document has one
// width: the prose is capped at a reading measure and the fence is not. The
// space between them is the block gap, which is the space one document puts
// between a block that is not a heading and the block under it.
func readingBlocks() (prose, fence []markdown.Block) {
	blocks := markdown.Parse([]byte(readingSample))
	n := len(blocks) - 1
	return blocks[:n:n], blocks[n:]
}

// Reading returns the prose section: one document laid out by the real
// markdown renderer rather than mocked up out of labels.
//
// It is the last section on the column, and its fence is the last block in
// it. A plate of syntax at the top of the column takes the attention the
// colours, the components and the compositions are there to get. A caller
// that has to show the code first scrolls there with [Inventory.ItemIndex]
// and [CodeSectionName].
func (inv *Inventory) Reading(c tokens.PlatformColors) []Section {
	return []Section{{
		Name:   readingSectionName,
		Title:  "Markdown — headings, links, chips, lists, tasks, a table, a quote and a code fence in the chosen syntax highlighter style",
		Height: 1235,
		Body:   inv.readingBody(c),
	}}
}

// readingSectionName identifies the reading sample's section. It is a
// constant because it is also the answer to "which row is the code on": see
// [Inventory.ItemIndex].
const readingSectionName = "markdown-reading"

// CodeSectionName is the name of the section the syntax specimen is drawn in,
// for a caller that scrolls the specimen into view.
func CodeSectionName() string { return readingSectionName }

// SetCodeStyle names one syntax highlighter style the reading sample's fence
// is drawn in, under both appearances — one of the highlighting package's own
// style names. The empty string, which is the value an [Inventory] starts
// with, means that package's own default; so does a name it does not
// recognise.
//
// It is a setting on the inventory rather than an argument to a section
// because it is not a fact about a style: which member of a style's pair
// reaches the fence is the appearance's to say, and the tokens are what a
// section is a function of.
func (inv *Inventory) SetCodeStyle(name string) {
	inv.SetCodeStyles(highlight.StylePair{Light: name, Dark: name})
}

// SetCodeStyles names a highlighter style per appearance, for a caller whose
// person has chosen one for each: the light member draws the code while the
// tokens are the light ones and the dark member while they are the dark ones,
// so a change of appearance is a change of style rather than a counterpart
// somebody never picked. An unset member means the highlighting package's own
// default for that appearance, as does one it does not recognise.
func (inv *Inventory) SetCodeStyles(p highlight.StylePair) { inv.codeStyles = p }

// applyStyle applies the chosen style to st's fenced code: the appearance c is a
// recording of picks the member, and that member's own fill, its syntax colours and its
// body colour are what a block is drawn with. A name that no longer resolves
// falls back to the default for its appearance rather than failing the page.
func (inv *Inventory) applyStyle(st *markdown.Style, c tokens.PlatformColors) {
	highlight.ApplyPair(st, highlight.StylePair{
		Light: highlight.StyleOrDefault(inv.codeStyles.Light),
		Dark:  highlight.StyleOrDefault(inv.codeStyles.Dark),
	}, c)
}

// markdownStyle is the style both parts of the reading sample draw with. The chosen
// highlighter style is used whole: the fence takes the fill its author drew
// their colours on, and those colours as they were drawn. Everything around
// it — the page, the prose, the chip an inline span sits on — stays the
// theme's, which is what makes the two judgeable side by side on one screen.
// The fence's bar and the table's bar follow the window's bar.
func (inv *Inventory) markdownStyle(c tokens.PlatformColors) markdown.Style {
	style := markdown.FromTokens(c, inv.typography(), SectionSurface(c))
	inv.applyStyle(&style, c)
	if inv.windowBar != nil {
		style.CodeScrollbar = followBar(style.CodeScrollbar, *inv.windowBar)
		style.TableScrollbar = followBar(style.TableScrollbar, *inv.windowBar)
	}
	return style
}

// SetScrollbar names the bar of the window that shows the inventory. The
// window builds that bar from the platform's Show scroll bars setting, which
// it reads once where it is composed. The bar inside an overflowing fence and
// the bar under a wide table then fade or stand as the window's bar does.
// Without a call both bars fade, which is the bar markdown's style draws.
//
// It is a setting on the inventory, as the highlighter style is, because the
// inventory is in a module with no composition root of its own: the setting
// belongs to whichever window shows the inventory, and a section is a
// function of the tokens alone.
func (inv *Inventory) SetScrollbar(bar scrollbar.Style) { inv.windowBar = &bar }

// followBar returns own standing when the window's bar stands, and with the
// window's fading otherwise. The colours stay own's: each bar is built from
// the fill it is drawn on, and the window's bar is built from the window's
// column.
func followBar(own, window scrollbar.Style) scrollbar.Style {
	if window.FadeDelay == 0 {
		return own.Always()
	}
	own.FadeDelay = window.FadeDelay
	own.FadeDuration = window.FadeDuration
	return own
}

func (inv *Inventory) readingBody(c tokens.PlatformColors) layout.Widget {
	style := inv.markdownStyle(c)
	return func(gtx layout.Context) layout.Dimensions {
		// LayoutColumn rather than Layout: the document is embedded in a
		// column that scrolls already, and its own viewport would fight the
		// outer one.
		//
		// The prose is capped at a reading measure because a long line of it
		// is hard to read. The fence has no cap: a code line wrapped or
		// scrolled out of sight is a line the style cannot be judged on.
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(560))
				return inv.doc.LayoutColumn(gtx, inv.shaper, style)
			}),
			layout.Rigid(layout.Spacer{Height: style.BlockGap}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return inv.code.LayoutColumn(gtx, inv.shaper, style)
			}),
		)
	}
}

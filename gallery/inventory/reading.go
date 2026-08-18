// The reading sample: the prose surface, rendered by the real markdown
// renderer rather than mocked up out of labels.
//
// A theme is judged on running text as much as on controls — heading ladder,
// link colour, code chip against the body, table rules, checkbox marks — and
// none of that shows on a page of widgets. The sample below is chosen to put
// every one of those on screen at once.
package inventory

import (
	"gioui.org/layout"

	"github.com/vibrantgio/markdown"
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

// Reading returns the prose section: a document laid out by the real
// markdown renderer rather than mocked up out of labels.
func (inv *Inventory) Reading(c tokens.ColorTokens) []Section {
	return []Section{{
		Name:   "markdown-reading",
		Title:  "Markdown — headings, links, chips, lists, tasks, a table, a quote and a code fence",
		Height: 966,
		Body:   inv.readingBody(c),
	}}
}

func (inv *Inventory) readingBody(c tokens.ColorTokens) layout.Widget {
	style := markdown.FromTokens(c, tokens.DefaultTypography)
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(560))
		// LayoutColumn rather than Layout: the document is embedded in a
		// column that scrolls already, and its own viewport would fight the
		// outer one.
		return inv.doc.LayoutColumn(gtx, inv.shaper, style)
	}
}

package input

// placeholderShows reports whether a text field draws its placeholder: while
// its value, valueLen long, is empty. Every text field draws through this
// rule, bordered, search and borderless, live and static. Focus is not an
// input: the placeholder stays while the empty field holds the keyboard, and
// it leaves when the user types the first character.
func placeholderShows(valueLen int) bool {
	return valueLen == 0
}

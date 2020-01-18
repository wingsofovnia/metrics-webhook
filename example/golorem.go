package main

const Lipsum = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore " +
	"et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex " +
	"ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat " +
	"nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim " +
	"id est laborum."

func Lorem(chars int) []string {
	lastPage := (chars + len(Lipsum) - 1) / len(Lipsum)

	var pages []string
	for page := 1; page <= lastPage; page++ {
		prevPagesChars := len(Lipsum) * (page - 1)
		charsLeft := chars - prevPagesChars
		currPageChars := min(len(Lipsum), charsLeft)

		pages = append(pages, Lipsum[:currPageChars])
	}

	return pages
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

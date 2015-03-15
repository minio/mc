package x2j

import (
	"fmt"
	"testing"
)

// the basic demo/test case - a small bibliography with mixed element types
func TestPathsForKey(t *testing.T) {
	fmt.Println("\nPathsForKey... doc1#author")
	m, _ := DocToMap(doc1)
	ss := PathsForKey(m, "author")
	fmt.Println("ss:", ss)

	fmt.Println("\nPathsForKey... doc1#books")
	// m, _ := DocToMap(doc1)
	ss = PathsForKey(m, "books")
	fmt.Println("ss:", ss)

	fmt.Println("\nPathsForKey...doc2#book")
	m, _ = DocToMap(doc2)
	ss = PathsForKey(m, "book")
	fmt.Println("ss:", ss)

	fmt.Println("\nPathForKeyShortest...doc2#book")
	m, _ = DocToMap(doc2)
	s := PathForKeyShortest(m, "book")
	fmt.Println("s:", s)
}

// the basic demo/test case - a small bibliography with mixed element types
func TestPathsForTag(t *testing.T) {
	fmt.Println("\nPathsForTag... doc1#author")
	ss, _ := PathsForTag(doc1, "author")
	fmt.Println("ss:", ss)

	fmt.Println("\nPathsForTag... doc1#books")
	ss, _ = PathsForTag(doc1, "books")
	fmt.Println("ss:", ss)

	fmt.Println("\nPathsForTag...doc2#book")
	ss, _ = PathsForTag(doc2, "book")
	fmt.Println("ss:", ss)

	fmt.Println("\nPathForTagShortest...doc2#book")
	s, _ := PathForTagShortest(doc2, "book")
	fmt.Println("s:", s)
}

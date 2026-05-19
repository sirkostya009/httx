package radix

import (
	"net/http"
	"regexp"
)

type nodeType uint8

const (
	root nodeType = iota
	static
	param
	wildcard
)

type nodeWildcard struct {
	path     string
	paramKey string
	handler  http.Handler
}

type node struct {
	path         string
	handler      http.Handler
	children     []*node
	wildcard     *nodeWildcard

	paramKeys  []string
	paramRegex *regexp.Regexp

	nType nodeType

	tsr          bool
	hasWildChild bool
}

type wildPath struct {
	path  string
	keys  []string

	pattern string
	regex   *regexp.Regexp

	start int
	end   int

	pType nodeType
}

// Tree is a routes storage
type Tree struct {
	root *node

	// If enabled, the node handler could be updated
	Mutable bool
}

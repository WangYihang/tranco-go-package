// Package model holds data structures for persisting a domain's historical
// Tranco ranks (e.g. to a document store, keyed by date).
package model

// Rank is a domain's Tranco rank as of a given date.
type Rank struct {
	Rank int64  `json:"rank"`
	Date string `json:"date"`
}

// Domain is a domain name together with its known Tranco ranks, keyed by
// date (in "2006-01-02" format).
type Domain struct {
	Name        string          `json:"name"`
	TrancoRanks map[string]Rank `json:"tranco_ranks"`
}

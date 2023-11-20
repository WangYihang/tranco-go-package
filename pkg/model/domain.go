package model

type Rank struct {
	Rank int64  `json:"rank"`
	Date string `json:"date"`
}

type Domain struct {
	Name        string          `json:"name"`
	TrancoRanks map[string]Rank `json:"tranco_ranks"`
}

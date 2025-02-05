package structs

// SearchInput is the main object provided to any search query.
type SearchInput struct {
	First          *int              `json:"first"`
	After          *string           `json:"after"`
	FullTextSearch *string           `json:"fullTextSearch"`
	Predicates     *[]QueryPredicate `json:"predicates"`
	OrderBy        *QueryOrder       `json:"orderBy"`
}

// QueryOrder is the order in which the results
// should be returned.
type QueryOrder struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

// QueryPredicate Field and Constraint pair
// for search input.
type QueryPredicate struct {
	Field      string               `json:"field"`
	Constraint QueryFieldConstraint `json:"constraint"`
	Exclude    bool                 `json:"exclude"`
}

// QueryFieldConstraint is a constraint used
// in a search query.
type QueryFieldConstraint struct {
	BooleanEquals *[]bool   `json:"booleanEquals"`
	EnumEquals    *[]string `json:"enumEquals"`
	StringMatches *[]string `json:"stringMatches"`
}

// PageInfo is the extra information about
// a searched page.
type PageInfo struct {
	EndCursor       string `json:"endCursor"`
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
}

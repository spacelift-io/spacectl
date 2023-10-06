package structs

import "github.com/shurcooL/graphql"

// SearchInput is the main object provided to any search query.
type SearchInput struct {
	First          *graphql.Int      `json:"first"`
	After          *graphql.String   `json:"after"`
	FullTextSearch *graphql.String   `json:"fullTextSearch"`
	Predicates     *[]QueryPredicate `json:"predicates"`
	OrderBy        *QueryOrder       `json:"orderBy"`
}

// QueryOrder is the order in which the results
// should be returned.
type QueryOrder struct {
	Field     graphql.String `json:"field"`
	Direction graphql.String `json:"direction"`
}

// QueryPredicate Field and Constraint pair
// for search input.
type QueryPredicate struct {
	Field      graphql.String       `json:"field"`
	Constraint QueryFieldConstraint `json:"constraint"`
}

// QueryFieldConstraint is a constraint used
// in a search query.
type QueryFieldConstraint struct {
	BooleanEquals *[]graphql.Boolean `json:"booleanEquals"`
	EnumEquals    *[]graphql.String  `json:"enumEquals"`
	StringMatches *[]graphql.String  `json:"stringMatches"`
}

// PageInfo is the extra information about
// a searched page.
type PageInfo struct {
	EndCursor       string `json:"endCursor"`
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
}

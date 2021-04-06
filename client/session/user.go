package session

type user struct {
	JWT        string `graphql:"jwt"`
	ValidUntil int64  `graphql:"validUntil"`
}

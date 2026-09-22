package constants

// SortOrder defines custom type for enum-like list sorting.
type SortOrder string

// List sorting keys (enum-like)
const (
	SortByLatest      SortOrder = "latest"
	SortByEarliest    SortOrder = "earliest"
	SortByLastUpdated SortOrder = "lastUpdated"
	SortByTitleAsc    SortOrder = "title-asc"
	SortByTitleDesc   SortOrder = "title-desc"
	SortByNameAsc     SortOrder = "name-asc"
	SortByNameDesc    SortOrder = "name-desc"
	SortByEmailAsc    SortOrder = "email-asc"
	SortByEmailDesc   SortOrder = "email-desc"
)


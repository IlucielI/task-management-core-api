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
)

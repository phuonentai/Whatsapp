package services

// ListResult carries the items of a single page along with the total number of
// rows matching the request's filters (ignoring limit/offset).
type ListResult[T any] struct {
	Items []T
	Total int32
}

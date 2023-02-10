package graph

type (
	Iterator[T any] interface {
		HasNext() bool
		Next() T
	}
)

func Loop[T any](iterator Iterator[T], handler func(T)) {
	for iterator.HasNext() {
		next := iterator.Next()
		handler(next)
	}
}

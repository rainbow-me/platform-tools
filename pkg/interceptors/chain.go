package interceptors

import (
	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
)

// Chain is a generic ordered chain that supports a variety of interactions to modify the internal state.
// None of the operations are concurrency-safe and may panic of the wrong types are passed.
type Chain struct {
	itemOrder []string
}

// UnaryServerInterceptorChain builds and requires grpc.UnaryServerInterceptor's
type UnaryServerInterceptorChain struct {
	Chain
	items map[string]grpc.UnaryServerInterceptor
}

// StreamServerInterceptorChain builds and requires grpc.StreamServerInterceptor's
type StreamServerInterceptorChain struct {
	Chain
	items map[string]grpc.StreamServerInterceptor
}

// UnaryClientInterceptorChain builds and requires grpc.UnaryClientInterceptor's
type UnaryClientInterceptorChain struct {
	Chain
	items map[string]grpc.UnaryClientInterceptor
}

// StreamClientInterceptorChain builds and requires grpc.StreamClientInterceptor's
type StreamClientInterceptorChain struct {
	Chain
	items map[string]grpc.StreamClientInterceptor
}

func (c *UnaryServerInterceptorChain) Exists(id string) bool {
	_, ok := c.items[id]
	return ok
}

func (c *StreamServerInterceptorChain) Exists(id string) bool {
	_, ok := c.items[id]
	return ok
}

func (c *UnaryClientInterceptorChain) Exists(id string) bool {
	_, ok := c.items[id]
	return ok
}

func (c *StreamClientInterceptorChain) Exists(id string) bool {
	_, ok := c.items[id]
	return ok
}

// Push adds a new interceptor onto the end of the chain.
// Returns a boolean about whether an item with the specified ID already exists.
// Push("b", <inter>)
//
//	Before: a
//	After: a -> b
func (c *UnaryServerInterceptorChain) Push(id string, inter grpc.UnaryServerInterceptor) bool {
	if _, ok := c.items[id]; ok {
		return false
	}

	c.items[id] = inter
	c.itemOrder = append(c.itemOrder, id)

	return true
}

// Push adds a new interceptor onto the end of the chain.
// Returns a boolean about whether an item with the specified ID already exists.
// Push("b", <inter>)
//
//	Before: a
//	After: a -> b
func (c *StreamServerInterceptorChain) Push(id string, inter grpc.StreamServerInterceptor) bool {
	if _, ok := c.items[id]; ok {
		return false
	}

	c.items[id] = inter
	c.itemOrder = append(c.itemOrder, id)

	return true
}

// Push adds a new interceptor onto the end of the chain.
// Returns a boolean about whether an item with the specified ID already exists.
// Push("b", <inter>)
//
//	Before: a
//	After: a -> b
func (c *UnaryClientInterceptorChain) Push(id string, inter grpc.UnaryClientInterceptor) bool {
	if _, ok := c.items[id]; ok {
		return false
	}

	c.items[id] = inter
	c.itemOrder = append(c.itemOrder, id)

	return true
}

// Push adds a new interceptor onto the end of the chain.
// Returns a boolean about whether an item with the specified ID already exists.
// Push("b", <inter>)
//
//	Before: a
//	After: a -> b
func (c *StreamClientInterceptorChain) Push(id string, inter grpc.StreamClientInterceptor) bool {
	if _, ok := c.items[id]; ok {
		return false
	}

	c.items[id] = inter
	c.itemOrder = append(c.itemOrder, id)

	return true
}

// InsertAfter inserts an interceptor after the specified interceptor in the chain.
// Returns a boolean about whether the operation was successful.
// InsertAfter("a", "c", <inter>)
//
//	Before: a -> b
//	After: a -> c -> b
func (c *UnaryServerInterceptorChain) InsertAfter(afterID string, id string, inter grpc.UnaryServerInterceptor) bool {
	if _, ok := c.items[id]; ok {
		return false
	}

	if _, ok := c.items[afterID]; !ok {
		return false
	}

	var index int
	for i := range c.itemOrder {
		if c.itemOrder[i] == afterID {
			index = i
			break
		}
	}

	c.itemOrder = append(c.itemOrder[:index+1],
		append([]string{id}, c.itemOrder[index+1:]...)...)
	c.items[id] = inter

	return true
}

// InsertAfter inserts an interceptor after the specified interceptor in the chain.
// Returns a boolean about whether the operation was successful.
// InsertAfter("a", "c", <inter>)
//
//	Before: a -> b
//	After: a -> c -> b
func (c *StreamServerInterceptorChain) InsertAfter(afterID string, id string, inter grpc.StreamServerInterceptor) bool {
	if _, ok := c.items[id]; ok {
		return false
	}

	if _, ok := c.items[afterID]; !ok {
		return false
	}

	var index int
	for i := range c.itemOrder {
		if c.itemOrder[i] == afterID {
			index = i
			break
		}
	}

	c.itemOrder = append(c.itemOrder[:index+1],
		append([]string{id}, c.itemOrder[index+1:]...)...)
	c.items[id] = inter

	return true
}

// InsertAfter inserts an interceptor after the specified interceptor in the chain.
// Returns a boolean about whether the operation was successful.
// InsertAfter("a", "c", <inter>)
//
//	Before: a -> b
//	After: a -> c -> b
func (c *UnaryClientInterceptorChain) InsertAfter(afterID string, id string, inter grpc.UnaryClientInterceptor) bool {
	if _, ok := c.items[id]; ok {
		return false
	}

	if _, ok := c.items[afterID]; !ok {
		return false
	}

	var index int
	for i := range c.itemOrder {
		if c.itemOrder[i] == afterID {
			index = i
			break
		}
	}

	c.itemOrder = append(c.itemOrder[:index+1],
		append([]string{id}, c.itemOrder[index+1:]...)...)
	c.items[id] = inter

	return true
}

// InsertAfter inserts an interceptor after the specified interceptor in the chain.
// Returns a boolean about whether the operation was successful.
// InsertAfter("a", "c", <inter>)
//
//	Before: a -> b
//	After: a -> c -> b
func (c *StreamClientInterceptorChain) InsertAfter(afterID string, id string, inter grpc.StreamClientInterceptor) bool {
	if _, ok := c.items[id]; ok {
		return false
	}

	if _, ok := c.items[afterID]; !ok {
		return false
	}

	var index int
	for i := range c.itemOrder {
		if c.itemOrder[i] == afterID {
			index = i
			break
		}
	}

	c.itemOrder = append(c.itemOrder[:index+1],
		append([]string{id}, c.itemOrder[index+1:]...)...)
	c.items[id] = inter

	return true
}

// InsertBefore inserts a new interceptor before the specified interceptor in the chain.
// InsertBefore("b", "c", <inter>)
//
//	Before: a -> b
//	After: a -> c -> b
func (c *UnaryServerInterceptorChain) InsertBefore(beforeID string, id string, inter grpc.UnaryServerInterceptor) bool {
	if _, ok := c.items[id]; ok {
		return false
	}

	if _, ok := c.items[beforeID]; !ok {
		return false
	}

	var index int
	for i := range c.itemOrder {
		if c.itemOrder[i] == beforeID {
			index = i
			break
		}
	}

	if index-1 < 0 {
		c.itemOrder = append([]string{id}, c.itemOrder...)
		c.items[id] = inter
	} else {
		c.itemOrder = append(c.itemOrder[:index-1],
			append([]string{id}, c.itemOrder[index-1:]...)...)
		c.items[id] = inter
	}

	return true
}

// InsertBefore inserts a new interceptor before the specified interceptor in the chain.
// InsertBefore("b", "c", <inter>)
//
//	Before: a -> b
//	After: a -> c -> b
func (c *StreamServerInterceptorChain) InsertBefore(
	beforeID string,
	id string,
	inter grpc.StreamServerInterceptor,
) bool {
	if _, ok := c.items[id]; ok {
		return false
	}

	if _, ok := c.items[beforeID]; !ok {
		return false
	}

	var index int
	for i := range c.itemOrder {
		if c.itemOrder[i] == beforeID {
			index = i
			break
		}
	}

	if index-1 < 0 {
		c.itemOrder = append([]string{id}, c.itemOrder...)
		c.items[id] = inter
	} else {
		c.itemOrder = append(c.itemOrder[:index-1],
			append([]string{id}, c.itemOrder[index-1:]...)...)
		c.items[id] = inter
	}

	return true
}

// InsertBefore inserts a new interceptor before the specified interceptor in the chain.
// InsertBefore("b", "c", <inter>)
//
//	Before: a -> b
//	After: a -> c -> b
func (c *UnaryClientInterceptorChain) InsertBefore(beforeID string, id string, inter grpc.UnaryClientInterceptor) bool {
	if _, ok := c.items[id]; ok {
		return false
	}

	if _, ok := c.items[beforeID]; !ok {
		return false
	}

	var index int
	for i := range c.itemOrder {
		if c.itemOrder[i] == beforeID {
			index = i
			break
		}
	}

	if index-1 < 0 {
		c.itemOrder = append([]string{id}, c.itemOrder...)
		c.items[id] = inter
	} else {
		c.itemOrder = append(c.itemOrder[:index-1],
			append([]string{id}, c.itemOrder[index-1:]...)...)
		c.items[id] = inter
	}

	return true
}

// InsertBefore inserts a new interceptor before the specified interceptor in the chain.
// InsertBefore("b", "c", <inter>)
//
//	Before: a -> b
//	After: a -> c -> b
func (c *StreamClientInterceptorChain) InsertBefore(
	beforeID string,
	id string,
	inter grpc.StreamClientInterceptor,
) bool {
	if _, ok := c.items[id]; ok {
		return false
	}

	if _, ok := c.items[beforeID]; !ok {
		return false
	}

	var index int
	for i := range c.itemOrder {
		if c.itemOrder[i] == beforeID {
			index = i
			break
		}
	}

	if index-1 < 0 {
		c.itemOrder = append([]string{id}, c.itemOrder...)
		c.items[id] = inter
	} else {
		c.itemOrder = append(c.itemOrder[:index-1],
			append([]string{id}, c.itemOrder[index-1:]...)...)
		c.items[id] = inter
	}

	return true
}

// Delete removes the specified interceptor from the list
// Delete("a")
//
//	Before: a -> b
//	After: b
func (c *UnaryServerInterceptorChain) Delete(id string) bool {
	if _, ok := c.items[id]; !ok {
		return false
	}

	var index int
	for i := range c.itemOrder {
		if c.itemOrder[i] == id {
			index = i
			break
		}
	}

	c.itemOrder = append(c.itemOrder[:index], c.itemOrder[index+1:]...)
	delete(c.items, id)

	return true
}

// Delete removes the specified interceptor from the list
// Delete("a")
//
//	Before: a -> b
//	After: b
func (c *StreamServerInterceptorChain) Delete(id string) bool {
	if _, ok := c.items[id]; !ok {
		return false
	}

	var index int
	for i := range c.itemOrder {
		if c.itemOrder[i] == id {
			index = i
			break
		}
	}

	c.itemOrder = append(c.itemOrder[:index], c.itemOrder[index+1:]...)
	delete(c.items, id)

	return true
}

// Delete removes the specified interceptor from the list
// Delete("a")
//
//	Before: a -> b
//	After: b
func (c *UnaryClientInterceptorChain) Delete(id string) bool {
	if _, ok := c.items[id]; !ok {
		return false
	}

	var index int
	for i := range c.itemOrder {
		if c.itemOrder[i] == id {
			index = i
			break
		}
	}

	c.itemOrder = append(c.itemOrder[:index], c.itemOrder[index+1:]...)
	delete(c.items, id)

	return true
}

// Delete removes the specified interceptor from the list
// Delete("a")
//
//	Before: a -> b
//	After: b
func (c *StreamClientInterceptorChain) Delete(id string) bool {
	if _, ok := c.items[id]; !ok {
		return false
	}

	var index int
	for i := range c.itemOrder {
		if c.itemOrder[i] == id {
			index = i
			break
		}
	}

	c.itemOrder = append(c.itemOrder[:index], c.itemOrder[index+1:]...)
	delete(c.items, id)

	return true
}

// Replace replaces the specified interceptor
// Replace("a")
//
//	Before: a -> b
//	After: a (new interceptor) -> b
func (c *UnaryServerInterceptorChain) Replace(id string, inter grpc.UnaryServerInterceptor) bool {
	if _, ok := c.items[id]; !ok {
		return false
	}

	c.items[id] = inter

	return true
}

// Replace replaces the specified interceptor
// Replace("a")
//
//	Before: a -> b
//	After: a (new interceptor) -> b
func (c *StreamServerInterceptorChain) Replace(id string, inter grpc.StreamServerInterceptor) bool {
	if _, ok := c.items[id]; !ok {
		return false
	}

	c.items[id] = inter

	return true
}

// Replace replaces the specified interceptor
// Replace("a")
//
//	Before: a -> b
//	After: a (new interceptor) -> b
func (c *UnaryClientInterceptorChain) Replace(id string, inter grpc.UnaryClientInterceptor) bool {
	if _, ok := c.items[id]; !ok {
		return false
	}

	c.items[id] = inter

	return true
}

// Replace replaces the specified interceptor
// Replace("a")
//
//	Before: a -> b
//	After: a (new interceptor) -> b
func (c *StreamClientInterceptorChain) Replace(id string, inter grpc.StreamClientInterceptor) bool {
	if _, ok := c.items[id]; !ok {
		return false
	}

	c.items[id] = inter

	return true
}

// Commit builds one large list of grpc.UnaryServerInterceptor's.
func (c *UnaryServerInterceptorChain) Commit() grpc.UnaryServerInterceptor {
	var interceptors []grpc.UnaryServerInterceptor

	for _, id := range c.itemOrder {
		interceptors = append(interceptors, c.items[id])
	}

	return grpcmiddleware.ChainUnaryServer(interceptors...)
}

// Commit builds one large list of grpc.UnaryServerInterceptor's.
func (c *StreamServerInterceptorChain) Commit() grpc.StreamServerInterceptor {
	var interceptors []grpc.StreamServerInterceptor

	for _, id := range c.itemOrder {
		interceptors = append(interceptors, c.items[id])
	}

	return grpcmiddleware.ChainStreamServer(interceptors...)
}

// Commit builds one large list of grpc.UnaryClientInterceptor's.
func (c *UnaryClientInterceptorChain) Commit() grpc.UnaryClientInterceptor {
	var interceptors []grpc.UnaryClientInterceptor

	for _, id := range c.itemOrder {
		interceptors = append(interceptors, c.items[id])
	}

	return grpcmiddleware.ChainUnaryClient(interceptors...)
}

// Commit builds one large list of grpc.UnaryClientInterceptor's.
func (c *StreamClientInterceptorChain) Commit() grpc.StreamClientInterceptor {
	var interceptors []grpc.StreamClientInterceptor

	for _, id := range c.itemOrder {
		interceptors = append(interceptors, c.items[id])
	}

	return grpcmiddleware.ChainStreamClient(interceptors...)
}

// NewUnaryServerInterceptorChain constructs a new interceptor chain that can be modified.
func NewUnaryServerInterceptorChain() *UnaryServerInterceptorChain {
	return &UnaryServerInterceptorChain{
		items: make(map[string]grpc.UnaryServerInterceptor),
	}
}

// NewStreamServerInterceptorChain constructs a new interceptor chain that can be modified.
func NewStreamServerInterceptorChain() *StreamServerInterceptorChain {
	return &StreamServerInterceptorChain{
		items: make(map[string]grpc.StreamServerInterceptor),
	}
}

// NewUnaryClientInterceptorChain constructs a new interceptor chain that can be modified.
func NewUnaryClientInterceptorChain() *UnaryClientInterceptorChain {
	return &UnaryClientInterceptorChain{
		items: make(map[string]grpc.UnaryClientInterceptor),
	}
}

// NewStreamClientInterceptorChain constructs a new interceptor chain that can be modified.
func NewStreamClientInterceptorChain() *StreamClientInterceptorChain {
	return &StreamClientInterceptorChain{
		items: make(map[string]grpc.StreamClientInterceptor),
	}
}

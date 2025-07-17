package interceptors_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/rainbow-me/platform-tools/grpc/interceptors"
)

func TestUnaryServerInterceptorChain(t *testing.T) {
	i1 := func(
		_ context.Context,
		_ interface{},
		_ *grpc.UnaryServerInfo,
		_ grpc.UnaryHandler,
	) (interface{}, error) {
		return 1, nil
	}

	i2 := func(
		_ context.Context,
		_ interface{},
		_ *grpc.UnaryServerInfo,
		_ grpc.UnaryHandler,
	) (interface{}, error) {
		return 2, nil
	}

	i3 := func(
		_ context.Context,
		_ interface{},
		_ *grpc.UnaryServerInfo,
		_ grpc.UnaryHandler,
	) (interface{}, error) {
		return 3, nil
	}

	t.Run("Push", func(t *testing.T) {
		chain := interceptors.NewUnaryServerInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.False(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.Push("c", i3))

		assert.Equal(t, []int{1, 2, 3}, FakeUnaryServerInterceptorChainCommit(chain))
	})

	t.Run("InsertAfter", func(t *testing.T) {
		chain := interceptors.NewUnaryServerInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.InsertAfter("a", "c", i3))

		assert.Equal(t, []int{1, 3, 2}, FakeUnaryServerInterceptorChainCommit(chain))
	})

	t.Run("InsertBefore", func(t *testing.T) {
		chain := interceptors.NewUnaryServerInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.InsertBefore("a", "c", i3))

		assert.Equal(t, []int{3, 1, 2}, FakeUnaryServerInterceptorChainCommit(chain))
	})

	t.Run("Replace", func(t *testing.T) {
		chain := interceptors.NewUnaryServerInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.Replace("a", i2))

		assert.Equal(t, []int{2, 2}, FakeUnaryServerInterceptorChainCommit(chain))
	})

	t.Run("Delete", func(t *testing.T) {
		chain := interceptors.NewUnaryServerInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.Push("c", i3))
		assert.True(t, chain.Delete("b"))

		assert.Equal(t, []int{1, 3}, FakeUnaryServerInterceptorChainCommit(chain))
	})
}

func TestStreamServerInterceptorChain(t *testing.T) {
	i1 := func(
		_ interface{},
		_ grpc.ServerStream,
		_ *grpc.StreamServerInfo,
		_ grpc.StreamHandler,
	) error {
		return errors.New("1")
	}

	i2 := func(
		_ interface{},
		_ grpc.ServerStream,
		_ *grpc.StreamServerInfo,
		_ grpc.StreamHandler,
	) error {
		return errors.New("2")
	}

	i3 := func(
		_ interface{},
		_ grpc.ServerStream,
		_ *grpc.StreamServerInfo,
		_ grpc.StreamHandler,
	) error {
		return errors.New("3")
	}

	t.Run("Push", func(t *testing.T) {
		chain := interceptors.NewStreamServerInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.False(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.Push("c", i3))

		expected := []error{errors.New("1"), errors.New("2"), errors.New("3")}
		assert.Equal(t, expected, FakeStreamServerInterceptorChainCommit(chain))
	})

	t.Run("InsertAfter", func(t *testing.T) {
		chain := interceptors.NewStreamServerInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.InsertAfter("a", "c", i3))

		expected := []error{errors.New("1"), errors.New("3"), errors.New("2")}
		assert.Equal(t, expected, FakeStreamServerInterceptorChainCommit(chain))
	})

	t.Run("InsertBefore", func(t *testing.T) {
		chain := interceptors.NewStreamServerInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.InsertBefore("a", "c", i3))

		expected := []error{errors.New("3"), errors.New("1"), errors.New("2")}
		assert.Equal(t, expected, FakeStreamServerInterceptorChainCommit(chain))
	})

	t.Run("Replace", func(t *testing.T) {
		chain := interceptors.NewStreamServerInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.Replace("a", i2))

		expected := []error{errors.New("2"), errors.New("2")}
		assert.Equal(t, expected, FakeStreamServerInterceptorChainCommit(chain))
	})

	t.Run("Delete", func(t *testing.T) {
		chain := interceptors.NewStreamServerInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.Push("c", i3))
		assert.True(t, chain.Delete("b"))

		expected := []error{errors.New("1"), errors.New("3")}
		assert.Equal(t, expected, FakeStreamServerInterceptorChainCommit(chain))
	})
}

func TestUnaryClientInterceptorChain(t *testing.T) {
	i1 := func(
		_ context.Context,
		_ string,
		_, _ interface{},
		_ *grpc.ClientConn,
		_ grpc.UnaryInvoker,
		_ ...grpc.CallOption,
	) error {
		return errors.New("1")
	}

	i2 := func(
		_ context.Context,
		_ string,
		_, _ interface{},
		_ *grpc.ClientConn,
		_ grpc.UnaryInvoker,
		_ ...grpc.CallOption,
	) error {
		return errors.New("2")
	}

	i3 := func(
		_ context.Context,
		_ string,
		_, _ interface{},
		_ *grpc.ClientConn,
		_ grpc.UnaryInvoker,
		_ ...grpc.CallOption,
	) error {
		return errors.New("3")
	}

	t.Run("Push", func(t *testing.T) {
		chain := interceptors.NewUnaryClientInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.False(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.Push("c", i3))

		expected := []error{errors.New("1"), errors.New("2"), errors.New("3")}
		assert.Equal(t, expected, FakeUnaryClientInterceptorChainCommit(chain))
	})

	t.Run("InsertAfter", func(t *testing.T) {
		chain := interceptors.NewUnaryClientInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.InsertAfter("a", "c", i3))

		expected := []error{errors.New("1"), errors.New("3"), errors.New("2")}
		assert.Equal(t, expected, FakeUnaryClientInterceptorChainCommit(chain))
	})

	t.Run("InsertBefore", func(t *testing.T) {
		chain := interceptors.NewUnaryClientInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.InsertBefore("a", "c", i3))

		expected := []error{errors.New("3"), errors.New("1"), errors.New("2")}
		assert.Equal(t, expected, FakeUnaryClientInterceptorChainCommit(chain))
	})

	t.Run("Replace", func(t *testing.T) {
		chain := interceptors.NewUnaryClientInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.Replace("a", i2))

		expected := []error{errors.New("2"), errors.New("2")}
		assert.Equal(t, expected, FakeUnaryClientInterceptorChainCommit(chain))
	})

	t.Run("Delete", func(t *testing.T) {
		chain := interceptors.NewUnaryClientInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.Push("c", i3))
		assert.True(t, chain.Delete("b"))

		expected := []error{errors.New("1"), errors.New("3")}
		assert.Equal(t, expected, FakeUnaryClientInterceptorChainCommit(chain))
	})
}

func TestStreamClientInterceptorChain(t *testing.T) {
	i1 := func(
		_ context.Context,
		_ *grpc.StreamDesc,
		_ *grpc.ClientConn,
		_ string,
		_ grpc.Streamer,
		_ ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		return nil, errors.New("1")
	}

	i2 := func(
		_ context.Context,
		_ *grpc.StreamDesc,
		_ *grpc.ClientConn,
		_ string,
		_ grpc.Streamer,
		_ ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		return nil, errors.New("2")
	}

	i3 := func(
		_ context.Context,
		_ *grpc.StreamDesc,
		_ *grpc.ClientConn,
		_ string,
		_ grpc.Streamer,
		_ ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		return nil, errors.New("3")
	}

	t.Run("Push", func(t *testing.T) {
		chain := interceptors.NewStreamClientInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.False(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.Push("c", i3))

		expected := []error{errors.New("1"), errors.New("2"), errors.New("3")}
		assert.Equal(t, expected, FakeStreamClientInterceptorChainCommit(chain))
	})

	t.Run("InsertAfter", func(t *testing.T) {
		chain := interceptors.NewStreamClientInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.InsertAfter("a", "c", i3))

		expected := []error{errors.New("1"), errors.New("3"), errors.New("2")}
		assert.Equal(t, expected, FakeStreamClientInterceptorChainCommit(chain))
	})

	t.Run("InsertBefore", func(t *testing.T) {
		chain := interceptors.NewStreamClientInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.InsertBefore("a", "c", i3))

		expected := []error{errors.New("3"), errors.New("1"), errors.New("2")}
		assert.Equal(t, expected, FakeStreamClientInterceptorChainCommit(chain))
	})

	t.Run("Replace", func(t *testing.T) {
		chain := interceptors.NewStreamClientInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.Replace("a", i2))

		expected := []error{errors.New("2"), errors.New("2")}
		assert.Equal(t, expected, FakeStreamClientInterceptorChainCommit(chain))
	})

	t.Run("Delete", func(t *testing.T) {
		chain := interceptors.NewStreamClientInterceptorChain()

		assert.True(t, chain.Push("a", i1))
		assert.True(t, chain.Push("b", i2))
		assert.True(t, chain.Push("c", i3))
		assert.True(t, chain.Delete("b"))

		expected := []error{errors.New("1"), errors.New("3")}
		assert.Equal(t, expected, FakeStreamClientInterceptorChainCommit(chain))
	})
}

func FakeUnaryServerInterceptorChainCommit(c *interceptors.UnaryServerInterceptorChain) []int {
	var results []int
	for _, id := range c.ItemOrder {
		r, _ := c.Items[id](nil, nil, nil, nil)
		results = append(results, r.(int)) //nolint:errcheck
	}
	return results
}

func FakeStreamServerInterceptorChainCommit(c *interceptors.StreamServerInterceptorChain) []error {
	var results []error
	for _, id := range c.ItemOrder {
		r := c.Items[id](nil, nil, nil, nil)
		results = append(results, r)
	}
	return results
}

func FakeUnaryClientInterceptorChainCommit(c *interceptors.UnaryClientInterceptorChain) []error {
	var results []error
	for _, id := range c.ItemOrder {
		r := c.Items[id](nil, "", nil, nil, nil, nil)
		results = append(results, r)
	}
	return results
}

func FakeStreamClientInterceptorChainCommit(c *interceptors.StreamClientInterceptorChain) []error {
	var results []error
	for _, id := range c.ItemOrder {
		_, r := c.Items[id](nil, nil, nil, "", nil)
		results = append(results, r)
	}
	return results
}

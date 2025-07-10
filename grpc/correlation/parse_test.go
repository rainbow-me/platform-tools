package correlation_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rainbow-me/platfomt-tools/grpc/correlation"
)

func TestGenerate(t *testing.T) {
	cases := []struct {
		strings  []string
		expected map[string]string
	}{
		{
			[]string{"test=true"},
			map[string]string{
				"test": "true",
			},
		},
		{
			[]string{"test=1"},
			map[string]string{
				"test": "1",
			},
		},
		{
			// Go map ordering is non-derministic.
			[]string{"test=1,k=v", "k=v,test=1"},
			map[string]string{
				"test": "1",
				"k":    "v",
			},
		},
		{
			[]string{"test=a+b"},
			map[string]string{
				"test": "a b",
			},
		},
		{
			[]string{""},
			nil,
		},
	}

	for _, c := range cases {
		ctx := correlation.Set(context.Background(), c.expected)

		assert.Contains(t, c.strings, correlation.Generate(ctx))
	}
}

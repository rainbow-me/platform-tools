package correlation_test

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"google.golang.org/grpc/metadata"

	corr "github.com/rainbow-me/platform-tools/common/correlation"
	"github.com/rainbow-me/platform-tools/common/logger"
	meta "github.com/rainbow-me/platform-tools/grpc/metadata"
)

func TestSet(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		values      map[string]string
		wantData    corr.Data
		wantBaggage map[string]string // Only checked if span present
	}{
		{
			name:        "empty values no span",
			ctx:         context.Background(),
			values:      map[string]string{},
			wantData:    corr.Data{},
			wantBaggage: nil,
		},
		{
			name:        "empty values with span",
			ctx:         tracer.ContextWithSpan(context.Background(), tracer.StartSpan("test")),
			values:      map[string]string{},
			wantData:    corr.Data{},
			wantBaggage: map[string]string{},
		},
		{
			name:        "valid values no span",
			ctx:         context.Background(),
			values:      map[string]string{"key1": "val1", "key2": "val2"},
			wantData:    corr.Data{"key1": "val1", "key2": "val2"},
			wantBaggage: nil,
		},
		{
			name:        "valid values with span",
			ctx:         tracer.ContextWithSpan(context.Background(), tracer.StartSpan("test")),
			values:      map[string]string{"key1": "val1", "key2": "val2"},
			wantData:    corr.Data{"key1": "val1", "key2": "val2"},
			wantBaggage: map[string]string{"key1": "val1", "key2": "val2"},
		},
		{
			name:        "skip empty keys and values",
			ctx:         context.Background(),
			values:      map[string]string{"": "val", "key": "", "valid": "val"},
			wantData:    corr.Data{"valid": "val"},
			wantBaggage: nil,
		},
		{
			name:        "skip empty keys and values with span",
			ctx:         tracer.ContextWithSpan(context.Background(), tracer.StartSpan("test")),
			values:      map[string]string{"": "val", "key": "", "valid": "val"},
			wantData:    corr.Data{"valid": "val"},
			wantBaggage: map[string]string{"valid": "val"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCtx := corr.Set(tt.ctx, tt.values)
			gotData := corr.Get(gotCtx)
			if !reflect.DeepEqual(gotData, tt.wantData) {
				t.Errorf("Set() got data = %v, want %v", gotData, tt.wantData)
			}
		})
	}
}

func TestSetKey(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		key         string
		value       string
		wantData    corr.Data
		wantBaggage map[string]string // Only checked if span present
	}{
		{
			name:        "empty key no change no span",
			ctx:         context.Background(),
			key:         "",
			value:       "val",
			wantData:    corr.Data{},
			wantBaggage: nil,
		},
		{
			name:        "empty key no change with span",
			ctx:         tracer.ContextWithSpan(context.Background(), tracer.StartSpan("test")),
			key:         "",
			value:       "val",
			wantData:    corr.Data{},
			wantBaggage: map[string]string{},
		},
		{
			name:        "add new key no span",
			ctx:         context.Background(),
			key:         "key1",
			value:       "val1",
			wantData:    corr.Data{"key1": "val1"},
			wantBaggage: nil,
		},
		{
			name:        "add new key with span",
			ctx:         tracer.ContextWithSpan(context.Background(), tracer.StartSpan("test")),
			key:         "key1",
			value:       "val1",
			wantData:    corr.Data{"key1": "val1"},
			wantBaggage: map[string]string{"key1": "val1"},
		},
		{
			name:        "update existing key",
			ctx:         corr.Set(context.Background(), map[string]string{"key1": "old"}),
			key:         "key1",
			value:       "new",
			wantData:    corr.Data{"key1": "new"},
			wantBaggage: nil,
		},
		{
			name:        "delete with empty value",
			ctx:         corr.Set(context.Background(), map[string]string{"key1": "val"}),
			key:         "key1",
			value:       "",
			wantData:    corr.Data{},
			wantBaggage: nil,
		},
		{
			name: "delete with empty value with span",
			ctx: corr.Set(tracer.ContextWithSpan(
				context.Background(),
				tracer.StartSpan("test")),
				map[string]string{"key1": "val"},
			),
			key:         "key1",
			value:       "",
			wantData:    corr.Data{},
			wantBaggage: map[string]string{"key1": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCtx := corr.SetKey(tt.ctx, tt.key, tt.value)
			gotData := corr.Get(gotCtx)
			if !reflect.DeepEqual(gotData, tt.wantData) {
				t.Errorf("SetKey() got data = %v, want %v", gotData, tt.wantData)
			}
		})
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want corr.Data
	}{
		{
			name: "nil ctx",
			ctx:  nil,
			want: corr.Data{},
		},
		{
			name: "empty ctx",
			ctx:  context.Background(),
			want: corr.Data{},
		},
		{
			name: "with data",
			ctx:  corr.Set(context.Background(), map[string]string{"key": "val"}),
			want: corr.Data{"key": "val"},
		},
		{
			name: "invalid value type",
			ctx:  context.WithValue(context.Background(), corr.Key, "not map"),
			want: corr.Data{},
		},
		{
			name: "nil value",
			ctx:  context.WithValue(context.Background(), corr.Key, nil),
			want: corr.Data{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := corr.Get(tt.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetValue(t *testing.T) {
	baseCtx := corr.Set(context.Background(), map[string]string{"key1": "val1"})

	tests := []struct {
		name string
		ctx  context.Context
		key  string
		want string
	}{
		{
			name: "empty key",
			ctx:  baseCtx,
			key:  "",
			want: "",
		},
		{
			name: "existing key",
			ctx:  baseCtx,
			key:  "key1",
			want: "val1",
		},
		{
			name: "non-existing key",
			ctx:  baseCtx,
			key:  "key2",
			want: "",
		},
		{
			name: "nil ctx",
			ctx:  nil,
			key:  "key1",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := corr.GetValue(tt.ctx, tt.key); got != tt.want {
				t.Errorf("GetValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHas(t *testing.T) {
	baseCtx := corr.Set(context.Background(), map[string]string{"key1": "val1"})

	tests := []struct {
		name string
		ctx  context.Context
		key  string
		want bool
	}{
		{
			name: "empty key",
			ctx:  baseCtx,
			key:  "",
			want: false,
		},
		{
			name: "existing key",
			ctx:  baseCtx,
			key:  "key1",
			want: true,
		},
		{
			name: "non-existing key",
			ctx:  baseCtx,
			key:  "key2",
			want: false,
		},
		{
			name: "nil ctx",
			ctx:  nil,
			key:  "key1",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := corr.Has(tt.ctx, tt.key); got != tt.want {
				t.Errorf("Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		key      string
		wantData corr.Data
	}{
		{
			name:     "empty key no change",
			ctx:      corr.Set(context.Background(), map[string]string{"key1": "val1"}),
			key:      "",
			wantData: corr.Data{"key1": "val1"},
		},
		{
			name:     "non-existing key no change",
			ctx:      corr.Set(context.Background(), map[string]string{"key1": "val1"}),
			key:      "key2",
			wantData: corr.Data{"key1": "val1"},
		},
		{
			name:     "delete existing key",
			ctx:      corr.Set(context.Background(), map[string]string{"key1": "val1", "key2": "val2"}),
			key:      "key1",
			wantData: corr.Data{"key2": "val2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCtx := corr.Delete(tt.ctx, tt.key)
			gotData := corr.Get(gotCtx)
			if !reflect.DeepEqual(gotData, tt.wantData) {
				t.Errorf("Delete() got data = %v, want %v", gotData, tt.wantData)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	baseCtx := corr.Set(context.Background(), map[string]string{"key1": "base1", "key2": "base2"})
	other1 := corr.Set(context.Background(), map[string]string{"key2": "other1", "key3": "other3"})
	other2 := corr.Set(context.Background(), map[string]string{"key3": "", "key4": "other4"})

	tests := []struct {
		name          string
		ctx           context.Context
		otherContexts []context.Context
		wantData      corr.Data
	}{
		{
			name:          "no others",
			ctx:           baseCtx,
			otherContexts: []context.Context{},
			wantData:      corr.Data{"key1": "base1", "key2": "base2"},
		},
		{
			name:          "merge one override and add",
			ctx:           baseCtx,
			otherContexts: []context.Context{other1},
			wantData:      corr.Data{"key1": "base1", "key2": "other1", "key3": "other3"},
		},
		{
			name:          "merge multiple with empty value skip",
			ctx:           baseCtx,
			otherContexts: []context.Context{other1, other2},
			wantData:      corr.Data{"key1": "base1", "key2": "other1", "key3": "other3", "key4": "other4"},
		},
		{
			name:          "empty base",
			ctx:           context.Background(),
			otherContexts: []context.Context{other1},
			wantData:      corr.Data{"key2": "other1", "key3": "other3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCtx := corr.Merge(tt.ctx, tt.otherContexts...)
			gotData := corr.Get(gotCtx)
			if !reflect.DeepEqual(gotData, tt.wantData) {
				t.Errorf("Merge() got data = %v, want %v", gotData, tt.wantData)
			}
		})
	}
}

func TestToZapFields(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want []logger.Field
	}{
		{
			name: "empty data",
			ctx:  context.Background(),
			want: nil,
		},
		{
			name: "with data skip empty",
			ctx:  corr.Set(context.Background(), map[string]string{"key1": "val1", "key2": ""}),
			want: []logger.Field{logger.String("key1", "val1")},
		},
		{
			name: "multiple fields",
			ctx:  corr.Set(context.Background(), map[string]string{"key1": "val1", "key2": "val2"}),
			want: []logger.Field{logger.String("key1", "val1"), logger.String("key2", "val2")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := corr.ToZapFields(tt.ctx)
			if !zapFieldsEqual(got, tt.want) {
				t.Errorf("ToZapFields() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper to compare logger.Fields (order insensitive)
func zapFieldsEqual(a, b []logger.Field) bool {
	if len(a) != len(b) {
		return false
	}
	am := make(map[string]logger.Field)
	for _, f := range a {
		am[f.Key] = f
	}
	for _, f := range b {
		af, ok := am[f.Key]
		if !ok || af.Type != f.Type || af.String != f.String {
			return false
		}
	}
	return true
}

func TestToMap(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want map[string]string
	}{
		{
			name: "empty",
			ctx:  context.Background(),
			want: map[string]string{},
		},
		{
			name: "with data",
			ctx:  corr.Set(context.Background(), map[string]string{"key1": "val1", "key2": "val2"}),
			want: map[string]string{"key1": "val1", "key2": "val2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := corr.ToMap(tt.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenancy(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			name: "not found",
			ctx:  context.Background(),
			want: "",
		},
		{
			name: "found",
			ctx:  corr.SetTenancy(context.Background(), "org1"),
			want: "org1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := corr.Tenancy(tt.ctx); got != tt.want {
				t.Errorf("Tenancy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetTenancy(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		val  string
		want string
	}{
		{
			name: "set",
			ctx:  context.Background(),
			val:  "org1",
			want: "org1",
		},
		{
			name: "update",
			ctx:  corr.SetTenancy(context.Background(), "old"),
			val:  "new",
			want: "new",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCtx := corr.SetTenancy(tt.ctx, tt.val)
			if got := corr.Tenancy(gotCtx); got != tt.want {
				t.Errorf("SetTenancy() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestID(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			name: "not found",
			ctx:  context.Background(),
			want: "",
		},
		{
			name: "found",
			ctx:  corr.SetID(context.Background(), "id1"),
			want: "id1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := corr.ID(tt.ctx); got != tt.want {
				t.Errorf("ID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetID(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		val  string
		want string
	}{
		{
			name: "set",
			ctx:  context.Background(),
			val:  "id1",
			want: "id1",
		},
		{
			name: "update",
			ctx:  corr.SetID(context.Background(), "old"),
			val:  "new",
			want: "new",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCtx := corr.SetID(tt.ctx, tt.val)
			if got := corr.ID(gotCtx); got != tt.want {
				t.Errorf("SetID() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIdempotencyKey(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			name: "not found",
			ctx:  context.Background(),
			want: "",
		},
		{
			name: "found",
			ctx:  corr.SetIdempotencyKey(context.Background(), "key1"),
			want: "key1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := corr.IdempotencyKey(tt.ctx); got != tt.want {
				t.Errorf("Idempotency() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetIdempotencyKey(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		val  string
		want string
	}{
		{
			name: "set",
			ctx:  context.Background(),
			val:  "key1",
			want: "key1",
		},
		{
			name: "update",
			ctx:  corr.SetIdempotencyKey(context.Background(), "old"),
			val:  "new",
			want: "new",
		},
		{
			name: "set empty",
			ctx:  context.Background(),
			val:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCtx := corr.SetIdempotencyKey(tt.ctx, tt.val)
			if got := corr.IdempotencyKey(gotCtx); got != tt.want {
				t.Errorf("SetIdempotency() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeys(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want []string
	}{
		{
			name: "empty",
			ctx:  context.Background(),
			want: nil,
		},
		{
			name: "with keys",
			ctx:  corr.Set(context.Background(), map[string]string{"key1": "val1", "key2": "val2"}),
			want: []string{"key1", "key2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := corr.Keys(tt.ctx)
			sort.Strings(got)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Keys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{
			name: "empty",
			ctx:  context.Background(),
			want: true,
		},
		{
			name: "not empty",
			ctx:  corr.Set(context.Background(), map[string]string{"key": "val"}),
			want: false,
		},
		{
			name: "nil ctx",
			ctx:  nil,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := corr.IsEmpty(tt.ctx); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			name: "empty",
			ctx:  context.Background(),
			want: "",
		},
		{
			name: "single",
			ctx:  corr.Set(context.Background(), map[string]string{"key1": "val1"}),
			want: "key1=val1",
		},
		{
			name: "multiple",
			ctx:  corr.Set(context.Background(), map[string]string{"key1": "val1", "key2": "val2"}),
			want: "key1=val1,key2=val2", // Order may vary, but we can sort for comparison
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := corr.String(tt.ctx)
			if tt.want == "" {
				if got != "" {
					t.Errorf("String() = %v, want %v", got, tt.want)
				}
				return
			}
			gotParts := strings.Split(got, ",")
			wantParts := strings.Split(tt.want, ",")
			sort.Strings(gotParts)
			sort.Strings(wantParts)
			if !reflect.DeepEqual(gotParts, wantParts) {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			name: "empty",
			ctx:  context.Background(),
			want: "",
		},
		{
			name: "single",
			ctx:  corr.Set(context.Background(), map[string]string{"key1": "val1"}),
			want: "key1=val1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := corr.Generate(tt.ctx); got != tt.want {
				t.Errorf("Generate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCorrelationHeader(t *testing.T) {
	tests := []struct {
		name      string
		headerVal string
		want      corr.Data
	}{
		{
			name:      "empty",
			headerVal: "",
			want:      corr.Data{},
		},
		{
			name:      "single",
			headerVal: "key1=val1",
			want:      corr.Data{"key1": "val1"},
		},
		{
			name:      "multiple",
			headerVal: "key1=val1,key2=val2",
			want:      corr.Data{"key1": "val1", "key2": "val2"},
		},
		{
			name:      "with spaces",
			headerVal: " key1 = val1 , key2 = val2 ",
			want:      corr.Data{"key1": "val1", "key2": "val2"},
		},
		{
			name:      "invalid parts",
			headerVal: "key1,=val1,key2=val2=extra",
			want:      corr.Data{"key2": "val2=extra"},
		},
		{
			name:      "empty pairs",
			headerVal: ",,key= , =val,",
			want:      corr.Data{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := corr.ParseCorrelationHeader(tt.headerVal)
			if err != nil {
				t.Errorf("ParseCorrelationHeader() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCorrelationHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFirst(t *testing.T) {
	tests := []struct {
		name string
		md   metadata.MD
		key  string
		want string
	}{
		{
			name: "not present",
			md:   metadata.MD{},
			key:  "key",
			want: "",
		},
		{
			name: "empty slice",
			md:   metadata.MD{"key": []string{}},
			key:  "key",
			want: "",
		},
		{
			name: "single value",
			md:   metadata.MD{"key": []string{"val"}},
			key:  "key",
			want: "val",
		},
		{
			name: "multiple take first",
			md:   metadata.MD{"key": []string{"val1", "val2"}},
			key:  "key",
			want: "val1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := meta.GetFirst(tt.md, tt.key); got != tt.want {
				t.Errorf("GetFirst() = %v, want %v", got, tt.want)
			}
		})
	}
}

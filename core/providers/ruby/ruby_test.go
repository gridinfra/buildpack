package ruby

import (
	"testing"

	"github.com/stretchr/testify/require"

	testingUtils "github.com/railwayapp/railpack/core/testing"
)

func TestMetadata(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		runtime string
		expose  string
	}{
		{name: "rails", path: "../../../examples/ruby-rails-api-app", runtime: "rails", expose: "3000"},
		{name: "rack", path: "../../../examples/ruby-sinatra", runtime: "ruby", expose: "3000"},
		{name: "ruby", path: "../../../examples/ruby-vanilla", runtime: "ruby"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testingUtils.CreateGenerateContext(t, tt.path)
			metadata := (&RubyProvider{}).Metadata(ctx)
			require.Equal(t, tt.runtime, metadata.Runtime)
			require.Equal(t, tt.expose, metadata.Expose)
		})
	}
}

func TestDetect(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "ruby",
			path: "../../../examples/ruby-vanilla",
			want: true,
		},
		{
			name: "no ruby",
			path: "../../../examples/go-mod",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testingUtils.CreateGenerateContext(t, tt.path)
			provider := RubyProvider{}
			got, err := provider.Detect(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

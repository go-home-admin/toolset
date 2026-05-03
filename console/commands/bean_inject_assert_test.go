package commands

import (
	"testing"

	"github.com/go-home-admin/toolset/parser"
)

func TestIfaceAssertParenBody_flatConfigValueVsPointer(t *testing.T) {
	redisPath := "example.com/lib/redis"

	tests := []struct {
		name string
		attr parser.GoTypeAttr
		m    map[string]string
		want string
	}{
		{
			name: "value_int_asserts_star_int",
			attr: parser.GoTypeAttr{TypeName: "int", InPackage: true},
			m:    nil,
			want: "*int",
		},
		{
			name: "pointer_int_asserts_star_int",
			attr: parser.GoTypeAttr{TypeName: "*int", InPackage: true},
			m:    nil,
			want: "*int",
		},
		{
			name: "pointer_double_star_int",
			attr: parser.GoTypeAttr{TypeName: "**int", InPackage: true},
			m:    nil,
			want: "**int",
		},
		{
			name: "value_string",
			attr: parser.GoTypeAttr{TypeName: "string", InPackage: true},
			m:    nil,
			want: "*string",
		},
		{
			name: "pointer_string",
			attr: parser.GoTypeAttr{TypeName: "*string", InPackage: true},
			m:    nil,
			want: "*string",
		},
		{
			name: "value_import_selector_rewrites_alias",
			attr: parser.GoTypeAttr{
				TypeName:   "redis.Client",
				TypeAlias:  "redis",
				TypeImport: redisPath,
				InPackage:  false,
			},
			m:    map[string]string{redisPath: "redis"},
			want: "*redis.Client",
		},
		{
			name: "pointer_import_selector_star_redis_Client",
			attr: parser.GoTypeAttr{
				TypeName:   "*redis.Client",
				TypeAlias:  "redis",
				TypeImport: redisPath,
				InPackage:  false,
			},
			m:    map[string]string{redisPath: "redis"},
			want: "*redis.Client",
		},
		{
			name: "import_fallback_uses_TypeAlias_when_map_misses",
			attr: parser.GoTypeAttr{
				TypeName:   "*redis.Client",
				TypeAlias:  "redis",
				TypeImport: redisPath,
				InPackage:  false,
			},
			m:    map[string]string{},
			want: "*redis.Client",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.m == nil {
				tc.m = map[string]string{}
			}
			got := ifaceAssertParenBody(tc.attr, tc.m)
			if got != tc.want {
				t.Fatalf("ifaceAssertParenBody() = %q, want %q", got, tc.want)
			}
		})
	}
}

package acl_test

import (
	"testing"

	"github.com/plgd-dev/device/schema/acl"
	"github.com/stretchr/testify/require"
)

func TestPermissionString(t *testing.T) {
	tests := []struct {
		name string
		s    acl.Permission
		want string
	}{
		{
			name: "Empty",
			s:    0,
			want: "",
		},
		{
			name: "Unknown",
			s:    acl.Permission_NOTIFY * 2, // double of the last acl.Permission value
			want: "unknown(32)",
		},
		{
			name: "Single",
			s:    acl.Permission_CREATE,
			want: "CREATE",
		},
		{
			name: "All",
			s:    acl.AllPermissions,
			want: "CREATE|READ|WRITE|DELETE|NOTIFY",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.String()
			require.Equal(t, tt.want, got)
		})
	}
}

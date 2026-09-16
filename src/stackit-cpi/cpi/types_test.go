package cpi

import (
	"encoding/json"
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

func TestRPCContext_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "valid context with array security groups",
			input: `{
				"director_uuid": "12345",
				"request_id": "req-123",
				"region": "eu01"
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx lib.RPCContext
			err := json.Unmarshal([]byte(tt.input), &ctx)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

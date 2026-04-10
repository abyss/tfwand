package workflow

import (
	"testing"
)

func TestParsePlanOutput(t *testing.T) {
	tests := []struct {
		name            string
		output          string
		wantStatus      string
		wantSummaryHas  string // non-empty means summary should contain this
		wantSummaryNone bool   // true means summary should be empty
	}{
		{
			name: "no changes",
			output: `Terraform will perform the following actions:

No changes. Infrastructure is up-to-date.

Plan: 0 to add, 0 to change, 0 to destroy.`,
			wantStatus:      "no-change",
			wantSummaryHas:  "",
			wantSummaryNone: true,
		},
		{
			name: "resources to add",
			output: `Terraform will perform the following actions:

  # aws_instance.example will be created

Plan: 1 to add, 0 to change, 0 to destroy.`,
			wantStatus:     "changed",
			wantSummaryHas: "Plan: 1 to add, 0 to change, 0 to destroy.",
		},
		{
			name: "resources to change",
			output: `Terraform will perform the following actions:

  # aws_instance.example will be updated in-place

Plan: 0 to add, 2 to change, 0 to destroy.`,
			wantStatus:     "changed",
			wantSummaryHas: "Plan: 0 to add, 2 to change, 0 to destroy.",
		},
		{
			name: "resources to destroy",
			output: `Terraform will perform the following actions:

  # aws_instance.example will be destroyed

Plan: 0 to add, 0 to change, 3 to destroy.`,
			wantStatus:     "changed",
			wantSummaryHas: "Plan: 0 to add, 0 to change, 3 to destroy.",
		},
		{
			name:           "mixed add change destroy",
			output:         `Plan: 2 to add, 1 to change, 1 to destroy.`,
			wantStatus:     "changed",
			wantSummaryHas: "Plan: 2 to add, 1 to change, 1 to destroy.",
		},
		{
			name: "output-only changes",
			output: `Changes to outputs:
  + foo = "bar"

You can apply this plan to save these new output values to the Terraform
state, without changing any real infrastructure.`,
			wantStatus:      "output-only",
			wantSummaryNone: true,
		},
		{
			name:            "empty output",
			output:          "",
			wantStatus:      "no-change",
			wantSummaryNone: true,
		},
		{
			name: "OpenTofu phrasing",
			output: `Plan: 0 to add, 0 to change, 0 to destroy.

Changes to outputs:
  + result = "hello"`,
			wantStatus:      "output-only",
			wantSummaryNone: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, status := parsePlanOutput([]byte(tt.output))

			if status != tt.wantStatus {
				t.Errorf("status = %q, want %q", status, tt.wantStatus)
			}
			if tt.wantSummaryNone && summary != "" {
				t.Errorf("expected empty summary, got %q", summary)
			}
			if tt.wantSummaryHas != "" && summary != tt.wantSummaryHas {
				t.Errorf("summary = %q, want %q", summary, tt.wantSummaryHas)
			}
		})
	}
}

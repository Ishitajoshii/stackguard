package cloud

import (
	"testing"
)

func TestCloudPlanRequiresExplicitRoleModel(t *testing.T) {
	if _, e := Plan("lab", "", "unknown"); e == nil {
		t.Fatal("invented role")
	}
	p, e := Plan("StackGuard", "arn:aws:iam::000000000000:role/lab", "explicit-model")
	if e != nil || p["maxIterations"] != 6 || p["maxTokens"] != 2048 {
		t.Fatal(p, e)
	}
	for _, tool := range p["allowedTools"].([]string) {
		if tool == "*" || tool == "shell" {
			t.Fatal("unexpected access")
		}
	}
}

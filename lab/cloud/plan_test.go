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
	lifecycle := p["environment"].(map[string]any)["agentCoreRuntimeEnvironment"].(map[string]any)["lifecycleConfiguration"].(map[string]int)
	if lifecycle["idleRuntimeSessionTimeout"] != 60 || lifecycle["maxLifetime"] != 300 {
		t.Fatal("runtime must not retain the service's 15-minute idle or eight-hour lifetime defaults")
	}
}

package research

import (
	"strings"
	"testing"
	"time"
)

func fixture() FrozenManifest {
	h := strings.Repeat("a", 64)
	return FrozenManifest{
		ProjectID: "project-001", ResearchProgramID: "program-001", HypothesisID: "hypothesis-001",
		CycleID: "cycle-001", ExperimentID: "experiment-001", SelectionFamilyID: "family-001",
		StrategySpec: StrategySpec{
			Version: StrategySpecVersion, Environment: SyntheticEnvironment, Instrument: "BTC-USD",
			TimeframeSeconds: 3600, Signal: ComponentRef{SHA256: h}, Entry: ComponentRef{SHA256: h},
			Exit: ComponentRef{SHA256: h}, Sizing: ComponentRef{SHA256: h}, Risk: ComponentRef{SHA256: h},
		},
		KernelSHA256: h, CodeSHA256: h, DatasetSHA256: h, PartitionSHA256: h,
		BenchmarkSHA256: h, ProtocolSHA256: h, CostSHA256: h, MetricsSHA256: h, RequiredGatesSHA256: h,
		DataOrigin: SyntheticEnvironment, PartitionRole: DevelopmentPartition,
		Seed: 42, FrozenAt: time.Date(2026, 10, 21, 12, 34, 56, 123000000, time.UTC),
	}
}

func TestFreezeVerifyAndGoldenIdentity(t *testing.T) {
	frozen, err := Freeze(fixture())
	if err != nil {
		t.Fatalf("Freeze() error = %v", err)
	}
	if frozen.SchemaVersion != FreezeSchemaVersion || frozen.CanonicalizationVersion != CanonicalizationVersion {
		t.Fatalf("unsupported frozen versions: %q / %q", frozen.SchemaVersion, frozen.CanonicalizationVersion)
	}
	if !validSHA256(frozen.StrategySpecHash) || !validSHA256(frozen.IdentityHash) {
		t.Fatal("Freeze did not produce SHA-256 identities")
	}
	if err := Verify(frozen); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if frozen.IdentityHash != "92cc31a04b9290993141084d570652e365992099b9feb9e896afa87ddfc9fa51" {
		t.Fatalf("golden digest changed: %s", frozen.IdentityHash)
	}
}

func TestFreezeIsDeterministicForSameTypedValues(t *testing.T) {
	first, err := Freeze(fixture())
	if err != nil {
		t.Fatal(err)
	}
	second, err := Freeze(fixture())
	if err != nil {
		t.Fatal(err)
	}
	if first.StrategySpecHash != second.StrategySpecHash || first.IdentityHash != second.IdentityHash {
		t.Fatal("identical typed input produced different identity hashes")
	}
}

func TestFreezeCopiesInputValues(t *testing.T) {
	input := fixture()
	frozen, err := Freeze(input)
	if err != nil {
		t.Fatal(err)
	}
	input.StrategySpec.Instrument = "ETH-USD"
	input.BenchmarkSHA256 = strings.Repeat("b", 64)
	if err := Verify(frozen); err != nil {
		t.Fatalf("mutating caller input changed frozen identity: %v", err)
	}
}

func TestVerifyRejectsIdentityMutation(t *testing.T) {
	base, err := Freeze(fixture())
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*FrozenManifest){
		"project id":     func(m *FrozenManifest) { m.ProjectID = "project-002" },
		"spec component": func(m *FrozenManifest) { m.StrategySpec.Signal.SHA256 = strings.Repeat("b", 64) },
		"seed":           func(m *FrozenManifest) { m.Seed++ },
		"benchmark":      func(m *FrozenManifest) { m.BenchmarkSHA256 = strings.Repeat("b", 64) },
		"protocol":       func(m *FrozenManifest) { m.ProtocolSHA256 = strings.Repeat("b", 64) },
		"frozen clock":   func(m *FrozenManifest) { m.FrozenAt = m.FrozenAt.Add(time.Second) },
		"manifest hash":  func(m *FrozenManifest) { m.IdentityHash = strings.Repeat("b", 64) },
		"spec hash":      func(m *FrozenManifest) { m.StrategySpecHash = strings.Repeat("b", 64) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			changed := base
			mutate(&changed)
			if err := Verify(changed); err == nil {
				t.Fatal("Verify accepted a changed identity")
			}
		})
	}
}

func TestFreezeRejectsMissingIdentityAndUnsupportedOrUnverifiableInputs(t *testing.T) {
	cases := map[string]func(*FrozenManifest){
		"missing project":              func(m *FrozenManifest) { m.ProjectID = "" },
		"short id":                     func(m *FrozenManifest) { m.ProjectID = "p1" },
		"unknown id syntax":            func(m *FrozenManifest) { m.ExperimentID = "../run" },
		"missing research program":     func(m *FrozenManifest) { m.ResearchProgramID = "" },
		"missing hypothesis":           func(m *FrozenManifest) { m.HypothesisID = "" },
		"missing cycle":                func(m *FrozenManifest) { m.CycleID = "" },
		"missing experiment":           func(m *FrozenManifest) { m.ExperimentID = "" },
		"missing selection family":     func(m *FrozenManifest) { m.SelectionFamilyID = "" },
		"unsupported schema":           func(m *FrozenManifest) { m.SchemaVersion = "future/v2" },
		"unsupported canonicalization": func(m *FrozenManifest) { m.CanonicalizationVersion = "rfc8785" },
		"real data":                    func(m *FrozenManifest) { m.DataOrigin = "market-feed" },
		"holdout":                      func(m *FrozenManifest) { m.PartitionRole = "holdout" },
		"prospective":                  func(m *FrozenManifest) { m.PartitionRole = "prospective" },
		"zero dataset hash":            func(m *FrozenManifest) { m.DatasetSHA256 = strings.Repeat("0", 64) },
		"short benchmark hash":         func(m *FrozenManifest) { m.BenchmarkSHA256 = "abc" },
		"uppercase code hash":          func(m *FrozenManifest) { m.CodeSHA256 = strings.Repeat("A", 64) },
		"missing kernel hash":          func(m *FrozenManifest) { m.KernelSHA256 = "" },
		"missing protocol hash":        func(m *FrozenManifest) { m.ProtocolSHA256 = "" },
		"zero time":                    func(m *FrozenManifest) { m.FrozenAt = time.Time{} },
		"non-UTC time": func(m *FrozenManifest) {
			m.FrozenAt = time.Date(2026, 10, 21, 12, 0, 0, 0, time.FixedZone("not-utc", 3600))
		},
		"out of range time":            func(m *FrozenManifest) { m.FrozenAt = time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC) },
		"unsupported strategy version": func(m *FrozenManifest) { m.StrategySpec.Version = "future/v2" },
		"non-synthetic strategy":       func(m *FrozenManifest) { m.StrategySpec.Environment = "live" },
		"missing instrument":           func(m *FrozenManifest) { m.StrategySpec.Instrument = " " },
		"invalid UTF-8 instrument":     func(m *FrozenManifest) { m.StrategySpec.Instrument = string([]byte{0xff, 0xfe}) },
		"invalid timeframe":            func(m *FrozenManifest) { m.StrategySpec.TimeframeSeconds = 0 },
		"missing component":            func(m *FrozenManifest) { m.StrategySpec.Risk.SHA256 = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			input := fixture()
			mutate(&input)
			if _, err := Freeze(input); err == nil {
				t.Fatal("Freeze accepted invalid input")
			}
		})
	}
}

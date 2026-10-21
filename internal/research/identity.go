// Package research contains typed, local-only research identity contracts.
// A verified identity proves byte integrity only; it does not prove that a
// research plan, dataset, benchmark, or scientific result is valid.
package research

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	StrategySpecVersion     = "asteriade/strategy-spec-v1"
	FreezeSchemaVersion     = "asteriade/frozen-manifest-v1"
	CanonicalizationVersion = "asteriade/freeze-json-v1"
	SyntheticEnvironment    = "synthetic"
	DevelopmentPartition    = "development"
)

var ErrInvalidIdentity = errors.New("invalid research identity")

// ComponentRef identifies a component by immutable content hash. This envelope
// records component identity; it does not parse or execute component code.
type ComponentRef struct {
	SHA256 string `json:"sha256"`
}

// StrategySpec is the Stage 0A envelope, not a complete strategy language.
// Its component references are never interpreted or executed here.
type StrategySpec struct {
	Version          string       `json:"version"`
	Environment      string       `json:"environment"`
	Instrument       string       `json:"instrument"`
	TimeframeSeconds int64        `json:"timeframe_seconds"`
	Signal           ComponentRef `json:"signal"`
	Entry            ComponentRef `json:"entry"`
	Exit             ComponentRef `json:"exit"`
	Sizing           ComponentRef `json:"sizing"`
	Risk             ComponentRef `json:"risk"`
}

// FrozenManifest fixes the typed inputs that identify one synthetic
// development experiment. IdentityHash and SpecHash are lowercase SHA-256 hex.
type FrozenManifest struct {
	SchemaVersion           string       `json:"schema_version"`
	CanonicalizationVersion string       `json:"canonicalization_version"`
	ProjectID               string       `json:"project_id"`
	ResearchProgramID       string       `json:"research_program_id"`
	HypothesisID            string       `json:"hypothesis_id"`
	CycleID                 string       `json:"cycle_id"`
	ExperimentID            string       `json:"experiment_id"`
	SelectionFamilyID       string       `json:"selection_family_id"`
	StrategySpec            StrategySpec `json:"strategy_spec"`
	StrategySpecHash        string       `json:"strategy_spec_hash"`
	KernelSHA256            string       `json:"kernel_sha256"`
	CodeSHA256              string       `json:"code_sha256"`
	DatasetSHA256           string       `json:"dataset_sha256"`
	PartitionSHA256         string       `json:"partition_sha256"`
	BenchmarkSHA256         string       `json:"benchmark_sha256"`
	ProtocolSHA256          string       `json:"protocol_sha256"`
	CostSHA256              string       `json:"cost_sha256"`
	MetricsSHA256           string       `json:"metrics_sha256"`
	RequiredGatesSHA256     string       `json:"required_gates_sha256"`
	DataOrigin              string       `json:"data_origin"`
	PartitionRole           string       `json:"partition_role"`
	Seed                    uint64       `json:"seed"`
	FrozenAt                time.Time    `json:"frozen_at"`
	IdentityHash            string       `json:"identity_hash"`
}

// Freeze validates and copies the typed inputs, computes the spec hash, then
// computes the manifest identity hash using fixed-field-order encoding/json.
// No arbitrary JSON, maps, slices, floats, or executable expressions are used.
// These hashes are not signatures, authentication, or clock attestations, and
// they cannot prevent tampering by a caller able to change and rehash inputs.
func Freeze(inputs FrozenManifest) (FrozenManifest, error) {
	if inputs.SchemaVersion != "" && inputs.SchemaVersion != FreezeSchemaVersion ||
		inputs.CanonicalizationVersion != "" && inputs.CanonicalizationVersion != CanonicalizationVersion {
		return FrozenManifest{}, fmt.Errorf("%w: unsupported schema or canonicalization version", ErrInvalidIdentity)
	}
	inputs.SchemaVersion = FreezeSchemaVersion
	inputs.CanonicalizationVersion = CanonicalizationVersion
	inputs.IdentityHash = ""
	if err := validateManifestFields(inputs, false); err != nil {
		return FrozenManifest{}, err
	}
	specHash, err := hashValue(inputs.StrategySpec)
	if err != nil {
		return FrozenManifest{}, fmt.Errorf("%w: encode StrategySpec: %v", ErrInvalidIdentity, err)
	}
	inputs.StrategySpecHash = specHash
	identityHash, err := manifestHash(inputs)
	if err != nil {
		return FrozenManifest{}, fmt.Errorf("%w: encode manifest: %v", ErrInvalidIdentity, err)
	}
	inputs.IdentityHash = identityHash
	return inputs, nil
}

// Verify recomputes both hashes and checks the supported synthetic-only
// envelope. Success establishes integrity, not gate completion or validity.
func Verify(frozen FrozenManifest) error {
	if err := validateManifestFields(frozen, true); err != nil {
		return err
	}
	specHash, err := hashValue(frozen.StrategySpec)
	if err != nil || specHash != frozen.StrategySpecHash {
		return fmt.Errorf("%w: StrategySpec hash mismatch", ErrInvalidIdentity)
	}
	actual, err := manifestHash(frozen)
	if err != nil || actual != frozen.IdentityHash {
		return fmt.Errorf("%w: manifest identity hash mismatch", ErrInvalidIdentity)
	}
	return nil
}

func validateManifestFields(m FrozenManifest, requireHashes bool) error {
	if m.SchemaVersion != FreezeSchemaVersion || m.CanonicalizationVersion != CanonicalizationVersion {
		return fmt.Errorf("%w: unsupported schema or canonicalization version", ErrInvalidIdentity)
	}
	if !validID(m.ProjectID) || !validID(m.ResearchProgramID) || !validID(m.HypothesisID) ||
		!validID(m.CycleID) || !validID(m.ExperimentID) || !validID(m.SelectionFamilyID) {
		return fmt.Errorf("%w: missing or out-of-range identity ID", ErrInvalidIdentity)
	}
	if err := validateStrategySpec(m.StrategySpec); err != nil {
		return err
	}
	if m.DataOrigin != SyntheticEnvironment || m.PartitionRole != DevelopmentPartition {
		return fmt.Errorf("%w: only synthetic development data is permitted", ErrInvalidIdentity)
	}
	if !validSHA256(m.KernelSHA256) || !validSHA256(m.CodeSHA256) || !validSHA256(m.DatasetSHA256) ||
		!validSHA256(m.PartitionSHA256) || !validSHA256(m.BenchmarkSHA256) || !validSHA256(m.ProtocolSHA256) || !validSHA256(m.CostSHA256) ||
		!validSHA256(m.MetricsSHA256) || !validSHA256(m.RequiredGatesSHA256) {
		return fmt.Errorf("%w: invalid SHA-256 reference", ErrInvalidIdentity)
	}
	if m.FrozenAt.IsZero() || m.FrozenAt.Location() != time.UTC || m.FrozenAt.Year() < 2000 || m.FrozenAt.Year() > 2100 {
		return fmt.Errorf("%w: frozen time must be a bounded UTC timestamp", ErrInvalidIdentity)
	}
	if requireHashes && (!validSHA256(m.StrategySpecHash) || !validSHA256(m.IdentityHash)) {
		return fmt.Errorf("%w: missing or invalid frozen hash", ErrInvalidIdentity)
	}
	return nil
}

func validateStrategySpec(s StrategySpec) error {
	if s.Version != StrategySpecVersion || s.Environment != SyntheticEnvironment ||
		!validLabel(s.Instrument, 128) || s.TimeframeSeconds <= 0 {
		return fmt.Errorf("%w: unsupported or incomplete StrategySpec envelope", ErrInvalidIdentity)
	}
	if !validSHA256(s.Signal.SHA256) || !validSHA256(s.Entry.SHA256) || !validSHA256(s.Exit.SHA256) ||
		!validSHA256(s.Sizing.SHA256) || !validSHA256(s.Risk.SHA256) {
		return fmt.Errorf("%w: invalid StrategySpec component SHA-256 reference", ErrInvalidIdentity)
	}
	return nil
}

func manifestHash(m FrozenManifest) (string, error) {
	// Canonical bytes are json.Marshal(FrozenManifest) in the struct's declared
	// field order, with identity_hash set to the empty string. The StrategySpec
	// digest uses json.Marshal(StrategySpec) in its declared field order. Go's
	// encoding/json defaults (including HTML escaping and RFC3339Nano time output)
	// are part of freeze-json-v1; this is not RFC 8785.
	m.IdentityHash = ""
	return hashValue(m)
}

func hashValue[T any](value T) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 || value == strings.Repeat("0", sha256.Size*2) {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func validID(value string) bool {
	if len(value) < 3 || len(value) > 128 || !isASCIIAlphaNumeric(value[0]) {
		return false
	}
	for i := 1; i < len(value); i++ {
		c := value[i]
		if !isASCIIAlphaNumeric(c) && c != '-' && c != '_' && c != '.' && c != ':' {
			return false
		}
	}
	return value != "." && value != ".."
}

func validLabel(value string, max int) bool {
	if !utf8.ValidString(value) || len(value) == 0 || len(value) > max || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func isASCIIAlphaNumeric(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

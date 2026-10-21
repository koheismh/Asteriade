package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidOperation = errors.New("invalid operation")
	ErrInvalidApproval  = errors.New("invalid approval")
)

var operationKinds = map[string]bool{
	"workspace.read": true, "workspace.write": true, "process.exec": true,
	"network.read": true, "external.send": true, "git.push": true,
}

func OperationHash(op Operation) (string, error) {
	if err := ValidateOperation(op); err != nil {
		return "", err
	}
	b, err := json.Marshal(op)
	if err != nil {
		return "", fmt.Errorf("marshal operation: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func ValidateOperation(op Operation) error {
	if !operationKinds[op.Kind] || strings.TrimSpace(op.Destination) == "" || strings.TrimSpace(op.Account) == "" ||
		strings.TrimSpace(op.Target) == "" || op.PolicyRevision <= 0 || op.EstimatedCostMicroJPY < 0 || !validSHA256(op.PayloadHash) {
		return ErrInvalidOperation
	}
	for _, s := range []string{op.Destination, op.Account, op.Target} {
		if strings.TrimSpace(s) != s || hasControl(s) {
			return ErrInvalidOperation
		}
	}
	// Only explicitly network-facing operations may name a URL. Require HTTPS and a host.
	parsed, err := url.Parse(op.Destination)
	if err != nil {
		return ErrInvalidOperation
	}
	hasScheme := parsed.Scheme != ""
	if hasScheme {
		if (op.Kind != "network.read" && op.Kind != "external.send") || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
			return ErrInvalidOperation
		}
	}
	if (op.Kind == "network.read" || op.Kind == "external.send") && (!hasScheme || parsed.Scheme != "https" || parsed.Host == "") {
		return ErrInvalidOperation
	}
	return nil
}

func validSHA256(s string) bool {
	if len(s) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func hasControl(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

func ValidateApproval(approval Approval, op Operation, now time.Time) error {
	if err := ValidateOperation(op); err != nil {
		return ErrInvalidApproval
	}
	h, err := OperationHash(op)
	if err != nil || h != approval.OperationHash || approval.Operation != op || approval.ID == "" || approval.ProjectID == "" ||
		approval.Version <= 0 || approval.Status != "granted" || approval.Scope != "once" || approval.ExpiresAt.IsZero() || !now.Before(approval.ExpiresAt) {
		return ErrInvalidApproval
	}
	return nil
}

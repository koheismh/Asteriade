package core

import (
	"strings"
	"testing"
	"time"
)

func validOp() Operation {
	return Operation{Kind: "external.send", Destination: "https://example.org/api", Account: "acct-1", Target: "post:42", PayloadHash: strings.Repeat("a", 64), PolicyRevision: 1, EstimatedCostMicroJPY: 0}
}

func TestOperationHashIsDeterministicAndCoversEveryField(t *testing.T) {
	op := validOp()
	first, err := OperationHash(op)
	if err != nil {
		t.Fatal(err)
	}
	second, err := OperationHash(op)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("hash changed for identical operation")
	}
	mutations := []func(*Operation){
		func(o *Operation) { o.Kind = "network.read" }, func(o *Operation) { o.Destination = "https://other.example" },
		func(o *Operation) { o.Account = "other" }, func(o *Operation) { o.Target = "other" },
		func(o *Operation) { o.PayloadHash = strings.Repeat("b", 64) }, func(o *Operation) { o.PolicyRevision++ },
		func(o *Operation) { o.EstimatedCostMicroJPY++ },
	}
	for i, mutate := range mutations {
		changed := op
		mutate(&changed)
		h, e := OperationHash(changed)
		if e != nil {
			t.Fatalf("mutation %d unexpectedly invalid: %v", i, e)
		}
		if h == first {
			t.Errorf("field mutation %d did not change hash", i)
		}
	}
}

func TestValidateOperationRejectsUnsafeOrMalformedInput(t *testing.T) {
	for name, mutate := range map[string]func(*Operation){
		"unknown kind": func(o *Operation) { o.Kind = "shell" }, "empty account": func(o *Operation) { o.Account = " " },
		"empty target": func(o *Operation) { o.Target = "" }, "empty destination": func(o *Operation) { o.Destination = "" },
		"negative cost": func(o *Operation) { o.EstimatedCostMicroJPY = -1 }, "revision zero": func(o *Operation) { o.PolicyRevision = 0 },
		"malformed hash": func(o *Operation) { o.PayloadHash = "123" }, "http url": func(o *Operation) { o.Destination = "http://example.org" },
		"url credentials":     func(o *Operation) { o.Destination = "https://user:secret@example.org" },
		"url on workspace op": func(o *Operation) { o.Kind = "workspace.write" },
		"control char":        func(o *Operation) { o.Target = "ok\nforged" },
	} {
		t.Run(name, func(t *testing.T) {
			op := validOp()
			mutate(&op)
			if ValidateOperation(op) == nil {
				t.Fatal("expected rejection")
			}
		})
	}
	for _, kind := range []string{"workspace.read", "workspace.write", "process.exec", "git.push"} {
		op := validOp()
		op.Kind = kind
		op.Destination = "repo/main"
		if err := ValidateOperation(op); err != nil {
			t.Errorf("valid %s rejected: %v", kind, err)
		}
	}
	for _, kind := range []string{"network.read", "external.send"} {
		op := validOp()
		op.Kind = kind
		if err := ValidateOperation(op); err != nil {
			t.Errorf("valid %s rejected: %v", kind, err)
		}
	}
}

func TestValidateApprovalRechecksScopeStatusExpiryAndOperation(t *testing.T) {
	op := validOp()
	hash, err := OperationHash(op)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	approval := Approval{ID: "approval-1", ProjectID: "project-1", OperationHash: hash, Operation: op, Scope: "once", Status: "granted", Version: 2, ExpiresAt: now.Add(time.Minute)}
	if err := ValidateApproval(approval, op, now); err != nil {
		t.Fatalf("valid approval rejected: %v", err)
	}
	for name, mutate := range map[string]func(*Approval){
		"hash tamper":                 func(a *Approval) { a.OperationHash = strings.Repeat("0", 64) },
		"embedded operation mismatch": func(a *Approval) { a.Operation.Target = "other" },
		"wrong scope":                 func(a *Approval) { a.Scope = "persistent" }, "not granted": func(a *Approval) { a.Status = "pending" },
		"no version": func(a *Approval) { a.Version = 0 }, "no project": func(a *Approval) { a.ProjectID = "" },
	} {
		t.Run(name, func(t *testing.T) {
			bad := approval
			mutate(&bad)
			if ValidateApproval(bad, op, now) == nil {
				t.Fatal("expected rejection")
			}
		})
	}
	if ValidateApproval(approval, op, approval.ExpiresAt) == nil {
		t.Fatal("approval accepted at expiry")
	}
	changed := op
	changed.Target = "different"
	if ValidateApproval(approval, changed, now) == nil {
		t.Fatal("approval accepted changed operation")
	}
}

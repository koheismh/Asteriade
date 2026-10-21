// Package runner exposes the deliberately unavailable Stage 0 local runner.
package runner

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/koheismh/Asteriade/internal/core"
)

var ErrUnverified = errors.New("runner safety is unverified")

func Describe() core.RunnerProfile {
	return core.RunnerProfile{
		OS:     runtime.GOOS,
		Status: "unverified",
		Capabilities: map[string]bool{
			"workspace.read":  false,
			"workspace.write": false,
			"process.exec":    false,
			"network.read":    false,
			"external.send":   false,
			"git.push":        false,
		},
		Reason: fmt.Sprintf("%s runner isolation, filesystem/network restrictions, and descendant-process controls have not passed the required safety validation", runtime.GOOS),
	}
}

func CheckReady() error { return ErrUnverified }

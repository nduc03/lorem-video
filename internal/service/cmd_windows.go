//go:build windows

package service

import (
	"context"
	"os/exec"
)

func createFFmpegCmd(ctx context.Context, args []string) *exec.Cmd {
	// On Windows, the "nice" command and Setpgid are not available.
	// We run ffmpeg directly.
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	
	return cmd
}

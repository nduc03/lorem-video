//go:build unix

package service

import (
	"context"
	"os/exec"
	"syscall"
)

func createFFmpegCmd(ctx context.Context, args []string) *exec.Cmd {
	// Use nice to lower process priority for background video generation
	niceArgs := append([]string{"-n", "10", "ffmpeg"}, args...)
	cmd := exec.CommandContext(ctx, "nice", niceArgs...)
	
	// Add resource limits for VPS environments
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group for better cleanup
	}
	
	return cmd
}

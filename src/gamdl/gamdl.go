// TODO: add downloading album / song

package gamdl

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type Runner struct {
	base string
	pre  []string
}

type GamDL struct {
	Runner     *Runner
	CookiePath string
}

func NewGamDL(cookiePath string) (*GamDL, error) {
	if cookiePath == "" {
		return nil, errors.New("cookie path is required")
	}

	info, err := os.Stat(cookiePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cookie path %q: %w", cookiePath, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("cookie path %q is a directory", cookiePath)
	}

	runner, err := NewRunner()
	if err != nil {
		return nil, err
	}

	return &GamDL{
		Runner:     runner,
		CookiePath: cookiePath,
	}, nil
}

func NewRunner() (*Runner, error) {
	if p, err := exec.LookPath("gamdl"); err == nil {
		return &Runner{base: p}, nil
	}
	if _, err := exec.LookPath("uv"); err == nil {
		return &Runner{base: "uv", pre: []string{"tool", "run", "gamdl"}}, nil
	}
	return nil, errors.New("gamdl not found (install gamdl or uv)")
}

func (r *Runner) Command(ctx context.Context, args ...string) *exec.Cmd {
	all := append(append([]string{}, r.pre...), args...)
	cmd := exec.CommandContext(ctx, r.base, all...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd
}

func (g *GamDL) Command(ctx context.Context, args ...string) (*exec.Cmd, error) {
	if g == nil || g.Runner == nil {
		return nil, errors.New("gamdl runner is not initialized")
	}

	gamdlArgs := []string{"--cookie-path", g.CookiePath}
	gamdlArgs = append(gamdlArgs, args...)

	return g.Runner.Command(ctx, gamdlArgs...), nil
}

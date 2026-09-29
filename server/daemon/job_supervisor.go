package daemon

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/sebday/evoplayer/server/download"
	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/worker"
)

const workerKillGrace = 3 * time.Second

func (d *Daemon) runSoundCloudDownloadJob(ctx context.Context, importAfter bool) error {
	args := []string{"soundcloud-download"}
	if importAfter {
		args = append(args, "--import")
	}
	return d.runWorkerJob(ctx, args...)
}

func (d *Daemon) runImportJob(ctx context.Context) error {
	return d.runWorkerJob(ctx, "import-incoming")
}

func (d *Daemon) runDownloadURLJob(ctx context.Context, rawURL string, importAfter bool) error {
	args := []string{"download-url", rawURL}
	if importAfter {
		args = append(args, "--import")
	}
	return d.runWorkerJob(ctx, args...)
}

func (d *Daemon) runCacheJob(ctx context.Context, genre string, force bool) error {
	args := []string{"cache"}
	if force {
		args = append(args, "--force")
	}
	if genre != "" {
		args = append(args, "--genre", genre)
	}
	return d.runWorkerJob(ctx, args...)
}

// runWorkerJob runs `evoplayer _job <args>` with background warming paused.
func (d *Daemon) runWorkerJob(ctx context.Context, args ...string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	d.warm.ClearPending()
	d.warm.Pause()
	defer d.warm.Resume()
	return superviseWorker(ctx, d.jobs, workerCmd(exe, append([]string{"_job"}, args...)...))
}

func isSoundCloudLikesURL(rawURL string) bool {
	return download.ClassifyURL(rawURL) == download.KindSCLikes
}

func workerCmd(exe string, args ...string) *exec.Cmd {
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stderr = os.Stderr
	return cmd
}

func superviseWorker(ctx context.Context, jm jobRelay, cmd *exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	scanDone := make(chan error, 1)
	go func() {
		scanDone <- relayWorkerStdout(stdout, jm)
	}()

	select {
	case <-ctx.Done():
		killWorkerGroup(cmd.Process.Pid, scanDone)
		_ = cmd.Wait()
		return ctx.Err()
	case scanErr := <-scanDone:
		err := cmd.Wait()
		if scanErr != nil {
			return scanErr
		}
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				return fmt.Errorf("worker exited with status %s", exitErr.ProcessState)
			}
			return err
		}
		return nil
	}
}

func relayWorkerStdout(r io.Reader, jm jobRelay) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var workerErr error
	for sc.Scan() {
		ev, err := worker.ParseEvent(sc.Bytes())
		if err != nil {
			continue
		}
		if ev.Type == "error" && ev.Message != "" {
			if workerErr == nil {
				workerErr = errors.New(ev.Message)
			}
			continue
		}
		worker.ApplyEvent(jm, ev)
	}
	scanErr := sc.Err()
	_, _ = io.Copy(io.Discard, r)
	if workerErr != nil {
		return workerErr
	}
	return scanErr
}

// killWorkerGroup stops the worker process group; the unreaped leader keeps pgid from being reused.
func killWorkerGroup(pgid int, stdoutDone <-chan error) {
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	select {
	case <-stdoutDone:
	case <-time.After(workerKillGrace):
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		<-stdoutDone
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
}

type jobRelay interface {
	AppendLog(string)
	SetProgress(jobs.Progress)
}

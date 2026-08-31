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
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return d.pauseWarmAndSupervise(ctx, soundCloudWorkerCmd(exe, importAfter))
}

func (d *Daemon) runImportJob(ctx context.Context) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return d.pauseWarmAndSupervise(ctx, importWorkerCmd(exe))
}

func (d *Daemon) runDownloadURLJob(ctx context.Context, rawURL string, importAfter bool) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return d.pauseWarmAndSupervise(ctx, downloadURLWorkerCmd(exe, rawURL, importAfter))
}

func (d *Daemon) runCacheJob(ctx context.Context, genre string, force bool) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return d.pauseWarmAndSupervise(ctx, cacheWorkerCmd(exe, genre, force))
}

func (d *Daemon) pauseWarmAndSupervise(ctx context.Context, cmd *exec.Cmd) error {
	d.warm.ClearPending()
	d.warm.Pause()
	defer d.warm.Resume()
	return superviseWorker(ctx, d.jobs, cmd)
}

func isSoundCloudLikesURL(rawURL string) bool {
	return download.ClassifyURL(rawURL) == download.KindSCLikes
}

func importWorkerCmd(exe string) *exec.Cmd {
	return workerCmd(exe, "_job", "import-incoming")
}

func soundCloudWorkerCmd(exe string, importAfter bool) *exec.Cmd {
	args := []string{"_job", "soundcloud-download"}
	if importAfter {
		args = append(args, "--import")
	}
	return workerCmd(exe, args...)
}

func downloadURLWorkerCmd(exe, rawURL string, importAfter bool) *exec.Cmd {
	args := []string{"_job", "download-url", rawURL}
	if importAfter {
		args = append(args, "--import")
	}
	return workerCmd(exe, args...)
}

func cacheWorkerCmd(exe, genre string, force bool) *exec.Cmd {
	args := []string{"_job", "cache"}
	if force {
		args = append(args, "--force")
	}
	if genre != "" {
		args = append(args, "--genre", genre)
	}
	return workerCmd(exe, args...)
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

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		killWorkerGroup(cmd.Process)
		<-waitDone
		<-scanDone
		return ctx.Err()
	case err := <-waitDone:
		scanErr := <-scanDone
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				return fmt.Errorf("worker exited with status %s", exitErr.ProcessState)
			}
			return err
		}
		if scanErr != nil {
			return scanErr
		}
		return nil
	}
}

func relayWorkerStdout(r io.Reader, jm jobRelay) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		ev, err := worker.ParseEvent(sc.Bytes())
		if err != nil {
			continue
		}
		if ev.Type == "error" && ev.Message != "" {
			return errors.New(ev.Message)
		}
		worker.ApplyEvent(jm, ev)
	}
	return sc.Err()
}

func killWorkerGroup(proc *os.Process) {
	if proc == nil {
		return
	}
	pgid := proc.Pid
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	deadline := time.Now().Add(workerKillGrace)
	for time.Now().Before(deadline) {
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
}

type jobRelay interface {
	AppendLog(string)
	SetProgress(jobs.Progress)
}

package util

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func RunShellCommand(timeout time.Duration, wd string, envs []string, stdout io.Writer, cmd string, args ...string) ([]byte, error) {
	if wd == "" {
		var err error
		wd, err = os.Getwd()
		if err != nil {
			log.Println(`Can't get current path. Set it as "."`)
			//wd = "."
			wd = ""
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, cmd, args...)
	command.Dir = wd
	command.Env = removeGODEBUG(append(envs[:len(envs):cap(envs)], os.Environ()...))
	var output []byte
	var err error
	if stdout != nil {
		command.Stdout = stdout
		command.Stderr = stdout
		err = command.Run()
	} else {
		var erroutput bytes.Buffer
		command.Stderr = &erroutput
		output, err = command.Output()
		if err != nil {
			if erroutput.Len() > 0 {
				err = fmt.Errorf("%w\n\n%s", err, erroutput.Bytes())
			}
		}
	}

	return output, err
}

func RunShell(timeout time.Duration, wd string, envs []string, stdout io.Writer, cmdAndArgs ...string) ([]byte, error) {
	if len(cmdAndArgs) == 0 {
		panic("command is not specified")
	}

	return RunShellCommand(timeout, wd, envs, stdout, cmdAndArgs[0], cmdAndArgs[1:]...)
}

func removeGODEBUG(envs []string) []string {
	r := envs[:0]
	for _, e := range envs {
		if !strings.HasPrefix(e, "GODEBUG=") {
			r = append(r, e)
		}
	}
	return r
}

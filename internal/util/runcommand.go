package util

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

func RunShellCommand(timeout time.Duration, wd string, buildEnv func() []string, stdout, stderr io.Writer, cmd string, args ...string) ([]byte, error) {
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
	if buildEnv != nil {
		command.Env = append(os.Environ(), buildEnv()...)
	}
	var output []byte
	var erroutput bytes.Buffer
	command.Stderr = &erroutput
	var err error
	if stdout != nil || stderr != nil {
		command.Stdout = stdout
		err = command.Run()
	} else {
		output, err = command.Output()
	}
	if err != nil {
		if erroutput.Len() > 0 {
			err = fmt.Errorf("%w\n\n%s", err, erroutput.Bytes())
		}
	} else if erroutput.Len() > 0 {
		err = fmt.Errorf("%s", erroutput.Bytes())
	}
	if stderr != nil {
		io.Copy(stderr, &erroutput)
	}

	return output, err
}

func RunShell(timeout time.Duration, wd string, buildEnv func() []string, stdout, stderr io.Writer, cmdAndArgs ...string) ([]byte, error) {
	if len(cmdAndArgs) == 0 {
		panic("command is not specified")
	}

	return RunShellCommand(timeout, wd, buildEnv, stdout, stderr, cmdAndArgs[0], cmdAndArgs[1:]...)
}

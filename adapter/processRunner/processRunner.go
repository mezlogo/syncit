package processrunner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	port "syncit/port/processRunner"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

var _ port.ProcessRunnerPort = (*Adapter)(nil)

func (a *Adapter) RunCommand(dryRun bool, name string, args []string) error {
	if dryRun {
		_, _ = fmt.Fprintln(os.Stdout, strings.Join(append([]string{name}, args...), " "))
		return nil
	}
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()

}

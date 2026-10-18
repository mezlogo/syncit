package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	adapterexec "basicgo/adapter/exec"
	adapterfs "basicgo/adapter/fs"
	cmdfind "basicgo/cmd/find"
	cmdls "basicgo/cmd/ls"
	corefind "basicgo/coremodule/find"
	corels "basicgo/coremodule/ls"
)

func main() {
	fileSystem := adapterfs.New()
	executor := adapterexec.New()

	lister := corels.New(fileSystem)
	finder := corefind.NewWithExecutor(fileSystem, executor)

	root := &cobra.Command{
		Use:          "basicgo",
		Short:        "basicgo is a small filesystem CLI",
		SilenceUsage: true,
	}
	root.AddCommand(cmdls.New(lister), cmdfind.New(finder))

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

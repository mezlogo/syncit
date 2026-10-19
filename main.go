package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	filesystemadapter "syncit/adapter/fileSystem"
	processrunneradapter "syncit/adapter/processRunner"
	repositorydeclarationadapter "syncit/adapter/repoAO"
	reporeportadapter "syncit/adapter/repoReport"
	reportcmd "syncit/cmd/report"
	synccmd "syncit/cmd/sync"
	difffileconfiguration "syncit/coremodule/diffFileConfiguration"
	mergefileconfiguration "syncit/coremodule/mergeFileConfiguration"
	processhandlers "syncit/coremodule/processHandlers"
	syncfileconfiguration "syncit/coremodule/syncFileConfiguration"
)

func main() {
	declarationPort := repositorydeclarationadapter.New()
	reportPort := reporeportadapter.New()
	fileSystemPort := filesystemadapter.New()
	processRunnerPort := processrunneradapter.New()

	mergeUseCase := mergefileconfiguration.New()
	diffUseCase := difffileconfiguration.New(fileSystemPort)
	syncUseCase := syncfileconfiguration.New(fileSystemPort)
	processUseCase := processhandlers.New(processRunnerPort)

	rootCmd := &cobra.Command{
		Use:           "syncit",
		Short:         "Persist and synchronize host configuration",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCmd.AddCommand(reportcmd.New(declarationPort, mergeUseCase, diffUseCase, processUseCase, reportPort))
	rootCmd.AddCommand(synccmd.New(declarationPort, mergeUseCase, syncUseCase, processUseCase, reportPort))

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

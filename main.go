package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	filesystemadapter "syncit/adapter/fileSystem"
	repositorydeclarationadapter "syncit/adapter/repoAO"
	reporeportadapter "syncit/adapter/repoReport"
	reportcmd "syncit/cmd/report"
	synccmd "syncit/cmd/sync"
	difffileconfiguration "syncit/coremodule/diffFileConfiguration"
	mergefileconfiguration "syncit/coremodule/mergeFileConfiguration"
	syncfileconfiguration "syncit/coremodule/syncFileConfiguration"
)

func main() {
	declarationPort := repositorydeclarationadapter.New()
	reportPort := reporeportadapter.New()
	fileSystemPort := filesystemadapter.New()

	mergeUseCase := mergefileconfiguration.New()
	diffUseCase := difffileconfiguration.New(fileSystemPort)
	syncUseCase := syncfileconfiguration.New(fileSystemPort)

	rootCmd := &cobra.Command{
		Use:           "syncit",
		Short:         "Persist and synchronize host configuration",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCmd.AddCommand(reportcmd.New(declarationPort, mergeUseCase, diffUseCase, reportPort))
	rootCmd.AddCommand(synccmd.New(declarationPort, mergeUseCase, syncUseCase, reportPort))

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

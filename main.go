package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	filesystemadapter "syncit/adapter/fileSystem"
	repositorydeclarationadapter "syncit/adapter/repoAO"
	reporeportadapter "syncit/adapter/repoReport"
	reportcmd "syncit/cmd/report"
	difffileconfiguration "syncit/coremodule/diffFileConfiguration"
	mergefileconfiguration "syncit/coremodule/mergeFileConfiguration"
)

func main() {
	declarationPort := repositorydeclarationadapter.New()
	reportPort := reporeportadapter.New()
	fileSystemPort := filesystemadapter.New()

	mergeUseCase := mergefileconfiguration.New()
	diffUseCase := difffileconfiguration.New(fileSystemPort)

	rootCmd := &cobra.Command{
		Use:           "syncit",
		Short:         "Persist and synchronize host configuration",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCmd.AddCommand(reportcmd.New(declarationPort, mergeUseCase, diffUseCase, reportPort))

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

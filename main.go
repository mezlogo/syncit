package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	repositorydeclarationadapter "syncit/adapter/repoAO"
	reporeportadapter "syncit/adapter/repoReport"
	reportcmd "syncit/cmd/report"
)

func main() {
	declarationPort := repositorydeclarationadapter.New()
	reportPort := reporeportadapter.New()

	rootCmd := &cobra.Command{
		Use:           "syncit",
		Short:         "Persist and synchronize host configuration",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCmd.AddCommand(reportcmd.New(declarationPort, reportPort))

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

package sync

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	mergefileconfiguration "syncit/port/mergeFileConfiguration"
	repositorydeclaration "syncit/port/repoAO"
	reporeport "syncit/port/repoReport"
	syncfileconfiguration "syncit/port/syncFileConfiguration"
	toolsconfiguraion "syncit/port/toolsConfiguraion"
)

func New(
	declarationPort repositorydeclaration.RepositoryDeclarationPort,
	mergePort mergefileconfiguration.MergeFileConfigurationUseCase,
	syncPort syncfileconfiguration.SyncFileConfigurationUseCase,
	reportPort reporeport.RepoReportUseCase,
) *cobra.Command {
	var repositories []string
	var hostName, outputHome, outputRoot string
	var overwrite, reset bool

	defaultRepositories := splitCommaSeparated(os.Getenv("SYNCIT_REPO"))
	defaultHostName := strings.TrimSpace(os.Getenv("SYNCIT_HOST"))
	if defaultHostName == "" {
		if h, err := os.Hostname(); err == nil {
			defaultHostName = h
		}
	}
	defaultOutputHome := strings.TrimSpace(os.Getenv("HOME"))
	if defaultOutputHome == "" {
		if h, err := os.UserHomeDir(); err == nil {
			defaultOutputHome = h
		}
	}

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize user file configuration (links only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			repos := normalizeList(repositories)
			if len(repos) == 0 {
				return errors.New("repository is required: use --repo or SYNCIT_REPO")
			}
			host := strings.TrimSpace(hostName)
			if host == "" {
				return errors.New("host is required: use --host or SYNCIT_HOST")
			}

			modules := make([]repositorydeclaration.RepositoryDeclarationModule, 0)
			for _, repo := range repos {
				result, err := declarationPort.ParseRepository(repositorydeclaration.RepositoryDeclarationCommand{
					RepositoryPath: repo,
					HostName:       host,
				})
				if err != nil {
					return fmt.Errorf("parse repository %q: %w", repo, err)
				}
				modules = append(modules, result.Modules...)
			}

			mapping := toolsconfiguraion.NewResolvedFileConfigurationMapping(outputRoot, outputHome)
			resolved, mergeErr := mergePort.MergeFiles(modules, mapping)

			decisions := syncPort.SyncUserFiles(resolved.UserFiles, syncfileconfiguration.SyncOptions{
				Overwrite: overwrite,
				Reset:     reset,
			})

			reportPort.ReportSync(strings.Join(repos, ", "), decisions, mergeErr)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&repositories, "repo", defaultRepositories, "repository directories separated by comma; defaults to $SYNCIT_REPO")
	cmd.Flags().StringVar(&hostName, "host", defaultHostName, "host declaration name; defaults to $SYNCIT_HOST or os hostname")
	cmd.Flags().StringVar(&outputHome, "output-home", defaultOutputHome, "home directory used for file mapping; defaults to $HOME")
	cmd.Flags().StringVar(&outputRoot, "output-root", "/", "root directory used for file mapping; defaults to /")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "recreate existing files or wrong links at the target")
	cmd.Flags().BoolVar(&reset, "reset", false, "recreate the link even when it already points to the declared file")

	return cmd
}

func normalizeList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, v := range values {
		for _, p := range strings.Split(v, ",") {
			if p = strings.TrimSpace(p); p != "" {
				result = append(result, p)
			}
		}
	}
	return result
}

func splitCommaSeparated(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return normalizeList([]string{value})
}

package sync

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	mergefileconfiguration "syncit/port/mergeFileConfiguration"
	processhandlers "syncit/port/processHandlers"
	repositorydeclaration "syncit/port/repoAO"
	reporeport "syncit/port/repoReport"
	syncfileconfiguration "syncit/port/syncFileConfiguration"
	toolsconfiguraion "syncit/port/toolsConfiguraion"
)

func New(
	declarationPort repositorydeclaration.RepositoryDeclarationPort,
	mergePort mergefileconfiguration.MergeFileConfigurationUseCase,
	syncPort syncfileconfiguration.SyncFileConfigurationUseCase,
	processPort processhandlers.ProcessHandlersUseCase,
	reportPort reporeport.RepoReportUseCase,
) *cobra.Command {
	var repositories []string
	var hostName, outputHome, outputRoot string
	var overwrite, reset, dryRun bool
	var includeUser, includeRoot, includeProcess bool

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
	defaultOutputRoot := "/"

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize file configuration and run process handlers",
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
			mergedHandlers := map[string]string{}
			for _, repo := range repos {
				result, err := declarationPort.ParseRepository(repositorydeclaration.RepositoryDeclarationCommand{
					RepositoryPath: repo,
					HostName:       host,
				})
				if err != nil {
					return fmt.Errorf("parse repository %q: %w", repo, err)
				}
				modules = append(modules, result.Modules...)

				handlers, err := declarationPort.ParseHandlers(repo)
				if err != nil {
					return fmt.Errorf("parse handlers %q: %w", repo, err)
				}
				for k, v := range handlers {
					mergedHandlers[k] = v
				}
			}

			mergedDeclarations := map[string][]string{}
			for _, m := range modules {
				for k, vals := range m.Declarations {
					mergedDeclarations[k] = append(mergedDeclarations[k], vals...)
				}
			}

			anyInclude := includeUser || includeRoot || includeProcess
			processUser := includeUser || !anyInclude
			processRoot := includeRoot || !anyInclude
			processHandlers := includeProcess || !anyInclude

			mapping := toolsconfiguraion.NewResolvedFileConfigurationMapping(outputRoot, outputHome)
			resolved, mergeErr := mergePort.MergeFiles(modules, mapping)
			opts := syncfileconfiguration.SyncOptions{Overwrite: overwrite, Reset: reset}

			processCmd := processhandlers.ProcessHandlersCommand{
				Handlers:     mergedHandlers,
				Declarations: mergedDeclarations,
				DryRun:       dryRun,
			}

			if processHandlers {
				pre := processPort.ProcessPreFileHandlers(processCmd)
				reportPort.ReportProcess(strings.Join(repos, ", "), pre.Decisions, nil)
			}
			if processUser {
				reportPort.ReportSync(strings.Join(repos, ", "),
					syncPort.SyncUserFiles(resolved.UserFiles, opts), mergeErr)
			}
			if processRoot {
				reportPort.ReportSyncRoot(strings.Join(repos, ", "),
					syncPort.SyncRootFiles(resolved.RootFiles, opts), mergeErr)
			}
			if processHandlers {
				post := processPort.ProcessPostFileHandlers(processCmd)
				reportPort.ReportProcess(strings.Join(repos, ", "), post.Decisions, nil)
			}
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&repositories, "repo", defaultRepositories, "repository directories separated by comma; defaults to $SYNCIT_REPO")
	cmd.Flags().StringVar(&hostName, "host", defaultHostName, "host declaration name; defaults to $SYNCIT_HOST or os hostname")
	cmd.Flags().StringVar(&outputHome, "output-home", defaultOutputHome, "home directory used for file mapping; defaults to $HOME")
	cmd.Flags().StringVar(&outputRoot, "output-root", defaultOutputRoot, "root directory used for file mapping; defaults to /")
	cmd.Flags().BoolVar(&includeUser, "include-user", false, "process user file configuration when any include flag is set")
	cmd.Flags().BoolVar(&includeRoot, "include-root", false, "process root file configuration when any include flag is set")
	cmd.Flags().BoolVar(&includeProcess, "include-process", false, "process process handlers when any include flag is set")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "recreate existing files or wrong links at the target")
	cmd.Flags().BoolVar(&reset, "reset", false, "recreate the link even when it already points to the declared file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print process handler commands instead of executing them")

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

package report

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	difffileconfiguration "syncit/port/diffFileConfiguration"
	mergefileconfiguration "syncit/port/mergeFileConfiguration"
	processhandlers "syncit/port/processHandlers"
	repositorydeclaration "syncit/port/repoAO"
	reporeport "syncit/port/repoReport"
	toolsconfiguraion "syncit/port/toolsConfiguraion"
)

func New(
	declarationPort repositorydeclaration.RepositoryDeclarationPort,
	mergePort mergefileconfiguration.MergeFileConfigurationUseCase,
	diffPort difffileconfiguration.DiffFileConfigurationUseCase,
	processPort processhandlers.ProcessHandlersUseCase,
	reportPort reporeport.RepoReportUseCase,
) *cobra.Command {
	var repositories []string
	var hostName string
	var merged, diff bool
	var outputHome, outputRoot string
	var includeUser, includeRoot, includeProcess bool

	defaultRepositories := splitCommaSeparated(os.Getenv("SYNCIT_REPO"))
	defaultHostName := strings.TrimSpace(os.Getenv("SYNCIT_HOST"))
	if defaultHostName == "" {
		if hostname, err := os.Hostname(); err == nil {
			defaultHostName = hostname
		}
	}
	defaultOutputHome := strings.TrimSpace(os.Getenv("HOME"))
	if defaultOutputHome == "" {
		if home, err := os.UserHomeDir(); err == nil {
			defaultOutputHome = home
		}
	}
	defaultOutputRoot := "/"

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Print repository declaration, merged files, diff or process list",
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

			modulesByRepo := make(map[string][]repositorydeclaration.RepositoryDeclarationModule, len(repos))
			for _, repo := range repos {
				result, err := declarationPort.ParseRepository(repositorydeclaration.RepositoryDeclarationCommand{
					RepositoryPath: repo,
					HostName:       host,
				})
				if err != nil {
					return fmt.Errorf("parse repository %q: %w", repo, err)
				}
				modules = append(modules, result.Modules...)
				modulesByRepo[repo] = result.Modules

			}

			mergedDeclarations := map[string][]string{}
			for _, m := range modules {
				for k, vals := range m.Declarations {
					mergedDeclarations[k] = append(mergedDeclarations[k], vals...)
				}
			}

			deduplicate := func(values []string) []string {
				if len(values) == 0 {
					return values
				}
				seen := make(map[string]struct{}, len(values))
				result := make([]string, 0, len(values))
				for _, value := range values {
					value = strings.TrimSpace(value)
					if value == "" {
						continue
					}
					if _, ok := seen[value]; ok {
						continue
					}
					seen[value] = struct{}{}
					result = append(result, value)
				}
				return result
			}
			for _, tool := range []string{"pacman", "aur", "systemd", "usersystemd"} {
				mergedDeclarations[tool] = deduplicate(mergedDeclarations[tool])
			}

			if merged || diff || includeProcess {
				anyInclude := includeUser || includeRoot || includeProcess
				processUser := includeUser || !anyInclude
				processRoot := includeRoot || !anyInclude

				mapping := toolsconfiguraion.NewResolvedFileConfigurationMapping(outputRoot, outputHome)
				resolved, mergeErr := mergePort.MergeFiles(modules, mapping)

				if merged {
					filtered := mergefileconfiguration.ResolvedFileConfiguration{}
					if processUser {
						filtered.UserFiles = resolved.UserFiles
					}
					if processRoot {
						filtered.RootFiles = resolved.RootFiles
					}
					reportPort.ReportMergedRepo(strings.Join(repos, ", "), filtered, mergeErr)
				}

				if diff {
					if processUser {
						reportPort.ReportDiff(strings.Join(repos, ", "), diffPort.CalculateDiffForUserFiles(resolved.UserFiles), mergeErr)
					}
					if processRoot {
						reportPort.ReportDiffRoot(strings.Join(repos, ", "), diffPort.CalculateDiffForRootFiles(resolved.RootFiles), mergeErr)
					}
				}

				if includeProcess {
					processCmd := processhandlers.ProcessHandlersCommand{

						Declarations: mergedDeclarations,
						DryRun:       true,
					}
					pre := processPort.ProcessPreFileHandlers(processCmd)
					reportPort.ReportProcess(strings.Join(repos, ", "), pre.Decisions, nil)
					post := processPort.ProcessPostFileHandlers(processCmd)
					reportPort.ReportProcess(strings.Join(repos, ", "), post.Decisions, nil)
				}

				reportPort.ReportDeclarations(strings.Join(repos, ", "), mergedDeclarations)
				return nil
			}

			for _, repo := range repos {
				reportPort.ReportRepo(repo, modulesByRepo[repo])
			}
			reportPort.ReportDeclarations(strings.Join(repos, ", "), mergedDeclarations)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&repositories, "repo", defaultRepositories, "repository directories separated by comma; defaults to $SYNCIT_REPO")
	cmd.Flags().StringVar(&hostName, "host", defaultHostName, "host declaration name; defaults to $SYNCIT_HOST or os hostname")
	cmd.Flags().BoolVar(&merged, "merged", false, "merge file configuration and report resolved files and errors")
	cmd.Flags().BoolVar(&diff, "diff", false, "calculate and report diff for resolved files")
	cmd.Flags().StringVar(&outputHome, "output-home", defaultOutputHome, "home directory used for merged file mapping; defaults to $HOME")
	cmd.Flags().StringVar(&outputRoot, "output-root", defaultOutputRoot, "root directory used for merged file mapping; defaults to /")
	cmd.Flags().BoolVar(&includeUser, "include-user", false, "process user file configuration when any include flag is set")
	cmd.Flags().BoolVar(&includeRoot, "include-root", false, "process root file configuration when any include flag is set")
	cmd.Flags().BoolVar(&includeProcess, "include-process", false, "report process handler commands (dry-run) when any include flag is set")

	return cmd
}

func normalizeList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
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

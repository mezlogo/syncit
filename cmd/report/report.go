package report

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	repositorydeclaration "syncit/port/repoAO"
)

// New creates the report command. It depends only on the repository declaration port.
func New(declarationPort repositorydeclaration.RepositoryDeclarationPort) *cobra.Command {
	var repositories []string
	var hostName string

	defaultRepositories := splitCommaSeparated(os.Getenv("SYNCIT_REPO"))
	defaultHostName := strings.TrimSpace(os.Getenv("SYNCIT_HOST"))
	if defaultHostName == "" {
		if hostname, err := os.Hostname(); err == nil {
			defaultHostName = hostname
		}
	}

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Print repository declaration for a host",
		RunE: func(cmd *cobra.Command, args []string) error {
			repos := normalizeList(repositories)
			if len(repos) == 0 {
				return errors.New("repository is required: use --repo or SYNCIT_REPO")
			}
			host := strings.TrimSpace(hostName)
			if host == "" {
				return errors.New("host is required: use --host or SYNCIT_HOST")
			}

			out := cmd.OutOrStdout()
			for i, repo := range repos {
				result, err := declarationPort.ParseRepository(repositorydeclaration.RepositoryDeclarationCommand{

					RepositoryPath: repo,
					HostName:       host,
				})
				if err != nil {
					return fmt.Errorf("parse repository %q: %w", repo, err)
				}
				if i > 0 {
					if _, err := fmt.Fprintln(out); err != nil {
						return err
					}
				}
				if err := printResult(out, repo, host, result); err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&repositories, "repo", defaultRepositories, "repository directories separated by comma; defaults to $SYNCIT_REPO")
	cmd.Flags().StringVar(&hostName, "host", defaultHostName, "host declaration name; defaults to $SYNCIT_HOST or os hostname")
	return cmd
}

func printResult(out io.Writer, repo string, host string, result repositorydeclaration.RepositoryDeclarationResult) error {
	var b strings.Builder
	appendf(&b, "repository: %s\n", repo)
	appendf(&b, "host: %s\n", host)
	if len(result.Modules) == 0 {
		appendf(&b, "modules: []\n")
		_, err := fmt.Fprint(out, b.String())
		return err
	}

	appendf(&b, "modules:\n")
	for _, module := range result.Modules {
		appendf(&b, "  - name: %s\n", module.ModuleName)
		appendf(&b, "    path: %s\n", module.ModuleAbsoulutePath)

		if len(module.Declarations) == 0 {
			appendf(&b, "    declarations: {}\n")
		} else {
			appendf(&b, "    declarations:\n")
			keys := make([]string, 0, len(module.Declarations))
			for key := range module.Declarations {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				values := module.Declarations[key]
				if len(values) == 0 {
					appendf(&b, "      %s: []\n", key)
					continue
				}
				appendf(&b, "      %s:\n", key)
				for _, value := range values {
					appendf(&b, "        - %s\n", value)
				}
			}
		}

		if len(module.RelativeFilePaths) == 0 {
			appendf(&b, "    files: []\n")
		} else {
			appendf(&b, "    files:\n")
			for _, file := range module.RelativeFilePaths {
				appendf(&b, "      - %s\n", file)
			}
		}
	}

	_, err := fmt.Fprint(out, b.String())
	return err
}

func appendf(b *strings.Builder, format string, args ...any) {
	_, _ = fmt.Fprintf(b, format, args...)
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

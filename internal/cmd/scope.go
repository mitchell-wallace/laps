package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchell-wallace/laps/internal/store"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	scopeActive bool
	scopeRoot   bool
	scopeStint  string
)

const scopeMutexAnnotation = "cobra_annotation_mutually_exclusive"

func addScopeFlags(cmd *cobra.Command) {
	ensureRootFlags()
	cmd.Flags().BoolVarP(&scopeActive, "active", "c", false, "target the deepest active queue")
	cmd.Flags().BoolVarP(&scopeRoot, "root", "r", false, "target the root queue")
	cmd.Flags().StringVarP(&scopeStint, "stint", "s", "", "target a named stint queue")
	markScopeMutex(rootCmd.PersistentFlags().Lookup("file"))
	markScopeMutex(cmd.Flags().Lookup("root"))
	markScopeMutex(cmd.Flags().Lookup("stint"))
	markScopeMutex(cmd.Flags().Lookup("active"))
}

func markScopeMutex(flag *pflag.Flag) {
	if flag == nil {
		panic("missing scope mutex flag")
	}
	if flag.Annotations == nil {
		flag.Annotations = map[string][]string{}
	}
	const group = "file root stint active"
	for _, existing := range flag.Annotations[scopeMutexAnnotation] {
		if existing == group {
			return
		}
	}
	flag.Annotations[scopeMutexAnnotation] = append(flag.Annotations[scopeMutexAnnotation], group)
}

func scopedStorePath(beadsDir string) string {
	if scopeStint != "" {
		path, err := resolveScopeStintPath(beadsDir, scopeStint)
		if err != nil {
			exit(2, "%v", err)
		}
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				exit(3, "stint %s not found", scopeStint)
			}
			exit(2, "%v", err)
		}
		return path
	}
	if scopeRoot {
		return filepath.Join(beadsDir, store.ResolveFile(""))
	}
	return filepath.Join(beadsDir, store.ResolveFile(fileFlag))
}

func resolveScopeStintPath(beadsDir, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("stint name cannot be blank")
	}
	return store.ResolveStintFile(beadsDir, name)
}

func scopeFlagInArgs(args []string) (string, bool) {
	for _, arg := range args {
		switch {
		case arg == "--active" || arg == "-c":
			return arg, true
		case arg == "--root" || arg == "-r":
			return arg, true
		case arg == "--stint" || arg == "-s":
			return arg, true
		case len(arg) > len("--stint=") && arg[:len("--stint=")] == "--stint=":
			return "--stint", true
		case len(arg) > len("-s=") && arg[:len("-s=")] == "-s=":
			return "-s", true
		}
	}
	return "", false
}

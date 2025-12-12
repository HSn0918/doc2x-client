package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// positionalAlwaysFlags returns all flags (local + inherited) even when user did not type a dash.
func positionalAlwaysFlags(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	flags := make([]string, 0, 16)

	add := func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		if f.Shorthand != "" {
			flags = append(flags, "-"+f.Shorthand)
		}
		flags = append(flags, "--"+f.Name)
	}

	cmd.NonInheritedFlags().VisitAll(add)
	cmd.InheritedFlags().VisitAll(add)

	return flags, cobra.ShellCompDirectiveNoFileComp
}

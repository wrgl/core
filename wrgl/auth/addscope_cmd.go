package auth

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wrgl/core/pkg/auth"
	authfs "github.com/wrgl/core/pkg/auth/fs"
	conffs "github.com/wrgl/core/pkg/conf/fs"
	"github.com/wrgl/core/wrgl/utils"
)

var validScopes = map[string]string{
	auth.ScopeRead:        "covers view-only actions such as fetch, diff, etc...",
	auth.ScopeReadConfig:  "covers read config action",
	auth.ScopeWrite:       "covers write actions such as push, commit, etc...",
	auth.ScopeWriteConfig: "covers write config action",
}

func allScopesString(indent int, withDesc bool) string {
	sl := []string{}
	spaces := strings.Repeat(" ", indent)
	for scope, desc := range validScopes {
		var s string
		if withDesc {
			s = fmt.Sprintf("%s%s: %s", spaces, scope, desc)
		} else {
			s = fmt.Sprintf("%s%s", spaces, scope)
		}
		sl = append(sl, s)
	}
	sort.Strings(sl)
	return strings.Join(sl, "\n")
}

func addscopeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addscope EMAIL SCOPE...",
		Short: "Add one or more scopes for a user.",
		Long:  "Add one or more scopes for a user. Valid scopes are:\n" + allScopesString(4, true),
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := utils.MustWRGLDir(cmd)
			cs := conffs.NewStore(dir, conffs.AggregateSource, "")
			c, err := cs.Open()
			if err != nil {
				return err
			}
			authnS, err := authfs.NewAuthnStore(dir, c.TokenDuration())
			if err != nil {
				return err
			}
			if !authnS.Exist(args[0]) {
				return UserNotFoundErr(args[0])
			}
			authzS, err := authfs.NewAuthzStore(dir)
			if err != nil {
				return err
			}
			for _, scope := range args[1:] {
				if _, ok := validScopes[scope]; !ok {
					return InvalidScopeErr(scope)
				}
				if err := authzS.AddPolicy(args[0], scope); err != nil {
					return err
				}
			}
			return authzS.Flush()
		},
	}
	return cmd
}

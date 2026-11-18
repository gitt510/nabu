package main

import (
	"github.com/spf13/cobra"

	"github.com/gitt510/nabu/internal/api"
)

// signIn runs the browser sign-in for the Worker at url; tests replace it.
var signIn = api.Login

func loginCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "sign in to the Worker through Cloudflare Access",
		Long: `sign in to the Worker at base_url through Cloudflare Access.
cloudflared opens the browser and keeps the token; the token is never printed.
run it once, and again when nabu says not signed in.`,
		Args: cobra.NoArgs,
		RunE: do(func([]string) int {
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			if err := signIn(c.URL, e.stderr); err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return emit(e.stdout, map[string]string{"action": "login", "url": c.URL})
		}),
	}
}

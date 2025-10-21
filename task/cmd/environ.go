package cmd

import (
	"os"
	"strings"
)

func Environ(extraEnv ...string) (environ []string) {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "CIRCLECI_GOAT") {
			// Prevent internal configuration from being unintentionally injected in the command environment
			continue
		}
		environ = append(environ, env)
	}
	if extraEnv != nil {
		environ = append(environ, extraEnv...)
	}

	return environ
}

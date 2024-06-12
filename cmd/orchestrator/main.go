package main

import (
	"log" //nolint:depguard // a non-O11y log is allowed for a top-level fatal exit

	"github.com/circleci/runner-init/cmd"
)

func main() {
	if err := run(cmd.Version, cmd.Date); err != nil {
		log.Fatal(err)
	}
}

//nolint:unparam // TODO
func run(version, date string) (err error) {
	println(version, date)

	return err
}

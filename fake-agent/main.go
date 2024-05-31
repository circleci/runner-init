package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
)

func main() {
	// Example arguments: _internal agent-runner --verbose --agentAPITaskToken=a84dc93b7900c6ca3e74091785d4a883 --runnerAPIBaseURL=http://127.0.0.1:65244

	opts := CommandRequest{}

	kong.Parse(&opts, kong.Name("circleci-agent"))

	// We need the fake task agent binary to be able to access the local test API from inside the cluster. If we are using kind (or another docker hosted k8s cluster)
	// for testing then using the domain below will forward requests on to the host machine's localhost interface
	driver := os.Getenv("DRIVER_MODE")
	dockerHostAddress := os.Getenv("DOCKER_HOST")
	if driver == "kubernetes" {
		url, err := url.Parse(opts.RunnerAPIBaseURL)
		mustNotError(err)
		url.Host = fmt.Sprintf("%s:%s", dockerHostAddress, url.Port())
		opts.RunnerAPIBaseURL = url.String()
	}

	run(opts)
}

func run(opts CommandRequest) {
	fmt.Println("fake circleci-agent 1f3k128h")

	b, err := io.ReadAll(os.Stdin)
	mustNotError(err)
	opts.Stdin = string(b)

	b, err = json.Marshal(opts)
	mustNotError(err)
	res, err := http.Post(opts.RunnerAPIBaseURL+"/fakes/agent/command", "application/json", bytes.NewReader(b))
	mustNotError(err)
	mustNotError(res.Body.Close())

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGTERM, os.Interrupt)

	shutdownTimer := time.NewTimer(time.Second * 4)
	for {
		intervalTimer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-shutdownTimer.C:
			intervalTimer.Stop()
			fmt.Println("fake circleci-agent shutting down after timeout")
			return
		case <-intervalTimer.C:
			resp, err := http.Get(opts.RunnerAPIBaseURL + "/fakes/agent/terminated")
			if err != nil {
				// Exit if the test API has been closed
				return
			}
			mustNotError(resp.Body.Close())

			if resp.StatusCode == http.StatusOK {
				shutdownTimer.Stop()
				fmt.Println("fake circleci-agent shutting down at the request of the test harness")
				resp, err := http.Post(opts.RunnerAPIBaseURL+"/fakes/agent/terminated", "",
					strings.NewReader(`{"by":"test-api"}`))
				mustNotError(err)
				mustNotError(resp.Body.Close())
				return
			}
			// Ignore the 413 as a thing. This is just a handy code to pick to trigger us to start misbehaving
			if resp.StatusCode == http.StatusRequestEntityTooLarge {
				fmt.Println("fake circleci-agent staring to misbehave")
				launchChildAndMisbehave(opts)
				panic("shouldn't get here in a test")
			}
		case <-terminate:
			intervalTimer.Stop()
			shutdownTimer.Stop()
			fmt.Println("fake circleci-agent shutting down after SIGTERM, notifying API")
			resp, err := http.Post(opts.RunnerAPIBaseURL+"/fakes/agent/terminated", "",
				strings.NewReader(`{"by":"launch-agent"}`))
			mustNotError(err)
			mustNotError(resp.Body.Close())
			return
		}
	}
}

// Misbehaving mode to test PID group cleanup
func launchChildAndMisbehave(opts CommandRequest) {
	signal.Ignore(os.Interrupt)

	cmd := exec.Command("/bin/sleep", "20")
	err := cmd.Start()
	mustNotError(err)

	res, err := http.Get(fmt.Sprintf("%s/fakes/agent/pids?taskagent=%d&taskagent-child=%d",
		opts.RunnerAPIBaseURL, os.Getpid(), cmd.Process.Pid))
	_ = res.Body.Close()
	mustNotError(err)

	err = cmd.Wait()
	mustNotError(err)
}

func mustNotError(err error) {
	if err != nil {
		panic(err)
	}
}

type CommandRequest struct {
	RunnerAPIBaseURL      string        `name:"runnerAPIBaseURL" required:"true"`
	Allocation            string        `name:"allocation" required:"true"`
	Verbose               bool          `name:"verbose"`
	DisableSpinUpStep     bool          `name:"disableSpinUpStep"`
	DisableIsolatedSSHDir bool          `name:"disableIsolatedSSHDir"`
	Args                  []string      `arg:""`
	WorkDir               string        `name:"workDir"`
	Stdin                 string        `kong:"-"`
	MaxRunTime            time.Duration `name:"maxRunTime"`
	PidFile               string        `name:"pidfile"`
}

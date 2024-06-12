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

func run(version, date string) (err error) {
	////orchestrator := setup(version, date)
	//task.Agent{}
	//
	//ctx, o11yCleanup, err := orchestrator.O11yConfig().Load(version, "orchestrator")
	//if err != nil {
	//	return err
	//}
	//defer o11yCleanup(ctx)
	//
	//ctx, runSpan := o11y.StartSpan(ctx, "main: run")
	//defer o11y.End(runSpan, &err)
	//
	//o11y.Log(ctx, "starting orchestrator",
	//	o11y.Field("version", version),
	//	o11y.Field("date", date),
	//)
	//
	sys := system.New()
	defer sys.Cleanup(ctx)
	//
	//if err := cmd.UpdateDefaultTransport(ctx); err != nil {
	//	return fmt.Errorf("failed to load rootcerts: %w", err)
	//}
	//
	//c := orchestrator.ServiceConfig()
	//if c.Driver, err = orchestrator.MakeLogger(); err != nil {
	//	return fmt.Errorf("failed to make logger: %w", err)
	//}
	//
	//service := logging.New(c)
	//
	//sys.AddService(service.Log)

	return sys.Run(ctx, agent.ServiceShutdownDelay())
}

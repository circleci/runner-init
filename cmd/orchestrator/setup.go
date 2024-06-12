package main

//type cli struct {
//	Version kong.VersionFlag `short:"v" help:"Print version information and quit."`
//
//	Kubernetes struct {
//		kubernetes.LoggingAgentConfig `embed:""`
//	} `cmd:"" default:"1"`
//}
//
//func setup(version, date string) config.LoggingAgent {
//	cli := cli{}
//
//	ctx := kong.Parse(&cli,
//		kong.Description(
//			"This is the logging orchestrator for CircleCI's self-hosted runner. "+
//				"It is responsible for collecting logs from service containers."+
//				"\n\nFor more information on CircleCI runner, visit https://circleci.com/docs/runner-overview/."),
//		kong.Vars{
//			"version": fmt.Sprintf("%s version %s (built %s)", "circleci-runner", version, date),
//		})
//
//	switch ctx.Command() {
//	case "kubernetes":
//		return cli.Kubernetes
//	default:
//		return nil
//	}
//}

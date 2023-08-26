package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/hashicorp/go-tfe"
	"github.com/nicholasjackson/env"
)

var token = env.String("TFC_TOKEN", true, "", "API Token for TFC/TFE")
var org = env.String("TFC_ORG", true, "", "TFC/TFE Organization")
var workspace = env.String("TFC_WORKSPACE", true, "", "TFC/TFE Workspace")

var out = flag.String("out", "out.json", "output file for plan")

func main() {
	flag.Parse()

	if flag.ErrHelp != nil {
		fmt.Println("usage: tfc-plan -out myfile.json")
		fmt.Println("")
		fmt.Println("Configuration values are set using environment variables, for info please see the following list.")
		fmt.Println(env.Help())
		os.Exit(0)
	}

	err := env.Parse()

	if err != nil {
		fmt.Println("Configuration values are set using environment variables, for info please see the following list.")
		fmt.Println(err)
		fmt.Println("")

		fmt.Println(env.Help())
		os.Exit(1)
	}

	cfg := tfe.DefaultConfig()
	cfg.Token = *token

	cli, err := tfe.NewClient(cfg)
	if err != nil {
		log.Fatal("Unable to create the client", "error", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// find the workspace
	ws, err := cli.Workspaces.Read(ctx, *org, *workspace)
	if err != nil {
		log.Fatal("Unable to fetch workspace details", "org", *org, "workspace", *workspace, "error", err)
	}

	log.Info("Creating plan", "org", *org, "workspace", *workspace)

	planOnly := true
	run, err := cli.Runs.Create(ctx, tfe.RunCreateOptions{PlanOnly: &planOnly, Workspace: ws})
	if err != nil {
		log.Fatal("Unable to create plan", "org", *org, "workspace", *workspace, "error", err)
	}

	log.Info(run.Message)

	// wait for the plan to finish
	for {
		if ctx.Err() != nil {
			log.Fatal("Timeout waiting for plan", "error", err)
		}

		log.Info("Checking plan status", "id", run.ID)
		s, err := cli.Runs.Read(ctx, run.ID)
		if err != nil {
			log.Fatal("Unable to check plan status", "error", err)
		}

		switch s.Status {
		case tfe.RunPlannedAndFinished:
			// output the plan
			log.Info("Plan complete")
			d, err := cli.Plans.ReadJSONOutput(ctx, s.Plan.ID)
			if err != nil {
				log.Fatal("Unable to read plan output", "error", err)
			}

			os.WriteFile(*out, d, os.ModePerm)
			log.Info("Successfully written plan", "out", *out)
			os.Exit(0)
		case tfe.RunErrored:
			log.Fatal("Unable to create plan", "message", s.Message)
		}

		log.Info("Wait for plan to complete", "current status", s.Status)

		time.Sleep(5 * time.Second)
	}
}

// Command backup-service periodically pulls a snapshot from the main KnowledgeOS
// application, pushes the code to GitHub and stores versioned data dumps locally.
// It is configured entirely through environment variables (see config.go) and
// logs to stdout. Run with --once to perform a single cycle and exit.
package main

import (
	"flag"
	"os"
	"time"
)

func main() {
	once := flag.Bool("once", false, "run a single backup cycle and exit")
	flag.Parse()

	cfg, err := LoadConfig()
	if err != nil {
		NewLogger("").Errorf("configuration error: %v", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(cfg.BackupsPath, 0o700); err != nil {
		NewLogger("", cfg.GitHubToken, cfg.AppAPIKey).Errorf("cannot create backups path %s: %v", cfg.BackupsPath, err)
		os.Exit(1)
	}

	log := NewLogger(JournalPath(cfg.BackupsPath), cfg.GitHubToken, cfg.AppAPIKey)

	if *once {
		if err := runCycle(cfg, log); err != nil {
			log.Errorf("%v", err)
			os.Exit(1)
		}
		return
	}

	sched, err := parseCron(cfg.ScheduleCron)
	if err != nil {
		log.Errorf("invalid SCHEDULE_CRON %q: %v", cfg.ScheduleCron, err)
		os.Exit(1)
	}

	log.Infof("backup-service started: schedule=%q (UTC) backups=%s keep=%d app=%s",
		cfg.ScheduleCron, cfg.BackupsPath, cfg.BackupsKeep, cfg.AppURL)

	runScheduler(cfg, sched, log)
}

// runScheduler wakes at every minute boundary (UTC) and runs a cycle whenever
// the schedule matches. A failing cycle is logged but never stops the loop
// (TZ §5.8).
func runScheduler(cfg Config, sched *cronSchedule, log *Logger) {
	for {
		next := time.Now().UTC().Truncate(time.Minute).Add(time.Minute)
		time.Sleep(time.Until(next))

		if sched.Matches(time.Now().UTC()) {
			if err := runCycle(cfg, log); err != nil {
				log.Errorf("%v", err)
			}
		}
	}
}

package main

import (
	"flag"
	"os"
	"os/signal"
	"time"

	"log"

	"bufio"

	"github.com/golang/glog"
	"github.com/mchestr/ethos-monitor/mining_monitor"
)

var (
	debug                  = flag.Bool("debug", false, "Used for debugging to set clients to READONLY mode")
	checkFailsBeforeReboot = flag.Int("check-fails", 2, "Number of failed checks before reboot, default 2")
	rebootFailsBeforePower = flag.Int("reboot-fails", 2, "Number of reboot fails before we toggle power on and off")
	statsInterval          = flag.Duration("stats-interval", 30*time.Second, "Interval to poll for statistics")
	stateInterval          = flag.Duration("state-interval", 3*time.Second, "Time in seconds to transition monitoring states")
	rebootInterval         = flag.Duration("reboot-interval", 5*time.Minute, "Time between successful reboots before attempting another")

	claymoreAddress  = flag.String("claymore-address", "", "Address for claymore remote management interface")
	claymorePassword = flag.String("claymore-password", "", "Password for claymore remote management interface")
	claymoreVersion  = flag.Float64("claymore-version", 10.2, "Claymore version")

	hashThreshold        = flag.String("hash-threshold", "<23000", "Threshold in kH/s per GPU if below will attempt reboot")
	powerThreshold       = flag.String("power-threshold", "", "Threshold in Watts for Rig")
	temperatureThreshold = flag.String("temp-threshold", "", "Threshold in degrees celsius for GPUs")
	fanPercentThreshold  = flag.String("fan-threshold", ">70", "Threshold in percent for GPUs")

	hs110PlugIp = flag.String("hs110plug-ip", "", "TPLink HS110 plug IP")

	emailEnabled  = flag.Bool("email-enabled", true, "Enable/Disable email flag")
	email         = flag.String("email", "", "Email to send from")
	emailHost     = flag.String("email-host", "", "Email Host, if set will send email on events")
	emailPassword = flag.String("email-password", "", "Email Pass")
	emailPort     = flag.Int("email-port", 25, "Email port, default 25")

	emailMaxInterval = flag.Int("email-max-interval", 5, "Max emails to send in email-timeout duration")
	emailTimeout     = flag.Duration("emailTimeout", 1*time.Hour, "Time between sending emails if maximum is reached")
)

func main() {
	flag.Parse()
	s := make(chan os.Signal, 1)
	in := make(chan string)
	signal.Notify(s, os.Interrupt)
	log.SetOutput(os.Stdout)

	var eventService *mining_monitor.EventService
	if *emailEnabled {
		es := mining_monitor.NewGMailService(*emailHost, *email, []string{*email}, *email, *emailPassword, *emailPort)
		es.SetMaxEmails(*emailMaxInterval, *emailTimeout)
		eventService = mining_monitor.NewEventServiceWithEmail(es)
	} else {
		eventService = mining_monitor.NewEventService()
	}

	ps := mining_monitor.NewHS110PowerService(*hs110PlugIp)
	c := mining_monitor.NewClaymoreClientWithPowerService(*claymoreAddress, *claymorePassword, *claymoreVersion, ps)
	c.SetReadOnly(*debug, true)

	m := mining_monitor.NewMonitor(eventService)

	hashThreshold, err := mining_monitor.NewHashRateThreshold(*hashThreshold, true, false)
	if err != nil {
		panic(err)
	}
	thresholds := []*mining_monitor.Threshold{hashThreshold}
	if *powerThreshold != "" {
		powerThreshold, err := mining_monitor.NewPowerThreshold(*powerThreshold, true, false)
		if err != nil {
			panic(err)
		}
		thresholds = append(thresholds, powerThreshold)
	}
	if *temperatureThreshold != "" {
		tempThreshold, err := mining_monitor.NewTemperatureThreshold(*temperatureThreshold, false, true)
		if err != nil {
			panic(err)
		}
		thresholds = append(thresholds, tempThreshold)
	}
	if *fanPercentThreshold != "" {
		fpThreshold, err := mining_monitor.NewFanPercentThreshold(*fanPercentThreshold, true, false)
		if err != nil {
			panic(err)
		}
		thresholds = append(thresholds, fpThreshold)
	}
	m.AddClient(c, mining_monitor.NewClientMonitorConfig(
		thresholds, *checkFailsBeforeReboot, *rebootFailsBeforePower,
		*rebootInterval, *statsInterval, *stateInterval,
	))

	m.Start()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input := scanner.Text()
			in <- input
		}
	}()

	glog.Info("Mining Monitor running\nCommands:\nstop|s - stop the monitoring\nresume|r - resume the monitoring\ndebug|d - enable debugging\n\n")
	for {
		select {
		case inputStr := <-in:
			switch inputStr {
			case "stop", "s":
				log.Printf("Stopping monitoring service...")
				m.Stop()
				log.Printf("Monitoring service stoppped")
			case "resume", "r":
				log.Printf("Starting monitoring service...")
				m.Start()
				log.Printf("Monitoring service started")
			case "debug", "d":
				log.Printf("Setting client to debug %t", !c.ReadOnly())
				c.SetReadOnly(!c.ReadOnly(), false)
			}
		case <-s:
			m.Stop()
			log.Println("Exitting Program.")
			time.Sleep(2 * time.Second)
			return
		}
	}
}

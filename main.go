package main

import (
	"flag"
	"time"

	"log"

	"os"

	"net/smtp"

	"fmt"

	"os/signal"

	"encoding/json"

	"github.com/mchestr/ethos-monitor/ethos_monitor"
	"github.com/oliveagle/jsonpath"
	"github.com/sausheong/hs1xxplug"
)

const (
	STARTING = iota
	POWERCYCLING
	RUNNING
	REBOOTING

	relayStatePath = "$.system.get_sysinfo.relay_state"
)

var (
	addr                   = flag.String("addr", "", "Address for claymore remote management interface")
	password               = flag.String("pass", "", "Password for claymore remote management interface")
	version                = flag.Float64("version", 10.2, "Claymore version")
	interval               = flag.Duration("i", 30*time.Second, "Interval to poll")
	rebootInterval         = flag.Duration("ri", 5*time.Minute, "Time between rebooting the machine if stats are not optimal")
	perGpuThreshold        = flag.Float64("t", 25000, "Threshold in kH/s per GPU if below will attempt reboot")
	failsBeforeReboot      = flag.Int("fails", 2, "Number of failed checks before reboot, default 2")
	rebootFailsBeforePower = flag.Int("reboot-fails", 2, "Number of reboot fails before we toggle power on and off")

	hs110plug = flag.String("plug", "", "TPLink HS110 plug IP")

	state = STARTING

	emailHost       = flag.String("ehost", "", "Email Host, if set will send email on events")
	emailUsername   = flag.String("euser", "", "Email User")
	emailPass       = flag.String("epass", "", "Email Pass")
	emailPort       = flag.Int("eport", 25, "Email port, default 25")
	email           = flag.String("email", "", "Email to send from")
	emailLogTimeoff = flag.Duration("emailLogTimeoff", 24*time.Hour, "Time between sending log emails")
)

type Email struct {
	Subject string
	Message string
}

func fmtErrors(errors []error) string {
	msg := ""
	for _, err := range errors {
		msg += err.Error() + "\r\n"
	}
	return msg
}

func sendEmail(subject, msg string) error {
	body := fmt.Sprintf(
		"To: %s\r\n"+
			"Subject: %s\r\n"+
			"\r\n"+
			"%s",
		*email, subject, msg,
	)
	auth := smtp.PlainAuth("", *emailUsername, *emailPass, *emailHost)
	if err := smtp.SendMail(fmt.Sprintf("%s:%d", *emailHost, *emailPort), auth, *email, []string{*email}, []byte(body)); err != nil {
		return err
	}
	log.Println("email sent successfully")
	return nil
}

func checkHashRateOptimal(stats *ethos_monitor.Statistics, threshold float64) []error {
	var errors []error
	for i, hash := range stats.MainGpuHashRate {
		if hash < threshold {
			errors = append(errors, fmt.Errorf("GPU %d has a lower than expected hashrate %0.2f<%0.2f", i, hash, *perGpuThreshold))
		}
	}
	return errors
}

func checkStats(c ethos_monitor.Client) []error {
	stats, err := c.Stats()
	if err != nil {
		return []error{fmt.Errorf("got error retrieving stats: %s", err)}
	}
	var errors []error
	if errs := checkHashRateOptimal(stats, *perGpuThreshold); len(errs) > 0 {
		errors = append(errors, errs...)
	}
	return errors
}

func validateArguments() error {
	if *emailHost != "" && *emailUsername == "" {
		return fmt.Errorf("-euser is required to send mail")
	}
	if *emailHost != "" && *emailPass == "" {
		return fmt.Errorf("-epass is required to send mail")
	}

	if *emailHost != "" && *email == "" {
		return fmt.Errorf("-email is required to send mail")
	}
	return nil
}

func messageProcessingRoutine(stopChan chan bool, emailChan chan Email, logChan chan string, errorChan chan error) {
	var logs []string
	var errors []error
	routineTicker := time.NewTicker(30 * time.Second)
	lastLogEmailSent := time.Now()

	defer func() {
		log.Printf("Message processing stopped")
	}()

	for {
		select {
		case email := <-emailChan:
			log.Printf("Sending email...")
			if err := sendEmail(email.Subject, email.Message); err != nil {
				log.Printf("Failed to send email: %s", err)
			} else {
				log.Printf("Email successfully sent")
			}
		case l := <-logChan:
			log.Println(l)
			logs = append(logs, l)
		case err := <-errorChan:
			log.Println("Error:", err)
			errors = append(errors, err)
		case <-routineTicker.C:
			if time.Now().Sub(lastLogEmailSent) > *emailLogTimeoff {
				var message string
				for index, l := range logs {
					message += fmt.Sprintf("Log %d: %s\r\n", index, l)
				}
				for index, err := range errors {
					message += fmt.Sprintf("Error %d: %s\r\n", index, err)
				}
				if message != "" {
					emailChan <- Email{Subject: "Errors and Logs Update", Message: message}
					lastLogEmailSent = time.Now()
				}
				logs = []string{}
				errors = []error{}
			}
		case <-stopChan:
			return
		}
	}
}

func getPlugState(plug hs1xxplug.Hs1xxPlug) (int, error) {
	info, err := plug.SystemInfo()
	if err != nil {
		return 0, err
	}

	var data interface{}
	if err := json.Unmarshal([]byte(info), &data); err != nil {
		return 0, fmt.Errorf("unable to unmarshal %s", info)
	}
	res, err := jsonpath.JsonPathLookup(data, relayStatePath)
	if err != nil {
		return 0, fmt.Errorf("unable to get relay_state from %s", info)
	}
	state := int(res.(float64))
	return state, nil
}

func main() {
	flag.Parse()
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	log.SetOutput(os.Stdout)

	emailChan := make(chan Email, 100)
	errorChan := make(chan error, 100)
	logChan := make(chan string, 100)
	stopChan := make(chan bool, 1)

	if err := validateArguments(); err != nil {
		panic(err)
	}

	go messageProcessingRoutine(stopChan, emailChan, logChan, errorChan)

	c := ethos_monitor.NewClaymoreClient(*addr, *password, *version)
	lastReboot := time.Now().Add(-10 * time.Minute)
	failedChecks := 0
	failedReboots := 0

	state = RUNNING
	// Check stats first, since the ticket won't tick for a full duration initially
	if errors := checkStats(c); errors != nil && len(errors) > 0 {
		failedChecks += 1
	}
	ticker := time.NewTicker(*interval)
	stateTicker := time.NewTicker(1 * time.Second)
	log.Println("Starting monitor...")
	var errors []error
	for {
		select {
		case <-stateTicker.C:
			if *hs110plug != "" && failedReboots > *rebootFailsBeforePower {
				if state != POWERCYCLING {
					log.Printf("transitioning to POWERCYCLING state...")
				}
				state = POWERCYCLING
			} else if failedChecks > *failsBeforeReboot && time.Now().Sub(lastReboot) > *rebootInterval {
				if state != REBOOTING {
					log.Printf("transitioning to REBOOTING state...")
				}
				state = REBOOTING
			} else {
				if state != RUNNING {
					log.Printf("transitioning to RUNNING state...")
				}
				state = RUNNING
			}
		case <-ticker.C:
			switch state {
			case RUNNING:
				if errors = checkStats(c); len(errors) > 0 {
					for _, err := range errors {
						errorChan <- err
					}
					failedChecks += 1
				} else {
					failedChecks = 0
				}
			case REBOOTING:
				logChan <- fmt.Sprintf("Attempting to reboot client %s...", c.IP())
				email := Email{}
				if err := c.Reboot(); err != nil {
					errorChan <- fmt.Errorf("failed to reboot client %s: %s", c.IP(), err)
					email.Subject = fmt.Sprintf("Failed to reboot client %s", c.IP())
					email.Message = fmt.Sprintf("Client was unable to be restarted due to error: %s", err)
					failedReboots += 1
				} else {
					log.Printf("client %s was rebooted successfully", c.IP())
					email.Subject = fmt.Sprintf("Successfully rebooted client %s", c.IP())
					email.Message = fmt.Sprintf("Client was restarted due to events: %s", fmtErrors(errors))
					failedChecks = 0
					failedReboots = 0
					lastReboot = time.Now()
					errors = []error{}
				}
				emailChan <- email
			case POWERCYCLING:
				plug := hs1xxplug.Hs1xxPlug{IPAddress: *hs110plug}
				plugState, err := getPlugState(plug)
				if err != nil {
					errorChan <- err
				}
				logChan <- fmt.Sprintf("failed to reboot system %d times, attempting to power cycle client %s...", failedReboots, c.IP())

				email := Email{}
				if plugState == 1 {
					if err := plug.TurnOff(); err != nil {
						errorChan <- fmt.Errorf("failed to turn hs110 plug off: %s", err)
						email.Subject = fmt.Sprintf("Failed to power cycle %s", c.IP())
						email.Message = fmt.Sprintf("Error turning power off: %s", err)
						emailChan <- email
						continue
					}
					logChan <- fmt.Sprintf("waiting 10 seconds before powering on client %s...", c.IP())
					time.Sleep(10 * time.Second)
				} else {
					logChan <- fmt.Sprintf("plug was found in an off state already, powering on...")
				}
				if err := plug.TurnOn(); err != nil {
					errorChan <- fmt.Errorf("failed to turn hs110 plug on")
					email.Subject = fmt.Sprintf("Failed to power cycle %s", c.IP())
					email.Message = fmt.Sprintf("Error turning power off: %s", err)
				} else {
					email.Subject = fmt.Sprintf("Successfully power cycled client %s", c.IP())
					email.Message = fmt.Sprintf("Client was successfully power cycled for reasons: %s", fmtErrors(errors))
					logChan <- fmt.Sprintf("successfully power cycled client %s", c.IP())
					failedChecks = 0
					failedReboots = 0
					lastReboot = time.Now()
					errors = []error{}
				}
				emailChan <- email
			}
		case <-s:
			stopChan <- true
			time.Sleep(500 * time.Millisecond)
			log.Println("Exitting Program.")
			return
		}
	}
}

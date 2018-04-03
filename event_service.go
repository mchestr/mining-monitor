package miningmonitor

import "github.com/golang/glog"

const (
	// LogType Event
	LogType = iota
	// ErrorType Event
	ErrorType
	// EmailType Event
	EmailType
)

// Event contains information of monitoring events
type Event struct {
	Type   int
	Client Client

	Subject string
	Message string
	Error   error
}

// NewLogEvent returns a new event for logging
func NewLogEvent(c Client, message string) Event {
	return Event{Client: c, Type: LogType, Message: message}
}

// NewEmailEvent returns an event that will trigger an email
func NewEmailEvent(c Client, subject, message string) Event {
	return Event{Client: c, Type: EmailType, Subject: subject, Message: message}
}

// NewErrorEvent will return a new error event that may or may not trigger an email depending on the client configuration
func NewErrorEvent(c Client, err error) Event {
	return Event{Client: c, Type: ErrorType, Error: err}
}

// EventService used to handle events within the monitoring services
type EventService struct {
	E            chan Event
	EmailService EmailService

	logs   []string
	errors []error
	stop   chan bool
}

// NewEventServiceWithEmail returns an Event Service that will send emails
func NewEventServiceWithEmail(es EmailService) *EventService {
	return &EventService{
		E:            make(chan Event, 100),
		EmailService: es,
		stop:         make(chan bool, 1),
	}
}

// NewEventService returns an Event Service with no email
func NewEventService() *EventService {
	return &EventService{
		E:    make(chan Event, 100),
		stop: make(chan bool, 1),
	}
}

// Start the EventService
func (es *EventService) Start() {
	for {
		select {
		case event := <-es.E:
			switch event.Type {
			case LogType:
				es.logs = append(es.logs, event.Message)
				glog.Infof("[%s]: %s", event.Client.IP(), event.Message)
			case ErrorType:
				es.errors = append(es.errors, event.Error)
				glog.Infof("[%s] Error: %s", event.Client.IP(), event.Error)
			case EmailType:
				if es.EmailService == nil {
					glog.Infof("email service not initialized, no email sent")
				} else {
					if err := es.EmailService.SendEmail(event.Subject, event.Message); err != nil {
						glog.Infof("unable to send email: %s", err)
					} else {
						glog.Infof("[%s]: successfully sent email", event.Client.IP())
					}
				}
			default:
				glog.Infof("[%s]: unknown event received %+v", event.Client.IP(), event)
			}
		case <-es.stop:
			glog.Infof("Event Service stopped")
			return
		}
	}
}

// Stop the EventService
func (es *EventService) Stop() {
	es.stop <- true
}

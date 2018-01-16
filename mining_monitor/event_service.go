package mining_monitor

import "log"

const (
	LogType = iota
	ErrorType
	EmailType
)

type Event struct {
	Type   int
	Client Client

	Subject string
	Message string
	Error   error
}

func NewLogEvent(c Client, message string) Event {
	return Event{Client: c, Type: LogType, Message: message}
}

func NewEmailEvent(c Client, subject, message string) Event {
	return Event{Client: c, Type: EmailType, Subject: subject, Message: message}
}

func NewErrorEvent(c Client, err error) Event {
	return Event{Client: c, Type: ErrorType, Error: err}
}

type EventService struct {
	E            chan Event
	EmailService EmailService

	logs   []string
	errors []error
	stop   chan bool
}

func NewEventServiceWithEmail(es EmailService) *EventService {
	return &EventService{
		E:            make(chan Event, 100),
		EmailService: es,
		stop:         make(chan bool, 1),
	}
}

func (es *EventService) Start() {
	for {
		select {
		case event := <-es.E:
			switch event.Type {
			case LogType:
				es.logs = append(es.logs, event.Message)
				log.Printf("[%s]: %s", event.Client.IP(), event.Message)
			case ErrorType:
				es.errors = append(es.errors, event.Error)
				log.Printf("[%s] Error: %s", event.Client.IP(), event.Error)
			case EmailType:
				if es.EmailService == nil {
					log.Printf("email service not initialized, no email sent")
				} else {
					if err := es.EmailService.SendEmail(event.Subject, event.Message); err != nil {
						log.Printf("unable to send email: %s", err)
					} else {
						log.Printf("[%s]: successfully sent email", event.Client.IP())
					}
				}
			default:
				log.Printf("[%s]: unknown event recieved %+v", event.Client.IP(), event)
			}
		case <-es.stop:
			log.Printf("Event Service stopped")
			return
		}
	}
}

func (es *EventService) Stop() {
	es.stop <- true
}

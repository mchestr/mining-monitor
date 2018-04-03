package miningmonitor

func fmtErrors(errors []error) string {
	msg := ""
	for _, err := range errors {
		msg += err.Error() + "\r\n"
	}
	return msg
}

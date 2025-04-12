package notification

import (
	"fmt"
	"net/smtp"

	"github.com/raykavin/backnrun/pkg/core"
	log "github.com/sirupsen/logrus"
)

// Mail handles email notifications for the application
type Mail struct {
	auth              smtp.Auth
	smtpServerPort    int
	smtpServerAddress string
	to                string
	from              string
}

// MailParams contains all parameters needed to initialize a Mail instance
type MailParams struct {
	SMTPServerPort    int
	SMTPServerAddress string
	To                string
	From              string
	Password          string
}

// NewMail creates a new Mail instance with the provided parameters
func NewMail(params MailParams) Mail {
	return Mail{
		from:              params.From,
		to:                params.To,
		smtpServerPort:    params.SMTPServerPort,
		smtpServerAddress: params.SMTPServerAddress,
		auth: smtp.PlainAuth(
			"",
			params.From,
			params.Password,
			params.SMTPServerAddress,
		),
	}
}

// Notify sends an email notification with the given text
func (m Mail) Notify(text string) {
	serverAddress := fmt.Sprintf("%s:%d", m.smtpServerAddress, m.smtpServerPort)

	message := fmt.Sprintf(
		`To: "User" <%s>
From: "NinjaBot" <%s>
%s`,
		m.to,
		m.from,
		text,
	)

	err := smtp.SendMail(
		serverAddress,
		m.auth,
		m.from,
		[]string{m.to},
		[]byte(message),
	)

	if err != nil {
		log.WithError(err).Error("notification/mail: failed to send email")
	}
}

// OnOrder sends an order notification based on its status
func (m Mail) OnOrder(order core.Order) {
	var title string

	switch order.Status {
	case core.OrderStatusTypeFilled:
		title = fmt.Sprintf("✅ ORDER FILLED - %s", order.Pair)
	case core.OrderStatusTypeNew:
		title = fmt.Sprintf("🆕 NEW ORDER - %s", order.Pair)
	case core.OrderStatusTypeCanceled, core.OrderStatusTypeRejected:
		title = fmt.Sprintf("❌ ORDER CANCELED / REJECTED - %s", order.Pair)
	}

	message := fmt.Sprintf("Subject: %s\nOrder %s", title, order)
	m.Notify(message)
}

// OnError sends an error notification
func (m Mail) OnError(err error) {
	message := fmt.Sprintf("Subject: 🛑 ERROR\nError %s", err)
	m.Notify(message)
}

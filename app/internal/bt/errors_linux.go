package bt

import (
	"errors"

	"github.com/godbus/dbus/v5"
)

// Erros do BlueZ que o código trata como sucesso (já feito / já buscando).
var (
	errExists     = errors.New("já pareado")
	errConnected  = errors.New("já conectado")
	errInProgress = errors.New("já em andamento")
)

// bluezErrors: nome do erro do BlueZ -> mensagem pra quem está no sofá.
var bluezErrors = map[string]error{
	"org.bluez.Error.AlreadyExists":            errExists,
	"org.bluez.Error.AlreadyConnected":         errConnected,
	"org.bluez.Error.InProgress":               errInProgress,
	"org.bluez.Error.NotReady":                 errOff,
	"org.bluez.Error.DoesNotExist":             errNotFound,
	"org.freedesktop.DBus.Error.UnknownObject": errNotFound,
	"org.bluez.Error.AuthenticationFailed":     errors.New("o controle recusou o pareamento: tente de novo"),
	"org.bluez.Error.AuthenticationCanceled":   errors.New("pareamento cancelado"),
	"org.bluez.Error.AuthenticationRejected":   errors.New("o controle recusou o pareamento: tente de novo"),
	"org.bluez.Error.AuthenticationTimeout":    errors.New("o controle demorou demais: deixe-o em modo de pareamento e tente de novo"),
	"org.bluez.Error.ConnectionAttemptFailed":  errors.New("não deu pra conectar: o controle está ligado e perto?"),
	"org.bluez.Error.NotAuthorized":            errors.New("o sistema não deixou o app mexer no Bluetooth"),
	"org.freedesktop.DBus.Error.AccessDenied":  errors.New("o sistema não deixou o app mexer no Bluetooth"),
	"org.freedesktop.DBus.Error.NoReply":       errors.New("o Bluetooth não respondeu a tempo"),
}

// translate troca o erro do D-Bus por um em português; desconhecido vira
// "falhou" com o detalhe do BlueZ.
func translate(err error) error {
	if err == nil {
		return nil
	}
	var de dbus.Error
	if !errors.As(err, &de) {
		var pe *dbus.Error
		if !errors.As(err, &pe) {
			return err
		}
		de = *pe
	}
	if known, ok := bluezErrors[de.Name]; ok {
		return known
	}
	return errors.New("o Bluetooth falhou: " + de.Error())
}

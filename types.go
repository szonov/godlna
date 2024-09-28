package upnp

type ErrorHandlerFunc func(err error, caller string)
type InfoHandlerFunc func(msg string, caller string)

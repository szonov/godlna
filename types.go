package upnp

type ErrorHandlerFunc func(err error, identity string)
type InfoHandlerFunc func(msg string, identity string)

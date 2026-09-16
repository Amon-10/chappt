// Package protocol defines the small, line-oriented protocol shared by the
// Chappt client and server.
package protocol

const (
	// Welcome confirms that the server accepted a username.
	Welcome = "OK welcome to Chappt"
	// ErrorPrefix marks a username rejection and is followed by its reason.
	ErrorPrefix = "ERR "
	// SystemPrefix marks join, leave, and validation notices.
	SystemPrefix = "# "
)

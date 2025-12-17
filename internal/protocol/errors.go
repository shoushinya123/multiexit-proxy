package protocol

import "errors"

var (
	ErrInvalidMessage = errors.New("invalid message format")
	ErrInvalidVersion = errors.New("invalid protocol version")
	ErrAuthFailed     = errors.New("authentication failed")
	ErrEncryption     = errors.New("encryption error")
	ErrDecryption     = errors.New("decryption error")
)



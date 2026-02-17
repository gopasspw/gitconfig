package gitconfig

import "errors"

var (
	// ErrInvalidKey indicates a config key missing section or key name.
	ErrInvalidKey = errors.New("invalid key")
	// ErrWorkdirNotSet indicates a workdir is required but not configured.
	ErrWorkdirNotSet = errors.New("no workdir set")
	// ErrCreateConfigDir indicates a config directory could not be created.
	ErrCreateConfigDir = errors.New("failed to create config directory")
	// ErrWriteConfig indicates a config file could not be written.
	ErrWriteConfig = errors.New("failed to write config")
)

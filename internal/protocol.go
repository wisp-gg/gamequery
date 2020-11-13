package internal

import (
	"github.com/wisp-gg/gamequery/api"
)

type Protocol interface {
	Name() string
	Aliases() []string
	DefaultPort() uint16
	Priority() uint16
	Network() string

	Execute(helper NetworkHelper) (api.Response, error)
}

package backnrun

import (
	"github.com/raykavin/backnrun/pkg/logger/zerolog"
)

func init() {
	log, err := zerolog.New("debug", "2006-01-02 15:04:05", true, false)
	if err != nil {
		panic(err)
	}

	DefaultLog = zerolog.NewAdapter(log.Logger)
}

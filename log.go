package seeker

import (
	"fmt"

	l4g "github.com/alecthomas/log4go"
	seelog "github.com/cihub/seelog"
)

const (
	filename = "flw.log"
)

func main() {
	fmt.Println("....")

	logger, err := seelog.LoggerFromConfigAsFile("seelog.xml")

	if err != nil {
		seelog.Critical("err parsing config log file", err)
		return
	}
	seelog.ReplaceLogger(logger)

	seelog.Error("seelog error")
	seelog.Info("seelog info")
	seelog.Debug("seelog debug")
	defer seelog.Flush()
	//
	l4g.LoadConfiguration("log4go.xml")

	// And now we're ready!
	l4g.Finest("This will only go to those of you really cool UDP kids!  If you change enabled=true.")
	l4g.Debug("Oh no!  %d + %d = %d!", 2, 2, 2+2)
	l4g.Info("About that time, eh chaps?")
	//	for {
	//	}
}

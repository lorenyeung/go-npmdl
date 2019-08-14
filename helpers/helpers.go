package helpers

import "log"

//Check logger for errors
func Check(e error, panic bool, logs string) {
	if e != nil && panic {
		log.Panicf("%s failed with error:%s\n", logs, e)
	}
	if e != nil && !panic {
		log.Printf("%s failed with error:%s\n", logs, e)
	}
}

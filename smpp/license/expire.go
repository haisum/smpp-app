package license

import (
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
)

// CheckExpiry expires after 1st july
// This function must be called in a thread otherwise it will block eternally.
func CheckExpiry() {
	msgPrinted := false
	for {
		now := time.Now().UTC()
		if now.Unix() > time.Date(2016, time.August, 25, 0, 0, 0, 0, now.Location()).Unix() {
			if !msgPrinted {
				log.Error("License has expired. Please purchase full version to continue usage.")
				msgPrinted = true
			}
			os.Exit(2)
		} else {
			if !msgPrinted {
				log.Info("This license is valid till 1st July.")
				msgPrinted = true
			}
		}
		time.Sleep(time.Minute * 5)
	}
}

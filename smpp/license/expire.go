package license

import (
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
)

// CheckExpiry expires after 1st july
// This function must be called in a thread otherwise it will block eternally.
func CheckExpiry() {
	for {
		now := time.Now().UTC()
		if now.Unix() > time.Date(2016, time.July, 1, 0, 0, 0, 0, now.Location()).Unix() {
			log.Error("License has expired. Please purchase full version to continue usage.")
			os.Exit(2)
		} else {
			log.Info("This license is valid till 1st July.")
		}
		time.Sleep(time.Minute * 5)
	}
}

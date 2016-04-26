package fresh

import (
	"bitbucket.org/codefreak/hsmpp/smpp"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
	"time"
)

func tconfig(s *r.Session, dbname string) error {
	_, err := r.DB(dbname).TableCreate("Config").RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
			"name":  dbname,
			"table": "Config",
		}).Error("Error occured in creating table.")
		return err
	}
	var c smpp.Config
	err = json.Unmarshal([]byte(`{
    "ConnGroups": [
        {
          "Name": "Default",
          "Conns" :  [
                {
                    "ID": "du-1",
                    "URL": "192.168.0.105:2775",
                    "User": "smppclient1",
                    "Passwd": "password",
                    "Pfxs": [
                        "+97105",
                        "+97106"
                    ],
                    "Size": 5,
                    "Time": 1,
                    "Fields" : {
                        "ServiceType":          "",
                        "SourceAddrTON":        0,
                        "SourceAddrNPI":        0,
                        "DestAddrTON":          0,
                        "DestAddrNPI":          0,
                        "ESMClass":             0,
                        "ProtocolID":           0,
                        "PriorityFlag" :        0,
                        "ScheduleDeliveryTime" : "",
                        "ReplaceIfPresentFlag" : 0,
                        "SMDefaultMsgID"       :0
                    }
                },
                {
                    "ID": "du-2",
                    "URL": "192.168.0.105:2775",
                    "User": "smppclient2",
                    "Passwd": "password",
                    "Pfxs": [
                        "+97107",
                        "+97108"
                    ],
                    "Size": 5,
                    "Time": 1
                }
            ],
          "DefaultPfx": "+97105"
        },
        {
          "Name" : "AADC",
          "Conns" :  [
                {
                    "ID": "du-2",
                    "URL": "192.168.0.105:2775",
                    "User": "smppclient2",
                    "Passwd": "password",
                    "Pfxs": [
                        "+97107",
                        "+97108"
                    ],
                    "Size": 5,
                    "Time": 1
                }
            ],
          "DefaultPfx": "+97105"
        }
	  ]
	}`), &c)
	if err != nil {
		log.WithError(err).Error("Couldn't load json in config struct.")
		return err
	}
	_, err = r.DB(dbname).Table("Config").Insert(c).RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
			"name":  dbname,
			"table": "Config",
		}).Error("Error occured in inserting config in table.")
	}
	return err
}

func ttoken(s *r.Session, dbname string) error {
	_, err := r.DB(dbname).TableCreate("Token").RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
			"name":  dbname,
			"table": "Token",
		}).Error("Error occured in creating table.")
		return err
	}
	err = createIndexes(s, dbname, "Token", []string{"Username", "Token"})
	if err != nil {
		log.WithError(err).Error("Couldn't create indexes.")
		return err
	}
	return err
}

func tnumfile(s *r.Session, dbname string) error {
	_, err := r.DB(dbname).TableCreate("NumFile").RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
			"name":  dbname,
			"table": "NumFile",
		}).Error("Error occured in creating table.")
		return err
	}
	err = createIndexes(s, dbname, "NumFile", []string{
		"Username",
		"LocalName",
		"UserId",
		"SubmittedAt",
		"Type",
		"Name",
		"Deleted",
	})
	if err != nil {
		log.WithError(err).Error("Couldn't create indexes.")
		return err
	}
	return err
}

func tmessage(s *r.Session, dbname string) error {
	_, err := r.DB(dbname).TableCreate("Message").RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
			"name":  dbname,
			"table": "Message",
		}).Error("Error occured in creating table.")
		return err
	}
	err = createIndexes(s, dbname, "Message", []string{
		"Username",
		"RespId",
		"ConnectionGroup",
		"Connection",
		"Enc",
		"Dst",
		"Src",
		"QueuedBefore",
		"QueuedAfter",
		"SubmittedBefore",
		"SubmittedAfter",
		"DeliveredBefore",
		"DeliveredAfter",
		"CampaignId",
		"Status",
		"Error",
	})
	if err != nil {
		log.WithError(err).Error("Couldn't create indexes.")
		return err
	}
	return err
}

func tuser(s *r.Session, dbname string) error {
	_, err := r.DB(dbname).TableCreate("User").RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
			"name":  dbname,
			"table": "User",
		}).Error("Error occured in creating table.")
		return err
	}
	err = createIndexes(s, dbname, "User", []string{
		"Username",
		"RegisteredAt",
		"ConnectionGroup",
		"Permissions",
	})
	if err != nil {
		log.WithError(err).Error("Couldn't create indexes.")
		return err
	}
	u := struct {
		Name            string
		Password        string
		Email           string
		Username        string
		NightStartAt    string
		NightEndAt      string
		ConnectionGroup string
		Permissions     []smpp.Permission
		RegisteredAt    int64
	}{
		Name:            "Admin",
		Password:        "admin123",
		Email:           "admin@localhost.dev",
		Username:        "admin",
		NightEndAt:      "00:00AM",
		NightStartAt:    "00:00AM",
		ConnectionGroup: "Default",
		Permissions:     smpp.GetPermissions(),
		RegisteredAt:    time.Now().Unix(),
	}
	u.Password, err = hash(u.Password)
	if err != nil {
		return err
	}
	err = r.DB(dbname).Table("User").Insert(u).Exec(s)
	if err != nil {
		return err
	}
	return nil
}

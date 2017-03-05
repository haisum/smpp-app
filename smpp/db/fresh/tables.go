package fresh

import (
	"encoding/json"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/user"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
)

func tconfig(s r.QueryExecutor, dbname string) error {
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
                    },
                    "Receiver": ""
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
                    "Time": 1,
                    "Receiver": ""
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
                    "Time": 1,
                    "Receiver": ""
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

func tuser(s r.QueryExecutor, dbname string) error {
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
		ConnectionGroup string
		Permissions     []user.Permission
		RegisteredAt    int64
	}{
		Name:            "Admin",
		Password:        "admin123",
		Email:           "admin@localhost.dev",
		Username:        "admin",
		ConnectionGroup: "Default",
		Permissions:     user.GetPermissions(),
		RegisteredAt:    time.Now().UTC().Unix(),
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

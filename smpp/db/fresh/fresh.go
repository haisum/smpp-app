package fresh

import (
	"bitbucket.com/codefreak/hsmpp/smpp"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
	"golang.org/x/crypto/bcrypt"
	"time"
)

func Create(s *r.Session, dbname string) error {
	w, err := r.DBCreate(dbname).RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"name": dbname,
		}).Error("Error occured in creating database.")
		return err
	}
	if w.DBsCreated != 1 {
		log.WithFields(log.Fields{
			"DBsCreated":    w.DBsCreated,
			"name":          dbname,
			"WriteResponse": jsonPrint(w),
		}).Error("Error occured in creating database.")
		return fmt.Errorf("Error occured in creating database.")
	}
	err = tuser(s, dbname)
	if err != nil {
		return err
	}
	err = ttoken(s, dbname)
	if err != nil {
		return err
	}
	err = tconfig(s, dbname)
	if err != nil {
		return err
	}
	return nil
}

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
    "AmqpURL": "amqp://guest:guest@localhost:5672/",
    "HTTPSPort": 8443,
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
	err = r.DB(dbname).Table("Token").IndexCreate("Username").Exec(s)
	if err != nil {
		log.WithError(err).Error("Couldn't create Username index.")
		return err
	}
	err = r.DB(dbname).Table("User").IndexCreate("Token").Exec(s)
	if err != nil {
		log.WithError(err).Error("Couldn't create Token index.")
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
	err = r.DB(dbname).Table("User").IndexCreate("Username").Exec(s)
	if err != nil {
		log.WithError(err).Error("Couldn't create Username index.")
		return err
	}
	err = r.DB(dbname).Table("User").IndexCreate("RegisteredAt").Exec(s)
	if err != nil {
		log.WithError(err).Error("Couldn't create RegisteredAt index.")
		return err
	}
	err = r.DB(dbname).Table("User").IndexCreate("ConnectionGroup").Exec(s)
	if err != nil {
		log.WithError(err).Error("Couldn't create ConnectionGroup index.")
		return err
	}
	err = r.DB(dbname).Table("User").IndexCreate("Permissions").Exec(s)
	if err != nil {
		log.WithError(err).Error("Couldn't create Permissions index.")
		return err
	}
	u := struct {
		Name            string
		Password        string
		Username        string
		NightStartAt    string
		NightEndAt      string
		ConnectionGroup string
		Permissions     []smpp.Permission
		RegisteredAt    int64
	}{
		Name:            "Admin",
		Password:        "admin123",
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

func hash(pass string) (string, error) {
	password := []byte(pass)
	// Hashing the password with the default cost of 10
	hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		log.WithFields(log.Fields{
			"err":      err,
			"password": pass,
		}).Error("Couldn't hash password")
		return "", err
	}
	return string(hashedPassword[:]), nil
}

func Drop(s *r.Session, name string) error {
	w, err := r.DBDrop(name).RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"name": name,
		}).Error("Error occured in droping database.")
		return err
	}
	if w.DBsDropped != 1 {
		log.WithFields(log.Fields{
			"DBsDropped":    w.DBsDropped,
			"name":          name,
			"WriteResponse": jsonPrint(w),
		}).Error("Error occured in dropping database.")
		return fmt.Errorf("Error occured in dropping database.")
	}
	return nil
}

func Exists(s *r.Session, name string) bool {
	cur, err := r.DBList().Run(s)
	if err != nil {
		log.WithError(err).Fatal("Couldn't get database list.")
		return false
	}
	var dbs []string
	cur.All(&dbs)
	for _, db := range dbs {
		if db == name {
			return true
		}
	}
	return false
}

func jsonPrint(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b[:])
}

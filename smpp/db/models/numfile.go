package models

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"fmt"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// NumFile represents file uploaded to system for saving
// files with numbers
type NumFile struct {
	Id          string `gorethink:"id,omitempty"`
	Name        string
	Description string
	LocalName   string
	Username    string
	UserId      string
	SubmittedAt int64
	Deleted     bool
	Type        NumFileType
}

//NumFileType represents type of file we're uploading
//can be excel/csv etc.
type NumFileType string

const (
	NumFileCSV  NumFileType = ".csv"
	NumFileTxt              = ".txt"
	NumFileXLS              = ".xls"
	NumFileXLSX             = ".xlsx"
	MaxFileSize int64       = 5 * 1024 * 1024
)

var (
	NumFilePath string = "./files"
)

// NumFileCriteria represents filters we can give to GetFiles method.
type NumFileCriteria struct {
	Id              string
	Username        string
	LocalName       string
	UserId          string
	SubmittedAfter  int64
	SubmittedBefore int64
	Type            NumFileType
	Name            string
	Deleted         bool
	OrderByKey      string
	OrderByDir      string
	From            string
	PerPage         int
}

// Delete marks Deleted=true for a NumFile
func (f *NumFile) Delete() error {
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return err
	}
	f.Deleted = true
	err = r.DB(db.DBName).Table("NumFile").Get(f.Id).Update(f).Exec(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB(db.DBName).Table("NumFile").Get(f.Id).Update(f).String(),
		}).Error("Error in updating file.")
		return err
	}
	return nil
}

// GetNumFiles filters files based on criteria
func GetNumFiles(c NumFileCriteria) ([]NumFile, error) {
	var f []NumFile
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return f, err
	}
	t := r.DB(db.DBName).Table("NumFile")

	var from interface{}
	if c.From != "" {
		if c.OrderByKey == "SubmittedAt" {
			from, err = strconv.ParseInt(c.From, 10, 64)
			if err != nil {
				return f, fmt.Errorf("Invalid value for from: %s", from)
			}
		} else {
			from = c.From
		}
	}
	t = orderBy(c.OrderByKey, c.OrderByDir, from, t)
	// keep between before Eq
	betweenFields := map[string]map[string]int64{
		"SubmittedAt": {
			"after":  c.SubmittedAfter,
			"before": c.SubmittedBefore,
		},
	}
	t = filterBetweenInt(betweenFields, t)
	strFields := map[string]string{
		"Id":        c.Id,
		"LocalName": c.LocalName,
		"Username":  c.Username,
		"UserId":    c.UserId,
		"Type":      string(c.Type),
		"Name":      c.Name,
	}
	t = filterEqStr(strFields, t)
	t = t.Filter(map[string]bool{"Deleted": c.Deleted})
	if c.OrderByKey == "" {
		c.OrderByKey = "SubmittedAt"
	}
	t = t.Limit(c.PerPage)
	log.WithFields(log.Fields{"query": t.String(), "crtieria": c}).Info("Running query.")
	cur, err := t.Run(s)
	if err != nil {
		log.WithError(err).Error("Couldn't run query.")
		return f, err
	}
	defer cur.Close()
	err = cur.All(&f)
	if err != nil {
		log.WithError(err).Error("Couldn't load files.")
	}
	return f, err
}

// Save saves a message struct in Message table
func (nf *NumFile) Save(name string, f multipart.File) (string, error) {
	var id string
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return id, err
	}
	fileType := NumFileType(filepath.Ext(strings.ToLower(name)))
	if fileType != NumFileCSV && fileType != NumFileTxt {
		return id, fmt.Errorf("Only csv and txt extensions are allowed Given file %s has extension %s.", name, fileType)
	}
	nf.Type = fileType
	nf.Name = name
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return id, fmt.Errorf("Couldn't read file.")
	}
	if http.DetectContentType(b) != "text/plain; charset=utf-8" {
		return id, fmt.Errorf("File doesn't seem to be a text file.")
	}
	numfilePath := fmt.Sprintf("%s/%s", NumFilePath, nf.UserId)
	err = os.MkdirAll(numfilePath, 0711)
	if err != nil {
		return id, fmt.Errorf("Couldn't create directory %s", numfilePath)
	}
	nf.LocalName = secureRandomAlphaString(20)
	err = ioutil.WriteFile(fmt.Sprintf("%s/%s", numfilePath, nf.LocalName), b, 0600)
	if err != nil {
		return id, fmt.Errorf("Couldn't write file to disk at path %s.", fmt.Sprintf("%s/%s", numfilePath, nf.LocalName))
	}
	_, err = nf.ToNumbers()
	if err != nil {
		log.WithError(err).Error("Couldn't read numbers from file.")
		return id, err
	}
	resp, err := r.DB(db.DBName).Table("NumFile").Insert(nf).RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB(db.DBName).Table("NumFile").Insert(nf).String(),
		}).Error("Error in inserting file in db.")
		return id, err
	}
	id = resp.GeneratedKeys[0]
	return id, nil
}

func (nf *NumFile) ToNumbers() ([]string, error) {
	var nums []string
	numfilePath := fmt.Sprintf("%s/%s", NumFilePath, nf.UserId)
	b, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", numfilePath, nf.LocalName))
	if err != nil {
		return nums, err
	}
	if nf.Type == NumFileCSV || nf.Type == NumFileTxt {
		nums = strings.Split(string(b[:]), ",")
		for i, num := range nums {
			num = strings.Trim(num, "\t\n\v\f\r \u0085\u00a0")
			if len(num) > 15 || len(num) < 5 {
				return nums, fmt.Errorf("Entry number %d in file %s is invalid. Number must be greater than 5 characters and lesser than 16. Please fix it and retry.", i+1, nf.Name)
			}
			nums = append(nums, num)
		}
	} else {
		return nums, fmt.Errorf("This file type isn't supported yet.")
	}
	if len(nums) < 1 {
		return nums, fmt.Errorf("No Numbers given in file.")
	}
	return nums, nil
}

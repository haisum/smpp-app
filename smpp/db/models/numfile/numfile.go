package numfile

import (
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
	"github.com/tealeg/xlsx"
)

// NumFile represents file uploaded to system for saving
// files with numbers
type NumFile struct {
	ID          string `gorethink:"id,omitempty"`
	Name        string
	Description string
	LocalName   string
	Username    string
	SubmittedAt int64
	Deleted     bool
	Type        Type
}

//Type represents type of file we're uploading
//can be excel/csv etc.
type Type string

func (n *Type) Scan(nf interface{}) error {
	*n = Type(fmt.Sprintf("%s", nf))
	return nil
}

const (
	//CSV is text file with .csv extension. This file should have comma separated numbers
	CSV Type = ".csv"
	//TXT is text file with .txt extension. This file should have comma separated numbers
	TXT = ".txt"
	//XLSX is excel file with .xlsx extension. These files should follow following structure:
	// -----------------------------------------
	// Destination | Param1 | Param2 | ..ParamN |
	// ------------------------------------------
	// 02398232390 | hello  | World  |  ValN    |
	//-------------------------------------------
	// First header must be Destination and firs cell value will be used as destination number
	// Rest of cells will be replacement values in message. A message with text "{{Param1}} {{Param2}} how are you" will become "hello World how are you"
	XLSX = ".xlsx"
	// MaxFileSize is maximum file size in bytes
	MaxFileSize int64 = 5 * 1024 * 1024
)

//Row represents one single Row in excel or csv file
type Row struct {
	Destination string
	Params      map[string]string
}

var (
	//Path is folder relative to path where httpserver binary is, we'll save all files here
	Path = "./files"
)

// Criteria represents filters we can give to GetFiles method.
type Criteria struct {
	ID              string
	Username        string
	UserID          string
	SubmittedAfter  int64
	SubmittedBefore int64
	Type            Type
	Name            string
	Deleted         bool
	OrderByKey      string
	OrderByDir      string
	From            string
	PerPage         int
}

// Delete marks Deleted=true for a NumFile
func (nf *NumFile) Delete() error {
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return err
	}
	nf.Deleted = true
	err = r.DB(db.DBName).Table("NumFile").Get(nf.ID).Update(nf).Exec(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB(db.DBName).Table("NumFile").Get(nf.ID).Update(nf).String(),
		}).Error("Error in updating file.")
		return err
	}
	return nil
}

// List filters files based on criteria
func List(c Criteria) ([]NumFile, error) {
	var (
		f          []NumFile
		indexUsed  bool
		filterUsed bool
	)
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return f, err
	}
	t := r.DB(db.DBName).Table("NumFile")

	if c.ID != "" {
		t = t.Get(c.ID)
		cur, errR := t.Run(s)
		if errR != nil {
			log.WithError(errR).Error("Couldn't run query.")
			return f, errR
		}
		defer cur.Close()
		errR = cur.All(&f)
		if err != nil {
			log.WithError(errR).Error("Couldn't load files.")
		}
		return f, errR
	}

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
	if from != nil || c.SubmittedAfter+c.SubmittedBefore != 0 {
		indexUsed = true
	}
	if c.OrderByKey == "" {
		c.OrderByKey = "SubmittedAt"
	}
	if !indexUsed {
		if c.Username != "" {
			if c.OrderByKey == SubmittedAt && !indexUsed {
				t = t.Between([]interface{}{c.Username, r.MinVal}, []interface{}{c.Username, r.MaxVal}, r.BetweenOpts{
					Index: "Username_SubmittedAt",
				})
				c.OrderByKey = "Username_SubmittedAt"
			} else {
				t = t.GetAllByIndex("Username", c.Username)
				indexUsed = true
			}
			c.Username = ""
		}
	}
	// keep between before Eq
	betweenFields := map[string]map[string]int64{
		"SubmittedAt": {
			"after":  c.SubmittedAfter,
			"before": c.SubmittedBefore,
		},
	}
	t, filterUsed = filterBetweenInt(betweenFields, t)
	strFields := map[string]string{
		"Username": c.Username,
		"UserID":   c.UserID,
		"Type":     string(c.Type),
		"Name":     c.Name,
	}
	var filtered bool
	t, filtered = filterEqStr(strFields, t)
	filterUsed = filterUsed || filtered
	t = orderBy(c.OrderByKey, c.OrderByDir, from, t, indexUsed, filterUsed)
	t = t.Filter(map[string]bool{"Deleted": c.Deleted})
	if c.PerPage == 0 {
		c.PerPage = 100
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
	fileType := Type(filepath.Ext(strings.ToLower(name)))
	if fileType != CSV && fileType != TXT && fileType != XLSX {
		return id, fmt.Errorf("Only csv, TXT and xlsx extensions are allowed Given file %s has extension %s.", name, fileType)
	}
	nf.Type = fileType
	nf.Name = name
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return id, fmt.Errorf("Couldn't read file.")
	}
	if http.DetectContentType(b) != "text/plain; charset=utf-8" && http.DetectContentType(b) != "application/zip" {
		return id, fmt.Errorf("File doesn't seem to be a text or excel file.")
	}
	numfilePath := fmt.Sprintf("%s/%s", Path, nf.UserID)
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

// NumbersFromString makes a Row list from comma separated numbers
func NumbersFromString(numbers string) []Row {
	var nums []Row
	if numbers == "" {
		return nums
	}
	parts := strings.Split(numbers, ",")
	for _, num := range parts {
		nums = append(nums, Row{
			Destination: num,
		})
	}
	return nums
}

// ToNumbers reads a csv or xlsx file and returns array of Row with Destination and Params map
func (nf *NumFile) ToNumbers() ([]Row, error) {
	var nums []Row
	nummap := make(map[string]Row) // used for unique numbers
	numfilePath := fmt.Sprintf("%s/%s/%s", Path, nf.UserID, nf.LocalName)
	b, err := ioutil.ReadFile(numfilePath)
	if err != nil {
		return nums, err
	}
	if nf.Type == CSV || nf.Type == TXT {
		for i, num := range strings.Split(string(b[:]), ",") {
			num = strings.Trim(num, "\t\n\v\f\r \u0085\u00a0")
			if len(num) > 15 || len(num) < 5 {
				return nums, fmt.Errorf("Entry number %d in file %s is invalid. Number must be greater than 5 characters and lesser than 16. Please fix it and retry.", i+1, nf.Name)
			}
			nummap[num] = Row{Destination: num}
		}
	} else if nf.Type == XLSX {
		xlFile, err := xlsx.OpenBinary(b)
		if err != nil {
			return nums, err
		}
		if len(xlFile.Sheets) != 1 {
			return nums, fmt.Errorf("xslx file should contain exactly one sheet")
		}
		if len(xlFile.Sheets[0].Rows) < 2 {
			return nums, fmt.Errorf("xslx file is empty")
		}
		if len(xlFile.Sheets[0].Rows[0].Cells) == 0 || xlFile.Sheets[0].Rows[0].Cells[0].Value != "Destination" {
			return nums, fmt.Errorf("First cell of excel sheet must be Destination header")
		}
		var keys []string
		for _, cell := range xlFile.Sheets[0].Rows[0].Cells {
			keys = append(keys, cell.Value)
		}
		for i := 1; i < len(xlFile.Sheets[0].Rows); i++ {
			if len(xlFile.Sheets[0].Rows[i].Cells) < 1 {
				return nums, fmt.Errorf("Row number %d doesn't have any value.", i+1)
			}
			num := xlFile.Sheets[0].Rows[i].Cells[0].Value
			num = strings.Trim(num, "\t\n\v\f\r \u0085\u00a0")
			if len(num) > 15 || len(num) < 5 {
				return nums, fmt.Errorf("Row number %d in file %s is invalid. Number must be greater than 5 characters and lesser than 16. Please fix it and retry.", i+1, nf.Name)
			}
			nr := Row{
				Destination: num,
				Params:      map[string]string{},
			}
			if len(xlFile.Sheets[0].Rows[i].Cells) < len(keys) {
				return nums, fmt.Errorf("Row number %d has blank values for some parameters.", i)
			}
			for j := 1; j < len(keys) && j < len(xlFile.Sheets[0].Rows[i].Cells); j++ {
				val := xlFile.Sheets[0].Rows[i].Cells[j].Value
				val = strings.Trim(val, "\t\n\v\f\r \u0085\u00a0")
				if val == "" {
					return nums, fmt.Errorf("Row number %d contains no value at cell number %d.", i, j)
				}
				nr.Params[keys[j]] = val
			}
			nummap[nr.Destination] = nr
		}
	} else {
		return nums, fmt.Errorf("This file type isn't supported yet.")
	}
	if len(nummap) < 1 {
		return nums, fmt.Errorf("No Numbers given in file.")
	}
	for _, v := range nummap {
		nums = append(nums, v)
	}
	return nums, nil
}

package models

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
	"gopkg.in/doug-martin/goqu.v3"
	"bitbucket.org/codefreak/hsmpp/smpp/stringutils"
	"io"
	"errors"
)

// NumFile represents file uploaded to system for saving
// files with numbers
type NumFile struct {
	ID          int64 `db:"id" goqu:"skipinsert"`
	Name        string `db:"name"`
	Description string `db:"description"`
	LocalName   string `db:"localname"`
	Username    string `db:"username"`
	UserID      string `db:"userid"`
	SubmittedAt int64  `db:"submittedat"`
	Deleted     bool   `db:"deleted"`
	Type        Type   `db:"type"`
}

type NumFileIO interface{
	Load(file io.Reader) ([]byte, error)
	LoadFile(filename string) ([]byte, error)
	Write(nf *NumFile) error
}

type RealNumFileIO struct {
	b []byte
}

func (nio *RealNumFileIO) Load(file io.Reader) ([]byte, error){
	var err error
	nio.b, err = ioutil.ReadAll(file)
	if err != nil {
		return nio.b, errors.New("Couldn't read file.")
	}
	if http.DetectContentType(nio.b) != "text/plain; charset=utf-8" && http.DetectContentType(nio.b) != "application/zip" {
		return nio.b, errors.New("File doesn't seem to be a text or excel file.")
	}
	return nio.b, nil
}

func (nio *RealNumFileIO) LoadFile(filename string) ([]byte, error){
	var err error
	nio.b, err = ioutil.ReadFile(filename)
	if err != nil {
		return nio.b, errors.New("Couldn't read file.")
	}
	if http.DetectContentType(nio.b) != "text/plain; charset=utf-8" && http.DetectContentType(nio.b) != "application/zip" {
		return nio.b, errors.New("File doesn't seem to be a text or excel file.")
	}
	return nio.b, nil
}

func (nio *RealNumFileIO) Write(file *NumFile) error {
	if file.LocalName == "" {
		return errors.New("Local Name can't be blank")
	}
	numfilePath := filepath.Join(NumFilePath, file.UserID)
	err := os.MkdirAll(numfilePath, 0711)
	if err != nil {
		return fmt.Errorf("Couldn't create directory %s", numfilePath)
	}
	err = ioutil.WriteFile(filepath.Join(numfilePath, file.LocalName), nio.b, 0600)
	if err != nil {
		return fmt.Errorf("Couldn't write file to disk at path %s.", filepath.Join(numfilePath, file.LocalName))
	}
	return nil
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

//NumFileRow represents one single row in excel or csv file
type NumFileRow struct {
	Destination string
	Params      map[string]string
}

var (
	//NumFilePath is folder relative to path where httpserver binary is, we'll save all files here
	NumFilePath = "./files"
)

// NumFileCriteria represents filters we can give to GetFiles method.
type NumFileCriteria struct {
	ID              string
	Username        string
	SubmittedAfter  int64
	SubmittedBefore int64
	Type            Type
	Name            string
	Deleted         bool
	OrderByKey      string
	OrderByDir      string
	From            string
	PerPage         uint
}

// Delete marks Deleted=true for a NumFile
func (nf *NumFile) Delete() error{
	nf.Deleted = true
	return nf.Update()

}

func (nf *NumFile) Update() error {
	_, err := db.Get().From("Message").Where(goqu.I("id").Eq(nf.ID)).Update(nf).Exec()
	if err != nil {
		log.WithError(err).Error("Couldn't update numfile. %+v", nf)
	}
	return err
}

// GetNumFiles filters files based on criteria
func GetNumFiles(c NumFileCriteria) ([]NumFile, error) {
	var (
		f          []NumFile
	)
	query := db.Get().From("NumFile")
	if c.ID != "" {
		query = query.Where(goqu.I("ID").Eq(c.ID))
	}

	var from interface{}
	if c.From != "" {
		if c.OrderByKey == "SubmittedAt" {
			from, err := strconv.ParseInt(c.From, 10, 64)
			if err != nil {
				return f, fmt.Errorf("Invalid value for from: %s", from)
			}
		} else {
			from = c.From
		}
	}
	if c.OrderByKey == "" {
		c.OrderByKey = "SubmittedAt"
	}
	if c.SubmittedAfter != 0 {
		query = query.Where(goqu.I("submittedat").Gte(c.SubmittedAfter))
	}
	if c.SubmittedBefore != 0 {
		query = query.Where(goqu.I("submittedat").Lte(c.SubmittedAfter))
	}
	if c.Username != ""{
		query = query.Where(goqu.I("username").Eq(c.Username))
	}
	if c.Type != ""{
		query = query.Where(goqu.I("type").Eq(c.Type))
	}
	if c.Name != ""{
		query = query.Where(goqu.I("name").Eq(c.Name))
	}
	query = query.Where(goqu.I("deleted").Eq(c.Deleted))
	orderDir := "DESC"
	if strings.ToUpper(c.OrderByDir) == "ASC" {
		orderDir = "ASC"
	}
	if from != nil {
		if orderDir == "ASC" {
			query = query.Where(goqu.I(c.OrderByKey).Gt(from))
		} else {
			query = query.Where(goqu.I(c.OrderByKey).Lt(from))
		}
	}
	orderExp := goqu.I(c.OrderByKey).Desc()
	if orderDir == "ASC" {
		orderExp = goqu.I(c.OrderByKey).Asc()
	}
	query = query.Order(orderExp)
	if c.PerPage == 0 {
		c.PerPage = 100
	}
	query = query.Limit(c.PerPage)
	queryStr, _, _ := query.ToSql()
	log.WithFields(log.Fields{"query": queryStr, "crtieria": c}).Info("Running query.")
	err := query.ScanStructs(&f)
	if err != nil {
		log.WithError(err).Error("Couldn't run query.")
	}
	return f, err
}

// Save saves a message struct in Message table
func (nf *NumFile) Save(name string, f multipart.File, fileIO NumFileIO) (int64, error) {
	fileType := Type(filepath.Ext(strings.ToLower(name)))
	if fileType != CSV && fileType != TXT && fileType != XLSX {
		return 0, fmt.Errorf("Only csv, txt and xlsx extensions are allowed Given file %s has extension %s.", name, fileType)
	}
	nf.Type = fileType
	nf.Name = name
	_, err := fileIO.Load(f)
	if err != nil {
		log.WithError(err).Error("Couldn't load file.")
		return 0, err
	}
	err = fileIO.Write(nf)
	if err != nil {
		return 0, fmt.Errorf("Couldn't write file to disk. Error: %s", err)
	}
	_, err = nf.ToNumbers(fileIO)
	if err != nil {
		log.WithError(err).Error("Couldn't read numbers from file.")
		return 0, err
	}
	resp, err := db.Get().From("NumFile").Insert(nf).Exec()
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
		}).Error("Error in inserting file in db.")
		return 0, err
	}
	return resp.LastInsertId()
}

// RowsFromString makes a NumFileRow list from comma separated numbers
func RowsFromString(numbers string) []NumFileRow {
	var nums []NumFileRow
	if numbers == "" {
		return nums
	}
	parts := strings.Split(numbers, ",")
	for _, num := range parts {
		nums = append(nums, NumFileRow{
			Destination: num,
		})
	}
	return nums
}

// ToNumbers reads a csv or xlsx file and returns array of NumFileRow with Destination and Params map
func (nf *NumFile) ToNumbers(nio NumFileIO) ([]NumFileRow, error) {
	var nums []NumFileRow
	nummap := make(map[string]NumFileRow) // used for unique numbers
	numfilePath := fmt.Sprintf("%s/%s/%s", NumFilePath, nf.UserID, nf.LocalName)
	b, err := nio.LoadFile(numfilePath)
	if err != nil {
		return nums, err
	}
	if nf.Type == CSV || nf.Type == TXT {
		for i, num := range strings.Split(stringutils.ByteToString(b), ",") {
			num = strings.Trim(num, "\t\n\v\f\r \u0085\u00a0")
			if len(num) > 15 || len(num) < 5 {
				return nums, fmt.Errorf("Entry number %d in file %s is invalid. Number must be greater than 5 characters and lesser than 16. Please fix it and retry.", i+1, nf.Name)
			}
			nummap[num] = NumFileRow{Destination: num}
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
			nr := NumFileRow{
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

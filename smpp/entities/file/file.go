package file

import (
	"fmt"
	"io"
)

// Store represents a numbers file store
type Store interface {
}

// File represents file uploaded to system for saving
// files with numbers
type File struct {
	ID          int64  `db:"id" goqu:"skipinsert"`
	Name        string `db:"name"`
	Description string `db:"description"`
	LocalName   string `db:"localname"`
	Username    string `db:"username"`
	SubmittedAt int64  `db:"submittedat"`
	Deleted     bool   `db:"deleted"`
	Type        Type   `db:"type"`
}

// FileIO is interface to save/load a file
type FileIO interface {
	Read(file io.Reader) ([]byte, error)
	Load(filename string) ([]byte, error)
	Write(nf *File) error
}

// Type represents type of file we're uploading
// can be excel/csv etc.
type Type string

func (n *Type) Scan(nf interface{}) error {
	*n = Type(fmt.Sprintf("%s", nf))
	return nil
}

const (
	// CSV is text file with .csv extension. This file should have comma separated numbers
	CSV Type = ".csv"
	// TXT is text file with .txt extension. This file should have comma separated numbers
	TXT = ".txt"
	// XLSX is excel file with .xlsx extension. These files should follow following structure:
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

// Row represents one single Row in excel or csv file
type Row struct {
	Destination string
	Params      map[string]string
}

var (
	// Path is folder relative to path where httpserver binary is, we'll save all files here
	Path = "./files"
)

// Criteria represents filters we can give to GetFiles method.
type Criteria struct {
	ID              int64
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

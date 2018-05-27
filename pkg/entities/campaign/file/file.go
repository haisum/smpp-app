package file

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"bitbucket.org/codefreak/hsmpp/pkg/stringutils"
)

// Store represents a numbers file store
type Store interface {
	List(c *Criteria) ([]File, error)
}

type ProcessExcelFunc func(reader io.Reader) (map[string]Row, error)

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
	MaxFileSize = 5 * 1024 * 1024
	// DefaultPath is location where files are supposed to be stored
	DefaultPath = "./files"
)

// Row represents one single Row in excel or csv file
type Row struct {
	Destination string
	Params      map[string]string
}

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

// RowsFromString makes a Row list from comma separated numbers
func RowsFromString(numbers string) []Row {
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
func ToNumbers(f *File, processExcel ProcessExcelFunc, reader io.Reader) ([]Row, error) {
	var nums []Row
	nummap := make(map[string]Row) // used for unique numbers
	if f.Type == CSV || f.Type == TXT {
		b, err := ioutil.ReadAll(reader)
		if err != nil {
			return nums, err
		}
		for i, num := range strings.Split(stringutils.ByteToString(b), ",") {
			num = strings.Trim(num, "\t\n\v\f\r \u0085\u00a0")
			if len(num) > 15 || len(num) < 5 {
				return nums, fmt.Errorf("entry number %d in file %s is invalid; number must be greater than 5 characters and lesser than 16; please fix it and retry", i+1, f.Name)
			}
			nummap[num] = Row{Destination: num}
		}
	} else if f.Type == XLSX {
		var err error
		nummap, err = processExcel(reader)
		if err != nil {
			return nums, err
		}
	} else {
		return nums, fmt.Errorf("this file type isn't supported yet")
	}
	if len(nummap) < 1 {
		return nums, fmt.Errorf("no numbers given in file")
	}
	for _, v := range nummap {
		nums = append(nums, v)
	}
	return nums, nil
}

package excel

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/haisum/smpp-app/pkg/entities/campaign/file"
	"github.com/tealeg/xlsx"
)

// ToNumbers reads bytes from reader as excel file then
// returns a map of file.Row records if valid records are found
// excel file must have following pattern:
// destination [ param1 param2 param3 ... ]
// 439099009   [ val1   val2   val3  ... ]
func ToNumbers(reader io.Reader) (map[string]file.Row, error) {
	numbers := make(map[string]file.Row)
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	xlFile, err := xlsx.OpenBinary(b)
	if err != nil {
		return numbers, err
	}
	if err = validateFile(xlFile); err != nil {
		return numbers, err
	}
	var keys []string
	for _, cell := range xlFile.Sheets[0].Rows[0].Cells {
		keys = append(keys, cell.Value)
	}
	for i := 1; i < len(xlFile.Sheets[0].Rows); i++ {
		if len(xlFile.Sheets[0].Rows[i].Cells) < 1 {
			return numbers, fmt.Errorf("row number %d doesn't have any value", i+1)
		}
		num := xlFile.Sheets[0].Rows[i].Cells[0].Value
		num = strings.Trim(num, "\t\n\v\f\r \u0085\u00a0")
		if len(num) > 15 || len(num) < 5 {
			return numbers, fmt.Errorf("row number %d in file is invalid; number must be greater than 5 characters and lesser than 16; please fix it and retry", i+1)
		}
		if len(xlFile.Sheets[0].Rows[i].Cells) < len(keys) {
			return numbers, fmt.Errorf("row number %d has blank values for some parameters", i)
		}
		row := file.Row{
			Destination: num,
			Params:      map[string]string{},
		}
		for j := 1; j < len(keys) && j < len(xlFile.Sheets[0].Rows[i].Cells); j++ {
			val := xlFile.Sheets[0].Rows[i].Cells[j].Value
			val = strings.Trim(val, "\t\n\v\f\r \u0085\u00a0")
			if val == "" {
				return numbers, fmt.Errorf("row number %d contains no value at cell number %d", i, j)
			}
			row.Params[keys[j]] = val
		}
		numbers[row.Destination] = row
	}
	return numbers, nil
}

func validateFile(xlFile *xlsx.File) error {
	if len(xlFile.Sheets) != 1 {
		return errors.New("xslx file should contain exactly one sheet")
	}
	if len(xlFile.Sheets[0].Rows) < 2 {
		return errors.New("xslx file is empty")
	}
	if len(xlFile.Sheets[0].Rows[0].Cells) == 0 || xlFile.Sheets[0].Rows[0].Cells[0].Value != "Destination" {
		return errors.New("first cell of excel sheet must be Destination header")
	}
	return nil
}

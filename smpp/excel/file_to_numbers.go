package excel

import (
	"bitbucket.org/codefreak/hsmpp/smpp/entities/campaign/file"
	"fmt"
	"github.com/tealeg/xlsx"
	"io"
	"io/ioutil"
	"strings"
)

func ToNumbers(reader io.Reader) (map[string]file.Row, error) {
	nums := make(map[string]file.Row)
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
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
		return nums, fmt.Errorf("first cell of excel sheet must be Destination header")
	}
	var keys []string
	for _, cell := range xlFile.Sheets[0].Rows[0].Cells {
		keys = append(keys, cell.Value)
	}
	for i := 1; i < len(xlFile.Sheets[0].Rows); i++ {
		if len(xlFile.Sheets[0].Rows[i].Cells) < 1 {
			return nums, fmt.Errorf("row number %d doesn't have any value", i+1)
		}
		num := xlFile.Sheets[0].Rows[i].Cells[0].Value
		num = strings.Trim(num, "\t\n\v\f\r \u0085\u00a0")
		if len(num) > 15 || len(num) < 5 {
			return nums, fmt.Errorf("row number %d in file is invalid; number must be greater than 5 characters and lesser than 16; please fix it and retry", i+1)
		}
		row := file.Row{
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
			row.Params[keys[j]] = val
		}
		nums[row.Destination] = row
	}
	return nums, nil
}

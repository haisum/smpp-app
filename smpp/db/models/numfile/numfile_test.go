package numfile

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/tealeg/xlsx"
	"gopkg.in/stretchr/testify.v1/mock"
	"io"
	"testing"
)

//Mock implementation of NumFileIO interface used for unit tests
type MockFileIO struct {
	mock.Mock
}

func (m *MockFileIO) Load(file io.Reader) ([]byte, error) {
	args := m.Called(file)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockFileIO) LoadFile(filename string) ([]byte, error) {
	args := m.Called(filename)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockFileIO) Write(nf *NumFile) error {
	args := m.Called(nf)
	return args.Error(0)
}

func TestNumFile_ToNumbers1(t *testing.T) {
	csvBytes := []byte("234235435,36545675467,324234234")
	csvFileIO := &MockFileIO{}
	csvFileIO.On("LoadFile", "./files/12/testfile.csv").Return(csvBytes, nil)
	nf := NumFile{
		UserID:    12,
		LocalName: "testfile.csv",
		Type:      CSV,
	}
	nums, err := nf.ToNumbers(csvFileIO)
	if len(nums) != 3 || err != nil {
		t.Errorf("Failed. %s, %+v", err, nums)
	}
	assert.Equal(t, Row{Destination: "234235435"}, nums[0])
	assert.Equal(t, Row{Destination: "36545675467"}, nums[1])
	assert.Equal(t, Row{Destination: "324234234"}, nums[2])
}

func TestNumFile_ToNumbers2(t *testing.T) {
	file := xlsx.NewFile()
	sheet, _ := file.AddSheet("sheet1")
	firstRow := sheet.AddRow()
	firstRow.AddCell().Value = "Destination"
	firstRow.AddCell().Value = "Param1"
	secondRow := sheet.AddRow()
	secondRow.AddCell().Value = "234235435"
	secondRow.AddCell().Value = "hello"
	thirdRow := sheet.AddRow()
	thirdRow.AddCell().Value = "36545675467"
	thirdRow.AddCell().Value = "world"
	buf := bytes.Buffer{}
	file.Write(&buf)
	xlsBytes := buf.Bytes()
	xlsFileIO := &MockFileIO{}
	xlsFileIO.On("LoadFile", "./files/12/testfile.xlsx").Return(xlsBytes, nil)
	nf := NumFile{
		UserID:    12,
		LocalName: "testfile.xlsx",
		Type:      XLSX,
	}
	nums, err := nf.ToNumbers(xlsFileIO)
	if len(nums) != 2 || err != nil {
		t.Fatalf("Failed. %s, %+v", err, nums)
	}
	assert.Contains(t, nums, Row{Destination: "234235435", Params: map[string]string{"Param1": "hello"}})
	assert.Contains(t, nums, Row{Destination: "36545675467", Params: map[string]string{"Param1": "world"}})
}

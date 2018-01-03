package numfile

import (
	"bytes"
	"io"
	"regexp"
	"testing"

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"github.com/stretchr/testify/assert"
	"github.com/tealeg/xlsx"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"gopkg.in/doug-martin/goqu.v3"
	"gopkg.in/stretchr/testify.v1/mock"
)

// MockFileIO is Mock implementation of NumFileIO interface used for unit tests
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

type MockFile struct {
	mock.Mock
}

func (m *MockFile) Read(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}
func (m *MockFile) ReadAt(p []byte, off int64) (n int, err error) {
	args := m.Called(p, off)
	return args.Int(0), args.Error(1)
}
func (m *MockFile) Seek(offset int64, whence int) (int64, error) {
	args := m.Called(offset, whence)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockFile) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNumFile_ToNumbers(t *testing.T) {
	csvBytes := []byte("234235435,36545675467,324234234")
	csvFileIO := &MockFileIO{}
	csvFileIO.On("LoadFile", "./files/admin/testfile.csv").Return(csvBytes, nil)
	nf := NumFile{
		Username:  "admin",
		LocalName: "testfile.csv",
		Type:      CSV,
	}
	nums, err := nf.ToNumbers(csvFileIO)
	if len(nums) != 3 || err != nil {
		t.Fatalf("Failed. %s, %+v", err, nums)
	}
	assert.Equal(t, Row{Destination: "234235435"}, nums[0])
	assert.Equal(t, Row{Destination: "36545675467"}, nums[1])
	assert.Equal(t, Row{Destination: "324234234"}, nums[2])
	assert.True(t, csvFileIO.AssertExpectations(t))
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
	xlsFileIO.On("LoadFile", "./files/admin/testfile.xlsx").Return(xlsBytes, nil)
	nf := NumFile{
		Username:  "admin",
		LocalName: "testfile.xlsx",
		Type:      XLSX,
	}
	nums, err := nf.ToNumbers(xlsFileIO)
	if len(nums) != 2 || err != nil {
		t.Fatalf("Failed. %s, %+v", err, nums)
	}
	assert.Contains(t, nums, Row{Destination: "234235435", Params: map[string]string{"Param1": "hello"}})
	assert.Contains(t, nums, Row{Destination: "36545675467", Params: map[string]string{"Param1": "world"}})
	assert.True(t, xlsFileIO.AssertExpectations(t))
}

func TestList(t *testing.T) {
	con1, dbmock, _ := db.ConnectMock(t)
	defer con1.Db.Close()
	assert := assert.New(t)
	expected, _, _ := db.Get().From("NumFile").Select(&NumFile{}).Where(goqu.I("ID").Eq(234),
		goqu.I("submittedat").Gte(12),
		goqu.I("submittedat").Lte(234),
		goqu.I("username").Eq("haisum"),
		goqu.I("name").Eq("myfile.xlsx"),
		goqu.I("deleted").Is(true),
		goqu.I("SubmittedAt").Gt(1)).Order(goqu.I("SubmittedAt").Asc()).Limit(10).ToSql()
	dbmock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1").AddRow("2").AddRow("3"))

	nums, err := List(Criteria{
		ID:              234,
		Username:        "haisum",
		Deleted:         true,
		Name:            "myfile.xslx",
		SubmittedAfter:  12,
		SubmittedBefore: 234,
		PerPage:         10,
		OrderByDir:      "ASC",
		From:            "1",
	})
	if !assert.Nil(err) {
		t.FailNow()
	}
	assert.Equal(3, len(nums))
	assert.Nil(dbmock.ExpectationsWereMet())
}

func TestNumFile_Delete(t *testing.T) {
	con1, dbmock, _ := db.ConnectMock(t)
	defer con1.Db.Close()
	assert := assert.New(t)
	dbmock.ExpectExec(regexp.QuoteMeta("`deleted`=1,`type`='' WHERE (`id` = 20)")).WillReturnResult(sqlmock.NewResult(0, 1))
	nf := NumFile{ID: 20}
	err := nf.Delete()
	assert.Nil(err)
	assert.Nil(dbmock.ExpectationsWereMet())
}

func TestNumFile_Save(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	assert := assert.New(t)
	nf := NumFile{ID: 20, Username: "haisum", LocalName: "myfile.txt", Type: TXT, Name: "hello.txt"}
	mockIO := &MockFileIO{}
	mockFile := &MockFile{}
	mockIO.On("Load", mockFile).Return([]byte{}, nil)
	mockIO.On("Write", &nf).Return(nil)
	mockIO.On("LoadFile", "./files/haisum/myfile.txt").Return([]byte("20340234"), nil)
	expected, _, _ := db.Get().From("NumFile").ToInsertSql(&nf)
	mock.ExpectExec(regexp.QuoteMeta(expected)).WillReturnResult(sqlmock.NewResult(23, 1))
	id, err := nf.Save("hello.txt", mockFile, mockIO)
	if !assert.Nil(err) {
		t.FailNow()
	}
	assert.Equal(int64(23), id)
	assert.Nil(mock.ExpectationsWereMet())
	assert.True(mockFile.AssertExpectations(t))
	assert.True(mockIO.AssertExpectations(t))
}

func TestNumFile_Update(t *testing.T) {
	con1, dbmock, _ := db.ConnectMock(t)
	defer con1.Db.Close()
	assert := assert.New(t)
	nf := NumFile{ID: 20, LocalName: "myfile"}
	expected, _, _ := db.Get().From("NumFile").Where(goqu.I("id").Eq(nf.ID)).ToUpdateSql(&nf)
	dbmock.ExpectExec(regexp.QuoteMeta(expected)).WillReturnResult(sqlmock.NewResult(0, 1))
	err := nf.Update()
	assert.Nil(err)
	assert.Nil(dbmock.ExpectationsWereMet())
}

func TestRowsFromString(t *testing.T) {
	numbers := "324324,435345,67567567,345345345"
	rows := RowsFromString(numbers)
	assert := assert.New(t)
	if !assert.Equal(len(rows), 4) {
		t.FailNow()
	}
	assert.Equal(rows[0].Destination, "324324")
	assert.Equal(rows[2].Destination, "67567567")
	assert.Equal(rows[3].Destination, "345345345")
}

func TestType_Scan(t *testing.T) {
	tp := Type("")
	tp.Scan([]byte("Hello"))
	assert.Equal(t, Type("Hello"), tp)
}

package file

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"io"

	"bitbucket.org/codefreak/hsmpp/pkg/db"
	"bitbucket.org/codefreak/hsmpp/pkg/entities/campaign/file"
	"github.com/pkg/errors"
	"gopkg.in/doug-martin/goqu.v3"
)

type store struct {
	db *db.DB
}

func NewStore(db *db.DB) *store {
	return &store{
		db,
	}
}

// Delete marks Deleted=true for a NumFile
func (s *store) Delete(f *file.File) error {
	f.Deleted = true
	return s.Update(f)

}

// Update updates values of a given num file. ID field must be populated in nf object before calling update.
func (s *store) Update(f *file.File) error {
	_, err := s.db.From("File").Where(goqu.I("id").Eq(f.ID)).Update(f).Exec()
	return err
}

// List filters files based on criteria
func (s *store) List(c *file.Criteria) ([]file.File, error) {
	var (
		f []file.File
	)
	query := s.db.From("File")
	if c.ID != 0 {
		query = query.Where(goqu.I("ID").Eq(c.ID))
	}
	if c.OrderByKey == "" {
		c.OrderByKey = "SubmittedAt"
	}
	var from interface{}
	if c.From != "" {
		if c.OrderByKey == "SubmittedAt" {
			var err error
			from, err = strconv.ParseInt(c.From, 10, 64)
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
		query = query.Where(goqu.I("submittedat").Lte(c.SubmittedBefore))
	}
	if c.Username != "" {
		query = query.Where(goqu.I("username").Eq(c.Username))
	}
	if c.Type != "" {
		query = query.Where(goqu.I("type").Eq(c.Type))
	}
	if c.Name != "" {
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
	err := query.ScanStructs(&f)
	return f, err
}

// Save saves a message struct in Message table
func (s *store) Save(f *file.File, name string, processExcelFunc file.ProcessExcelFunc, reader io.ReadCloser, writer io.WriteCloser) (int64, error) {
	fileType := file.Type(filepath.Ext(strings.ToLower(name)))
	if fileType != file.CSV && fileType != file.TXT && fileType != file.XLSX {
		return 0, fmt.Errorf("only csv, txt and xlsx extensions are allowed; given file %s has extension %s", name, fileType)
	}
	f.Type = fileType
	f.Name = name
	_, err := file.ToNumbers(f, processExcelFunc, reader)
	defer reader.Close()
	if err != nil {
		return 0, err
	}
	_, err = io.Copy(writer, reader)
	defer writer.Close()
	if err != nil {
		return 0, errors.Wrap(err, "couldn't write file to disk")
	}
	resp, err := s.db.From("File").Insert(f).Exec()
	if err != nil {
		return 0, err
	}
	return resp.LastInsertId()
}

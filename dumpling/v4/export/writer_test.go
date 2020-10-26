package export

import (
	"context"
	"database/sql/driver"
	"io/ioutil"
	"os"
	"path"

	. "github.com/pingcap/check"
)

var _ = Suite(&testWriterSuite{})

type testWriterSuite struct{}

func (s *testDumpSuite) TestWriteDatabaseMeta(c *C) {
	dir := c.MkDir()
	ctx := context.Background()

	config := DefaultConfig()
	config.OutputDirPath = dir
	err := adjustConfig(ctx, config)
	c.Assert(err, IsNil)

	writer, err := NewSimpleWriter(config)
	c.Assert(err, IsNil)
	err = writer.WriteDatabaseMeta(ctx, "test", "CREATE DATABASE `test`")
	c.Assert(err, IsNil)
	p := path.Join(dir, "test-schema-create.sql")
	_, err = os.Stat(p)
	c.Assert(err, IsNil)
	bytes, err := ioutil.ReadFile(p)
	c.Assert(err, IsNil)
	c.Assert(string(bytes), Equals, "/*!40101 SET NAMES binary*/;\nCREATE DATABASE `test`;\n")
}

func (s *testDumpSuite) TestWriteTableMeta(c *C) {
	dir := c.MkDir()
	ctx := context.Background()

	config := DefaultConfig()
	config.OutputDirPath = dir
	err := adjustConfig(ctx, config)
	c.Assert(err, IsNil)

	writer, err := NewSimpleWriter(config)
	c.Assert(err, IsNil)
	err = writer.WriteTableMeta(ctx, "test", "t", "CREATE TABLE t (a INT)")
	c.Assert(err, IsNil)
	p := path.Join(dir, "test.t-schema.sql")
	_, err = os.Stat(p)
	c.Assert(err, IsNil)
	bytes, err := ioutil.ReadFile(p)
	c.Assert(err, IsNil)
	c.Assert(string(bytes), Equals, "/*!40101 SET NAMES binary*/;\nCREATE TABLE t (a INT);\n")
}

func (s *testDumpSuite) TestWriteViewMeta(c *C) {
	dir := c.MkDir()
	ctx := context.Background()

	config := DefaultConfig()
	config.OutputDirPath = dir
	err := adjustConfig(ctx, config)
	c.Assert(err, IsNil)

	writer, err := NewSimpleWriter(config)
	c.Assert(err, IsNil)
	specCmt := "/*!40101 SET NAMES binary*/;\n"
	createTableSQL := "CREATE TABLE `v`(\n`a` int\n)ENGINE=MyISAM;\n"
	createViewSQL := "DROP TABLE IF EXISTS `v`;\nDROP VIEW IF EXISTS `v`;\nSET @PREV_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT;\nSET @PREV_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS;\nSET @PREV_COLLATION_CONNECTION=@@COLLATION_CONNECTION;\nSET character_set_client = utf8;\nSET character_set_results = utf8;\nSET collation_connection = utf8_general_ci;\nCREATE ALGORITHM=UNDEFINED DEFINER=`root`@`localhost` SQL SECURITY DEFINER VIEW `v` (`a`) AS SELECT `t`.`a` AS `a` FROM `test`.`t`;\nSET character_set_client = @PREV_CHARACTER_SET_CLIENT;\nSET character_set_results = @PREV_CHARACTER_SET_RESULTS;\nSET collation_connection = @PREV_COLLATION_CONNECTION;\n"
	err = writer.WriteViewMeta(ctx, "test", "v", createTableSQL, createViewSQL)
	c.Assert(err, IsNil)

	p := path.Join(dir, "test.v-schema.sql")
	_, err = os.Stat(p)
	c.Assert(err, IsNil)
	bytes, err := ioutil.ReadFile(p)
	c.Assert(err, IsNil)
	c.Assert(string(bytes), Equals, specCmt+createTableSQL)

	p = path.Join(dir, "test.v-schema-view.sql")
	_, err = os.Stat(p)
	c.Assert(err, IsNil)
	bytes, err = ioutil.ReadFile(p)
	c.Assert(err, IsNil)
	c.Assert(string(bytes), Equals, specCmt+createViewSQL)
}

func (s *testDumpSuite) TestWriteTableData(c *C) {
	dir := c.MkDir()

	ctx := context.Background()

	config := DefaultConfig()
	config.OutputDirPath = dir
	err := adjustConfig(ctx, config)
	c.Assert(err, IsNil)

	simpleWriter, err := NewSimpleWriter(config)
	c.Assert(err, IsNil)
	writer := SQLWriter{SimpleWriter: simpleWriter}

	data := [][]driver.Value{
		{"1", "male", "bob@mail.com", "020-1234", nil},
		{"2", "female", "sarah@mail.com", "020-1253", "healthy"},
		{"3", "male", "john@mail.com", "020-1256", "healthy"},
		{"4", "female", "sarah@mail.com", "020-1235", "healthy"},
	}
	colTypes := []string{"INT", "SET", "VARCHAR", "VARCHAR", "TEXT"}
	specCmts := []string{
		"/*!40101 SET NAMES binary*/;",
		"/*!40014 SET FOREIGN_KEY_CHECKS=0*/;",
	}
	tableIR := newMockTableIR("test", "employee", data, specCmts, colTypes)
	err = writer.WriteTableData(ctx, tableIR)
	c.Assert(err, IsNil)

	p := path.Join(dir, "test.employee.0.sql")
	_, err = os.Stat(p)
	c.Assert(err, IsNil)
	bytes, err := ioutil.ReadFile(p)
	c.Assert(err, IsNil)

	expected := "/*!40101 SET NAMES binary*/;\n" +
		"/*!40014 SET FOREIGN_KEY_CHECKS=0*/;\n" +
		"INSERT INTO `employee` VALUES\n" +
		"(1,'male','bob@mail.com','020-1234',NULL),\n" +
		"(2,'female','sarah@mail.com','020-1253','healthy'),\n" +
		"(3,'male','john@mail.com','020-1256','healthy'),\n" +
		"(4,'female','sarah@mail.com','020-1235','healthy');\n"
	c.Assert(string(bytes), Equals, expected)
}

func (s *testDumpSuite) TestWriteTableDataWithFileSize(c *C) {
	dir := c.MkDir()

	ctx := context.Background()

	config := DefaultConfig()
	config.OutputDirPath = dir
	config.FileSize = 50
	err := adjustConfig(ctx, config)
	c.Assert(err, IsNil)
	specCmts := []string{
		"/*!40101 SET NAMES binary*/;",
		"/*!40014 SET FOREIGN_KEY_CHECKS=0*/;",
	}
	config.FileSize += uint64(len(specCmts[0]) + 1)
	config.FileSize += uint64(len(specCmts[1]) + 1)
	config.FileSize += uint64(len("INSERT INTO `employees` VALUES\n"))

	simpleWriter, err := NewSimpleWriter(config)
	c.Assert(err, IsNil)
	writer := SQLWriter{SimpleWriter: simpleWriter}

	data := [][]driver.Value{
		{"1", "male", "bob@mail.com", "020-1234", nil},
		{"2", "female", "sarah@mail.com", "020-1253", "healthy"},
		{"3", "male", "john@mail.com", "020-1256", "healthy"},
		{"4", "female", "sarah@mail.com", "020-1235", "healthy"},
	}
	colTypes := []string{"INT", "SET", "VARCHAR", "VARCHAR", "TEXT"}
	tableIR := newMockTableIR("test", "employee", data, specCmts, colTypes)
	err = writer.WriteTableData(ctx, tableIR)
	c.Assert(err, IsNil)

	cases := map[string]string{
		"test.employee.0.sql": "/*!40101 SET NAMES binary*/;\n" +
			"/*!40014 SET FOREIGN_KEY_CHECKS=0*/;\n" +
			"INSERT INTO `employee` VALUES\n" +
			"(1,'male','bob@mail.com','020-1234',NULL),\n" +
			"(2,'female','sarah@mail.com','020-1253','healthy');\n",
		"test.employee.1.sql": "/*!40101 SET NAMES binary*/;\n" +
			"/*!40014 SET FOREIGN_KEY_CHECKS=0*/;\n" +
			"INSERT INTO `employee` VALUES\n" +
			"(3,'male','john@mail.com','020-1256','healthy'),\n" +
			"(4,'female','sarah@mail.com','020-1235','healthy');\n",
	}

	for p, expected := range cases {
		p := path.Join(dir, p)
		_, err = os.Stat(p)
		c.Assert(err, IsNil)
		bytes, err := ioutil.ReadFile(p)
		c.Assert(err, IsNil)
		c.Assert(string(bytes), Equals, expected)
	}
}

func (s *testDumpSuite) TestWriteTableDataWithStatementSize(c *C) {
	dir := c.MkDir()

	ctx := context.Background()

	config := DefaultConfig()
	config.OutputDirPath = dir
	config.StatementSize = 50
	config.StatementSize += uint64(len("INSERT INTO `employee` VALUES\n"))
	var err error
	config.OutputFileTemplate, err = ParseOutputFileTemplate("specified-name")
	c.Assert(err, IsNil)
	err = adjustConfig(ctx, config)
	c.Assert(err, IsNil)

	simpleWriter, err := NewSimpleWriter(config)
	c.Assert(err, IsNil)
	writer := SQLWriter{SimpleWriter: simpleWriter}

	data := [][]driver.Value{
		{"1", "male", "bob@mail.com", "020-1234", nil},
		{"2", "female", "sarah@mail.com", "020-1253", "healthy"},
		{"3", "male", "john@mail.com", "020-1256", "healthy"},
		{"4", "female", "sarah@mail.com", "020-1235", "healthy"},
	}
	colTypes := []string{"INT", "SET", "VARCHAR", "VARCHAR", "TEXT"}
	specCmts := []string{
		"/*!40101 SET NAMES binary*/;",
		"/*!40014 SET FOREIGN_KEY_CHECKS=0*/;",
	}
	tableIR := newMockTableIR("te%/st", "employee", data, specCmts, colTypes)
	err = writer.WriteTableData(ctx, tableIR)
	c.Assert(err, IsNil)

	// only with statement size
	cases := map[string]string{
		"specified-name.sql": "/*!40101 SET NAMES binary*/;\n" +
			"/*!40014 SET FOREIGN_KEY_CHECKS=0*/;\n" +
			"INSERT INTO `employee` VALUES\n" +
			"(1,'male','bob@mail.com','020-1234',NULL),\n" +
			"(2,'female','sarah@mail.com','020-1253','healthy');\n" +
			"INSERT INTO `employee` VALUES\n" +
			"(3,'male','john@mail.com','020-1256','healthy'),\n" +
			"(4,'female','sarah@mail.com','020-1235','healthy');\n",
	}

	for p, expected := range cases {
		p := path.Join(config.OutputDirPath, p)
		_, err = os.Stat(p)
		c.Assert(err, IsNil)
		bytes, err := ioutil.ReadFile(p)
		c.Assert(err, IsNil)
		c.Assert(string(bytes), Equals, expected)
	}

	// with file size and statement size
	config.FileSize = 204
	config.StatementSize = 95
	config.FileSize += uint64(len(specCmts[0]) + 1)
	config.FileSize += uint64(len(specCmts[1]) + 1)
	config.StatementSize += uint64(len("INSERT INTO `employee` VALUES\n"))
	// test specifying filename format
	config.OutputFileTemplate, err = ParseOutputFileTemplate("{{.Index}}-{{.Table}}-{{fn .DB}}")
	c.Assert(err, IsNil)
	os.RemoveAll(config.OutputDirPath)
	config.OutputDirPath, err = ioutil.TempDir("", "dumpling")
	newStorage, err := config.createExternalStorage(context.Background())
	c.Assert(err, IsNil)
	config.ExternalStorage = newStorage

	cases = map[string]string{
		"0-employee-te%25%2Fst.sql": "/*!40101 SET NAMES binary*/;\n" +
			"/*!40014 SET FOREIGN_KEY_CHECKS=0*/;\n" +
			"INSERT INTO `employee` VALUES\n" +
			"(1,'male','bob@mail.com','020-1234',NULL),\n" +
			"(2,'female','sarah@mail.com','020-1253','healthy');\n" +
			"INSERT INTO `employee` VALUES\n" +
			"(3,'male','john@mail.com','020-1256','healthy');\n",
		"1-employee-te%25%2Fst.sql": "/*!40101 SET NAMES binary*/;\n" +
			"/*!40014 SET FOREIGN_KEY_CHECKS=0*/;\n" +
			"INSERT INTO `employee` VALUES\n" +
			"(4,'female','sarah@mail.com','020-1235','healthy');\n",
	}

	tableIR = newMockTableIR("te%/st", "employee", data, specCmts, colTypes)
	c.Assert(writer.WriteTableData(ctx, tableIR), IsNil)
	c.Assert(err, IsNil)
	for p, expected := range cases {
		p := path.Join(config.OutputDirPath, p)
		_, err = os.Stat(p)
		c.Assert(err, IsNil)
		bytes, err := ioutil.ReadFile(p)
		c.Assert(err, IsNil)
		c.Assert(string(bytes), Equals, expected)
	}
}
/*----------------------------------------------------------------
 *  Copyright (c) ThoughtWorks, Inc.
 *  Licensed under the Apache License, Version 2.0
 *  See LICENSE in the project root for license information.
 *----------------------------------------------------------------*/

package common

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "gopkg.in/check.v1"
)

const dummyProject = "dummy_proj"

func Test(t *testing.T) { TestingT(t) }

type MySuite struct {
	testDir string
}

var _ = Suite(&MySuite{})

func (s *MySuite) SetUpSuite(c *C) {
	cwd, _ := os.Getwd()
	s.testDir = getAbsPath(cwd)
	createDummyProject(dummyProject)
}

func (s *MySuite) SetUpTest(c *C) {
	os.Chdir(s.testDir)
}

func (s *MySuite) TearDownTest(c *C) {
	os.Chdir(s.testDir)
}

func (s *MySuite) TearDownSuite(c *C) {
	os.RemoveAll(dummyProject)
}

func createDummyProject(project string) {
	dirsToCreate := []string{project,
		filepath.Join(project, "specs"),
		filepath.Join(project, "concepts"),
		filepath.Join(project, "specs", "nested"),
		filepath.Join(project, "specs", "nested", "deep_nested"),
		filepath.Join(project, "concepts", "nested"),
		filepath.Join(project, "concepts", "nested", "deep_nested"),
		filepath.Join(project, EnvDirectoryName),
		filepath.Join(project, ".git"),
		filepath.Join(project, EnvDirectoryName, DefaultEnvDir)}

	filesToCreate := []string{filepath.Join(project, ManifestFile),
		filepath.Join(project, ".git", "fourth.cpt"),
		filepath.Join(project, ".git", "fifth.cpt"),
		filepath.Join(project, "specs", "first.spec"),
		filepath.Join(project, "specs", "second.spec"),
		filepath.Join(project, "specs", "nested", "nested.spec"),
		filepath.Join(project, "specs", "nested", "deep_nested", "deep_nested.spec"),
		filepath.Join(project, "concepts", "first.cpt"),
		filepath.Join(project, "concepts", "nested", "nested.cpt"),
		filepath.Join(project, "concepts", "nested", "deep_nested", "deep_nested.cpt"),
		filepath.Join(project, EnvDirectoryName, DefaultEnvDir, DefaultEnvFileName)}

	for _, dirPath := range dirsToCreate {
		os.Mkdir(dirPath, (os.FileMode)(0777))
	}

	for _, filePath := range filesToCreate {
		_, err := os.Create(filePath)
		if err != nil {
			panic(err)
		}
	}
}

func (s *MySuite) TestGetProjectRoot(c *C) {
	expectedRoot := getAbsPath(dummyProject)
	os.Chdir(dummyProject)

	root, err := GetProjectRoot()

	c.Assert(err, IsNil)
	c.Assert(root, Equals, expectedRoot)
}

func (s *MySuite) TestGetProjectRootFromNestedDir(c *C) {
	expectedRoot := getAbsPath(dummyProject)
	os.Chdir(filepath.Join(dummyProject, "specs", "nested", "deep_nested"))

	root, err := GetProjectRoot()

	c.Assert(err, IsNil)
	c.Assert(root, Equals, expectedRoot)
}

func (s *MySuite) TestGetProjectFailing(c *C) {

	_, err := GetProjectRoot()
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Failed to find Gauge project directory. Missing manifest.json file.")
}

func (s *MySuite) TestGetDirInProject(c *C) {
	os.Chdir(dummyProject)

	concepts, err := GetDirInProject("concepts", "")

	c.Assert(err, IsNil)
	c.Assert(concepts, Equals, filepath.Join(s.testDir, dummyProject, "concepts"))
}

func (s *MySuite) TestGetDirInProjectFromNestedDir(c *C) {
	os.Chdir(filepath.Join(dummyProject, "specs", "nested", "deep_nested"))

	concepts, err := GetDirInProject("concepts", "")

	c.Assert(err, IsNil)
	c.Assert(concepts, Equals, filepath.Join(s.testDir, dummyProject, "concepts"))
}

func (s *MySuite) TestGetNotExistingDirInProject(c *C) {
	os.Chdir(filepath.Join(dummyProject, "specs", "nested", "deep_nested"))

	_, err := GetDirInProject("invalid", "")

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, fmt.Sprintf("Could not find invalid directory. %s does not exist", filepath.Join(s.testDir, dummyProject, "invalid")))
}

func (s *MySuite) TestFindFilesInDir(c *C) {
	foundSpecFiles := FindFilesInDir(filepath.Join(dummyProject, "specs"), func(filePath string) bool {
		return filepath.Ext(filePath) == ".spec"
	}, func(p string, f os.FileInfo) bool {
		return false
	})

	c.Assert(len(foundSpecFiles), Equals, 4)

	foundConceptFiles := FindFilesInDir(filepath.Join(dummyProject, "concepts"), func(filePath string) bool {
		return filepath.Ext(filePath) == ".cpt"
	}, func(p string, f os.FileInfo) bool {
		return false
	})

	c.Assert(len(foundConceptFiles), Equals, 3)
}

func (s *MySuite) TestFindFilesInDirFiltersDirectoriesThatAreSkipped(c *C) {
	foundConceptFiles := FindFilesInDir(dummyProject, func(filePath string) bool {
		return filepath.Ext(filePath) == ".cpt"
	}, func(p string, f os.FileInfo) bool {
		return strings.HasPrefix(f.Name(), ".")
	})

	c.Assert(len(foundConceptFiles), Equals, 3)
}

func (s *MySuite) TestFileExists(c *C) {
	c.Assert(FileExists(filepath.Join(dummyProject, ManifestFile)), Equals, true)
	c.Assert(FileExists("invalid"), Equals, false)
}

func (s *MySuite) TestGetDefaultPropertiesFile(c *C) {
	os.Chdir(dummyProject)
	envFile, err := GetDefaultPropertiesFile()
	c.Assert(err, IsNil)
	c.Assert(envFile, Equals, filepath.Join(s.testDir, dummyProject, EnvDirectoryName, DefaultEnvDir, DefaultEnvFileName))
}

func (s *MySuite) TestAppendingPropertiesToFile(c *C) {
	os.Chdir(dummyProject)
	defaultProperties, err := GetDefaultPropertiesFile()
	c.Assert(err, IsNil)

	firstProperty := &Property{Name: "first", Comment: "firstComment", DefaultValue: "firstValue"}
	secondProperty := &Property{Name: "second", Comment: "secondComment", DefaultValue: "secondValue"}
	err = AppendProperties(defaultProperties, firstProperty, secondProperty)
	c.Assert(err, IsNil)

	contents, _ := ReadFileContents(defaultProperties)
	c.Assert(strings.Contains(contents, firstProperty.String()), Equals, true)
	c.Assert(strings.Contains(contents, secondProperty.String()), Equals, true)
	indexIsLesser := strings.Index(contents, firstProperty.String()) < strings.Index(contents, secondProperty.String())
	c.Assert(indexIsLesser, Equals, true)

}

func (s *MySuite) TestReadingContentsInUTF8WithoutSignature(c *C) {
	filePath, _ := filepath.Abs(filepath.Join("_testdata", "utf8WithoutSig.csv"))

	contents, err := ReadFileContents(filePath)

	c.Assert(err, Equals, nil)
	if isWindows() {
		c.Assert(contents, Equals, "column1,column2\r\nvalue1,value2\r\n")
	} else {
		c.Assert(contents, Equals, "column1,column2\nvalue1,value2\n")
	}
}

func (s *MySuite) TestReadingContentsInUTF8WithSignature(c *C) {
	filePath, _ := filepath.Abs(filepath.Join("_testdata", "utf8WithSig.csv"))
	bytes, _ := os.ReadFile(filePath)
	contents, err := ReadFileContents(filePath)

	c.Assert(err, Equals, nil)
	if isWindows() {
		c.Assert(string(bytes), Equals, "\ufeff"+"word,count\r\ngauge,3\r\n")
		c.Assert(contents, Equals, "word,count\r\ngauge,3\r\n")
	} else {
		c.Assert(string(bytes), Equals, "\ufeff"+"word,count\ngauge,3\n")
		c.Assert(contents, Equals, "word,count\ngauge,3\n")
	}
}

func (s *MySuite) TestGetProjectRootFromSpecPath(c *C) {
	expectedRoot, _ := filepath.Abs(filepath.Join(dummyProject))
	absProjPath, _ := filepath.Abs(dummyProject)
	os.Chdir(os.TempDir())

	root, err := GetProjectRootFromSpecPath(absProjPath + "/specs/")

	c.Assert(err, IsNil)
	c.Assert(root, Equals, expectedRoot)
}

func (s *MySuite) TestGetProjectRootGivesErrorWhenProvidedInvalidSpecFilePath(c *C) {
	os.Chdir(os.TempDir())

	root, err := GetProjectRootFromSpecPath("/specs/nested/deep_nested/deep_nested.spec")

	c.Assert(err.Error(), Equals, fmt.Sprintf("Failed to find Gauge project directory. Missing manifest.json file."))
	c.Assert(root, Equals, "")
}

func (s *MySuite) TestGetProjectRootFromSpecFilePath(c *C) {
	expectedRoot, _ := filepath.Abs(filepath.Join(dummyProject))
	absProjPath, _ := filepath.Abs(dummyProject)
	os.Chdir(os.TempDir())

	root, err := GetProjectRootFromSpecPath(absProjPath + "/specs/nested/deep_nested/deep_nested.spec")

	c.Assert(err, IsNil)
	c.Assert(root, Equals, expectedRoot)
}

func (s *MySuite) TestSubDirectoryExists(c *C) {
	rootDir, _ := filepath.Abs(filepath.Join(dummyProject))

	specsExists := SubDirectoryExists(rootDir, "specs")
	fooExists := SubDirectoryExists(rootDir, "foo")

	c.Assert(specsExists, Equals, true)
	c.Assert(fooExists, Equals, false)
}

func (s *MySuite) TestGetExecutableCommand(c *C) {
	wd, _ := os.Getwd()
	workingDirectory := "/working/directory"
	logger1 := createLogger("logger1")
	logger2 := createLogger("logger2")
	command := "gauge"

	cmd := prepareCommand(false, []string{command, "-v", "-d"}, workingDirectory, logger1, logger2)

	pd, _ := os.Getwd()
	args := make(map[string]bool)
	for _, v := range cmd.Args {
		args[v] = true
	}

	c.Assert(wd, Equals, pd)
	c.Assert(cmd, NotNil)
	c.Assert(cmd.Path, Equals, command)
	c.Assert(cmd.Dir, Equals, workingDirectory)
	c.Assert(logger1.equals(cmd.Stdout.(logger)), Equals, true)
	c.Assert(logger2.equals(cmd.Stderr.(logger)), Equals, true)
	c.Assert(args["-v"], Equals, true)
	c.Assert(args["-d"], Equals, true)
}

func (s *MySuite) TestGetExecutableCommandForCommandsWithPath(c *C) {
	wd, _ := os.Getwd()

	workingDirectory := "/working/directory"
	logger1 := createLogger("logger1")
	logger2 := createLogger("logger2")
	command := "/bin/java"

	cmd := prepareCommand(false, []string{command, "-v", "-d"}, workingDirectory, logger1, logger2)

	pd, _ := os.Getwd()
	args := make(map[string]bool)
	for _, v := range cmd.Args {
		args[v] = true
	}

	c.Assert(wd, Equals, pd)
	c.Assert(cmd, NotNil)
	c.Assert(cmd.Path, Equals, command)
	c.Assert(cmd.Dir, Equals, workingDirectory)
	c.Assert(logger1.equals(cmd.Stdout.(logger)), Equals, true)
	c.Assert(logger2.equals(cmd.Stderr.(logger)), Equals, true)
	c.Assert(args["-v"], Equals, true)
	c.Assert(args["-d"], Equals, true)
}

func (s *MySuite) TestGetExecutableCommandForSystemCommands(c *C) {
	wd, _ := os.Getwd()

	command := "go"
	cmd := GetExecutableCommand(true, command)

	pd, _ := os.Getwd()
	expectedCommand := exec.Command("go")

	c.Assert(wd, Equals, pd)
	c.Assert(cmd, NotNil)
	c.Assert(cmd.Path, Equals, expectedCommand.Path)
}

func (s *MySuite) TestGetGaugeHomeDirectory(c *C) {
	path := "value string"
	os.Setenv(GaugeHome, path)

	home, err := GetGaugeHomeDirectory()

	c.Assert(err, Equals, nil)
	c.Assert(home, Equals, path)
}

func (s *MySuite) TestGetGaugeHomeDirectoryWhen_GAUGE_HOME_IsNotSet(c *C) {
	os.Setenv(GaugeHome, "")

	home, err := GetGaugeHomeDirectory()

	c.Assert(err, Equals, nil)
	if isWindows() {
		c.Assert(home, Equals, filepath.Join(os.Getenv(appData), ProductName))
	} else {
		c.Assert(home, Equals, filepath.Join(os.Getenv("HOME"), DotGauge))
	}
}

func getAbsPath(path string) string {
	abs, _ := filepath.Abs(path)
	absPath, _ := filepath.EvalSymlinks(abs)
	return absPath
}

type logger struct {
	name string
}

func (l logger) Write(b []byte) (n int, err error) {
	return 1, nil
}

func createLogger(name string) logger {
	return logger{name}
}

func (l logger) equals(l1 logger) bool {
	return l.name == l1.name
}

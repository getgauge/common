package common

import (
	. "launchpad.net/gocheck"
	"testing"
	"os"
	"path/filepath"
)

const dummyProject = "dummy_proj"

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{
	testDir string
}

var _ = Suite(&MySuite{})

func (s *MySuite) SetUpSuite(c *C) {
	cwd, _ := os.Getwd()
	s.testDir, _ = filepath.Abs(cwd)
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
		filepath.Join(project, "concepts", "nested", "deep_nested")}

	filesToCreate := []string{filepath.Join(project, "manifest.json"),
		filepath.Join(project, "specs", "first.spec"),
		filepath.Join(project, "specs", "second.spec"),
		filepath.Join(project, "specs", "nested", "nested.spec"),
		filepath.Join(project, "specs", "nested", "deep_nested", "deep_nested.spec"),
		filepath.Join(project, "concepts", "first.cpt"),
		filepath.Join(project, "concepts", "nested", "nested.cpt"),
		filepath.Join(project, "concepts", "nested", "deep_nested", "deep_nested.cpt")}

	for _, dirPath := range dirsToCreate {
		os.Mkdir(dirPath, (os.FileMode)(0777))
	}

	for _, filePath := range filesToCreate {
		_, err := os.Create(filePath)
		if (err != nil) {
			panic(err)
		}
	}
}

func (s *MySuite) TestGetProjectRoot(c *C) {
	expectedRoot, _ := filepath.Abs(filepath.Join(dummyProject))
	os.Chdir(dummyProject)

	root, err := GetProjectRoot()

	c.Assert(err, Equals, nil)
	c.Assert(root, Equals, expectedRoot)
}

func (s *MySuite) TestGetProjectRootFromNestedDir(c *C) {
	expectedRoot, _ := filepath.Abs(filepath.Join(dummyProject))
	os.Chdir(filepath.Join(dummyProject, "specs", "nested", "deep_nested"))

	root, err := GetProjectRoot()

	c.Assert(err, Equals, nil)
	c.Assert(root, Equals, expectedRoot)
}

func (s *MySuite) TestGetProjectFailing(c *C) {

	_, err := GetProjectRoot()
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Failed to find project directory, run the command inside the project")
}

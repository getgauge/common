// Common functions shared across all files
package common

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	ManifestFile            = "manifest.json"
	PluginJsonFile          = "plugin.json"
	NewDirectoryPermissions = 0755
	NewFilePermissions      = 0644
	DefaultEnvJSONFileName  = "default.json"
	EnvDirectoryName        = "env"
	ConceptsDirectoryName   = "concepts"
	ConceptFileExtension    = ".cpt"
)

// A project root is where a manifest.json files exists
// this routine keeps going upwards searching for manifest.json
func GetProjectRoot() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to find project root directory. %s\n", err.Error())
		os.Exit(2)
	}

	manifestExists := func(dir string) bool {
		return FileExists(path.Join(dir, ManifestFile))
	}
	dir := pwd
	for {
		if manifestExists(dir) {
			return dir, nil
		}
		if dir == filepath.Clean(fmt.Sprintf("%c", os.PathSeparator)) || dir == "" {
			return "", errors.New("Failed to find project directory, run the command inside the project")
		}
		dir = filepath.Clean(fmt.Sprintf("%s%c..", dir, os.PathSeparator))
	}

	return "", errors.New("Failed to find project directory, run the command inside the project")
}

func GetDirInProject(dirName string) (string, error) {
	projectRoot, err := GetProjectRoot()
	if (err != nil ) {
		return "", err
	}

	requiredDir := path.Join(projectRoot, dirName)
	if (!DirExists(requiredDir)) {
		return "", errors.New(fmt.Sprintf("Could not find %s directory. %s does not exist", dirName, requiredDir))
	}

	return requiredDir, nil
}

func FindFilesInDir(dirPath string, isValidFile func(path string) (bool)) ([]string) {
	files := make([]string, 0)
	filepath.Walk(dirPath, func(path string, f os.FileInfo, err error) error {
			if err == nil && !f.IsDir() && isValidFile(path) {
				files = append(files, path)
			}
			return err
		})
	return files
}


func GetSearchPathForSharedFiles() []string {
	return []string{"/usr/local/share/twist2", "/usr/share/twist2"}
}

func GetLanguageJSONFilePath(language string) (string, error) {
	searchPaths := GetSearchPathForSharedFiles()
	for _, p := range searchPaths {
		languageJson := filepath.Join(p, "languages", fmt.Sprintf("%s.json", language))
		_, err := os.Stat(languageJson)
		if err == nil {
			return languageJson, nil
		}
	}

	return "", errors.New(fmt.Sprintf("Failed to find the implementation for: %s", language))
}

func GetSkeletonFilePath(filename string) (string, error) {
	searchPaths := GetSearchPathForSharedFiles()
	for _, p := range searchPaths {
		skelFile := filepath.Join(p, "skel", filename)
		if FileExists(skelFile) {
			return skelFile, nil
		}
	}

	return "", errors.New(fmt.Sprintf("Failed to find the skeleton file: %s", filename))
}

func GetPluginsPath() (string, error) {
	searchPaths := GetSearchPathForSharedFiles()
	for _, p := range searchPaths {
		pluginsDir := filepath.Join(p, "plugins")
		if DirExists(pluginsDir) {
			return pluginsDir, nil
		}
	}

	return "", errors.New("Failed to find the plugins directory")
}

func IsASupportedLanguage(language string) bool {
	_, err := GetLanguageJSONFilePath(language)
	return err == nil
}

func ReadFileContents(file string) string {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("Failed to read: %s. %s\n", file, err.Error())
		os.Exit(1)
	}

	return string(bytes)
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

func DirExists(dirPath string) bool {
	stat, err := os.Stat(dirPath)
	if err == nil && stat.IsDir() {
		return true
	}

	return false
}

func GetUniqueId() int64 {
	return time.Now().UnixNano()
}

func CopyFile(src, dest string) error {
	if !FileExists(src) {
		return errors.New(fmt.Sprintf("%s doesn't exist", src))
	}

	b, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dest, b, NewFilePermissions)
	if err != nil {
		return err
	}

	return nil
}

// A wrapper around os.SetEnv
// This handles duplicate env variable assignments and fails
func SetEnvVariable(key, value string) error {
	existingValue := os.Getenv(key)
	if existingValue == "" {
		if strings.TrimSpace(value) == "" {
			return nil
		}
		err := os.Setenv(key, value)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to set: %s = %s. %s", key, value, err.Error()))
		}
	} else {
		return errors.New(fmt.Sprintf("Failed to set: %s = %s. It is already assigned a value '%s'. Multiple assignments to same variable is not allowed", key, value, existingValue))
	}

	return nil
}

func GetExecutableCommand(command string) *exec.Cmd {
	var cmd *exec.Cmd
	cmdParts := strings.Split(command, " ")
	if len(cmdParts) == 0 {
		panic(errors.New("Invalid executable command"))
	} else if len(cmdParts) > 1 {
		cmd = exec.Command(cmdParts[0], cmdParts[1:]...)
	} else {
		cmd = exec.Command(cmdParts[0])
	}
	return cmd
}

func downloadUsingWget(url, targetDir string) error {
	wgetCommand := fmt.Sprintf("wget %s --directory-prefix %s", url, targetDir)
	cmd := GetExecutableCommand(wgetCommand)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func downloadUsingCurl(url, targetDir string) error {
	curlCommand := fmt.Sprintf("curl -o %s %s", filepath.Join(targetDir, filepath.Base(url)), url)
	cmd := GetExecutableCommand(curlCommand)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func downloadUsingGo(url, targetDir string) error {
	out, err := os.Create(filepath.Join(targetDir, filepath.Base(url)))
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func Download(url, targetDir string) error {
	if !DirExists(targetDir) {
		return errors.New(fmt.Sprintf("%s doesn't exists", targetDir))
	}

	if _, err := exec.LookPath("wget"); err == nil {
		return downloadUsingWget(url, targetDir)
	}

	if _, err := exec.LookPath("curl"); err == nil {
		return downloadUsingCurl(url, targetDir)
	}

	return downloadUsingGo(url, targetDir)
}

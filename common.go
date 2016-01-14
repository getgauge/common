// Copyright 2015 ThoughtWorks, Inc.

// This file is part of getgauge/common.

// getgauge/common is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// getgauge/common is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with getgauge/common.  If not, see <http://www.gnu.org/licenses/>.

// Package common functions shared across all files
package common

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/dmotylev/goproperties"
)

const (
	ManifestFile            = "manifest.json"
	PluginJSONFile          = "plugin.json"
	NewDirectoryPermissions = 0755
	NewFilePermissions      = 0644
	DefaultEnvFileName      = "default.properties"
	EnvDirectoryName        = "env"
	DefaultEnvDir           = "default"
	ProductName             = "gauge"
	dotGauge                = ".gauge"
	SpecsDirectoryName      = "specs"
	ConceptFileExtension    = ".cpt"
	Plugins                 = "plugins"
	appData                 = "APPDATA"
	gaugePropertiesFile     = "gauge.properties"
)

const (
	GaugeProjectRootEnv      = "GAUGE_PROJECT_ROOT"
	GaugeRootEnvVariableName = "GAUGE_ROOT" //specifies the installation path if installs to non-standard location
	GaugePortEnvName         = "GAUGE_PORT" // user specifies this to use a specific port
	GaugeInternalPortEnvName = "GAUGE_INTERNAL_PORT"
	APIPortEnvVariableName   = "GAUGE_API_PORT"
	GaugeDebugOptsEnv        = "GAUGE_DEBUG_OPTS" //specify the debug options to be used while launching the runner
)

type Property struct {
	Name         string
	Comment      string
	DefaultValue string
}

// A project root is where a manifest.json files exists
// this routine keeps going upwards searching for manifest.json
func GetProjectRoot() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("Failed to find project root directory. %s\n", err.Error())
	}
	return findManifestInPath(pwd)
}
func findManifestInPath(pwd string) (string, error) {
	wd, err := filepath.Abs(pwd)
	if err != nil {
		return "", fmt.Errorf("Failed to find project directory: %s", err)
	}
	manifestExists := func(dir string) bool {
		return FileExists(path.Join(dir, ManifestFile))
	}
	dir := wd

	for {
		if manifestExists(dir) {
			return dir, nil
		}
		if dir == filepath.Clean(fmt.Sprintf("%c", os.PathSeparator)) || dir == "" {
			return "", fmt.Errorf("Failed to find project directory")
		}
		oldDir := dir
		dir = filepath.Clean(fmt.Sprintf("%s%c..", dir, os.PathSeparator))
		if dir == oldDir {
			return "", fmt.Errorf("Failed to find project directory")
		}
	}
}

func GetDirInProject(dirName string, specPath string) (string, error) {
	projectRoot, err := GetProjectRootFromSpecPath(specPath)
	if err != nil {
		return "", err
	}
	requiredDir := filepath.Join(projectRoot, dirName)
	if !DirExists(requiredDir) {
		return "", fmt.Errorf("Could not find %s directory. %s does not exist", dirName, requiredDir)
	}

	return requiredDir, nil
}

func GetProjectRootFromSpecPath(specPath string) (string, error) {
	projectRoot, err := GetProjectRoot()
	if err != nil {
		dir, _ := path.Split(specPath)
		fullPath, pathErr := filepath.Abs(dir)
		if pathErr != nil {
			return "", fmt.Errorf("Unable to get absolute path to specifications. %s", err)
		}
		return findManifestInPath(fullPath)
	}
	return projectRoot, err
}

func GetDefaultPropertiesFile() (string, error) {
	envDir, err := GetDirInProject(EnvDirectoryName, "")
	if err != nil {
		return "", err
	}
	defaultEnvFile := filepath.Join(envDir, DefaultEnvDir, DefaultEnvFileName)
	if !FileExists(defaultEnvFile) {
		return "", fmt.Errorf("Default environment file does not exist: %s \n", defaultEnvFile)
	}
	return defaultEnvFile, nil
}

func AppendProperties(propertiesFile string, properties ...*Property) error {
	file, err := os.OpenFile(propertiesFile, os.O_RDWR|os.O_APPEND, NewFilePermissions)
	if err != nil {
		return err
	}
	for _, property := range properties {
		file.WriteString(fmt.Sprintf("\n%s\n", property.String()))
	}
	return file.Close()
}

func FindFilesInDir(dirPath string, isValidFile func(path string) bool) []string {
	var files []string
	filepath.Walk(dirPath, func(path string, f os.FileInfo, err error) error {
		if err == nil && !f.IsDir() && isValidFile(path) {
			files = append(files, path)
		}
		return err
	})
	return files
}

// gets the installation directory prefix
// /usr or /usr/local or gauge_root
func GetInstallationPrefix() (string, error) {
	gaugeRoot := os.Getenv(GaugeRootEnvVariableName)
	if gaugeRoot != "" {
		return gaugeRoot, nil
	}
	var possibleInstallationPrefixes []string
	if isWindows() {
		programFilesPath := os.Getenv("PROGRAMFILES")
		if programFilesPath == "" {
			return "", fmt.Errorf("Cannot locate gauge shared file. Could not find Program Files directory.")
		}
		possibleInstallationPrefixes = []string{filepath.Join(programFilesPath, ProductName)}
	} else {
		possibleInstallationPrefixes = []string{"/usr/local", "/usr"}
	}

	for _, p := range possibleInstallationPrefixes {
		if FileExists(path.Join(p, "bin", ExecutableName())) {
			return p, nil
		}
	}

	return "", fmt.Errorf("Can't find installation files")
}

func ExecutableName() string {
	if isWindows() {
		return "gauge.exe"
	}
	return "gauge"
}

func GetSearchPathForSharedFiles() (string, error) {
	installationPrefix, err := GetInstallationPrefix()
	if err != nil {
		return "", err
	}
	return filepath.Join(installationPrefix, "share", ProductName), nil
}

func GetSkeletonFilePath(filename string) (string, error) {
	searchPath, err := GetSearchPathForSharedFiles()
	if err != nil {
		return "", err
	}
	skelFile := filepath.Join(searchPath, "skel", filename)
	if FileExists(skelFile) {
		return skelFile, nil
	}

	return "", fmt.Errorf("Failed to find the skeleton file: %s", filename)
}

func GetPluginsInstallDir(pluginName string) (string, error) {
	pluginInstallPrefixes, err := GetPluginInstallPrefixes()
	if err != nil {
		return "", err
	}

	for _, prefix := range pluginInstallPrefixes {
		if SubDirectoryExists(prefix, pluginName) {
			return prefix, nil
		}
	}
	return "", fmt.Errorf("Plugin '%s' not installed on following locations : %s", pluginName, pluginInstallPrefixes)
}

func SubDirectoryExists(pluginDir string, pluginName string) bool {
	files, err := ioutil.ReadDir(pluginDir)
	if err != nil {
		return false
	}

	for _, f := range files {
		if f.Name() == pluginName && f.IsDir() {
			return true
		}
	}
	return false
}

func GetPluginInstallPrefixes() ([]string, error) {
	primaryPluginInstallDir, err := GetPrimaryPluginsInstallDir()
	if err != nil {
		return nil, err
	}
	return []string{primaryPluginInstallDir}, nil
}

func GetGaugeHomeDirectory() (string, error) {
	if isWindows() {
		appDataDir := os.Getenv(appData)
		if appDataDir == "" {
			return "", fmt.Errorf("Failed to find plugin installation path. Could not get APPDATA")
		}
		return filepath.Join(appDataDir, ProductName), nil
	}
	userHome, err := getUserHome()
	if err != nil {
		return "", fmt.Errorf("Failed to find plugin installation path. Could not get User home directory: %s", err)
	}
	return filepath.Join(userHome, dotGauge), nil
}

func GetPrimaryPluginsInstallDir() (string, error) {
	gaugeHome, err := GetGaugeHomeDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(gaugeHome, Plugins), nil
}

func GetLibsPath() (string, error) {
	prefix, err := GetInstallationPrefix()
	if err != nil {
		return "", err
	}
	return filepath.Join(prefix, "lib", ProductName), nil
}

func IsPluginInstalled(name, version string) bool {
	pluginsDir, err := GetPluginsInstallDir(name)
	if err != nil {
		return false
	}
	return DirExists(path.Join(pluginsDir, name, version))
}

func GetGaugeConfiguration() (properties.Properties, error) {
	sharedDir, err := GetSearchPathForSharedFiles()
	if err != nil {
		return nil, err
	}
	propertiesFile := filepath.Join(sharedDir, gaugePropertiesFile)
	config, err := properties.Load(propertiesFile)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func ReadFileContents(file string) (string, error) {
	if !FileExists(file) {
		return "", fmt.Errorf("File %s doesn't exist.", file)
	}
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("Failed to read the file %s.", file)
	}

	return string(bytes), nil
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

// Modified version of bradfitz's camlistore (https://github.com/bradfitz/camlistore/blob/master/make.go)
func MirrorDir(src, dst string) ([]string, error) {
	var filesAdded []string
	err := filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		suffix, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("Failed to find Rel(%q, %q): %v", src, path, err)
		}

		err = MirrorFile(path, filepath.Join(dst, suffix))
		filesAdded = append(filesAdded, suffix)
		return err
	})
	return filesAdded, err
}

// Modified version of bradfitz's camlistore (https://github.com/bradfitz/camlistore/blob/master/make.go)
func MirrorFile(src, dst string) error {
	sfi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if sfi.Mode()&os.ModeType != 0 {
		log.Fatalf("mirrorFile can't deal with non-regular file %s", src)
	}
	dfi, err := os.Stat(dst)
	if err == nil &&
		isExecMode(sfi.Mode()) == isExecMode(dfi.Mode()) &&
		(dfi.Mode()&os.ModeType == 0) &&
		dfi.Size() == sfi.Size() &&
		dfi.ModTime().Unix() == sfi.ModTime().Unix() {
		// Seems to not be modified.
		return nil
	}

	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	df, err := os.Create(dst)
	if err != nil {
		return err
	}
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()

	n, err := io.Copy(df, sf)
	if err == nil && n != sfi.Size() {
		err = fmt.Errorf("copied wrong size for %s -> %s: copied %d; want %d", src, dst, n, sfi.Size())
	}
	cerr := df.Close()
	if err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(dst, sfi.Mode())
	}
	if err == nil {
		err = os.Chtimes(dst, sfi.ModTime(), sfi.ModTime())
	}
	return err
}

func isExecMode(mode os.FileMode) bool {
	return (mode & 0111) != 0
}

func GetUniqueID() int64 {
	return time.Now().UnixNano()
}

func CopyFile(src, dest string) error {
	if !FileExists(src) {
		return fmt.Errorf("%s doesn't exist", src)
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
func SetEnvVariable(key, value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	err := os.Setenv(key, value)
	if err != nil {
		return fmt.Errorf("Failed to set: %s = %s. %s", key, value, err.Error())
	}
	return nil
}

func ExecuteCommand(command []string, workingDir string, outputStreamWriter io.Writer, errorStreamWriter io.Writer) (*exec.Cmd, error) {
	cmd := prepareCommand(false, command, workingDir, outputStreamWriter, errorStreamWriter)
	err := cmd.Start()
	return cmd, err

}

func ExecuteSystemCommand(command []string, workingDir string, outputStreamWriter io.Writer, errorStreamWriter io.Writer) (*exec.Cmd, error) {
	cmd := prepareCommand(true, command, workingDir, outputStreamWriter, errorStreamWriter)
	err := cmd.Start()
	return cmd, err
}

func ExecuteCommandWithEnv(command []string, workingDir string, outputStreamWriter io.Writer, errorStreamWriter io.Writer, env []string) (*exec.Cmd, error) {
	cmd := prepareCommand(false, command, workingDir, outputStreamWriter, errorStreamWriter)
	cmd.Env = env
	err := cmd.Start()
	return cmd, err
}

func prepareCommand(isSystemCommand bool, command []string, workingDir string, outputStreamWriter io.Writer, errorStreamWriter io.Writer) *exec.Cmd {
	cmd := GetExecutableCommand(isSystemCommand, command...)
	cmd.Dir = workingDir
	cmd.Stdout = outputStreamWriter
	cmd.Stderr = errorStreamWriter
	cmd.Stdin = os.Stdin
	return cmd
}

func GetExecutableCommand(isSystemCommand bool, command ...string) *exec.Cmd {
	if len(command) == 0 {
		panic(fmt.Errorf("Invalid executable command"))
	}
	cmd := &exec.Cmd{Path: command[0]}
	if len(command) > 1 {
		if isSystemCommand {
			cmd = exec.Command(command[0], command[1:]...)
		}
		cmd.Args = append([]string{command[0]}, command[1:]...)
	} else {
		if isSystemCommand {
			cmd = exec.Command(command[0])
		}
		cmd.Args = append([]string{command[0]})
	}
	return cmd
}

func downloadUsingGo(url, targetFile string) error {
	out, err := os.Create(targetFile)
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

func Download(url, targetDir string) (string, error) {
	if !DirExists(targetDir) {
		return "", fmt.Errorf("%s doesn't exists", targetDir)
	}
	targetFile := filepath.Join(targetDir, filepath.Base(url))

	fileExist, err := UrlExists(url)
	if !fileExist {
		return "", err
	}

	return targetFile, downloadUsingGo(url, targetFile)
}

func DownloadToTempDir(url string) (string, error) {
	return Download(url, GetTempDir())
}

func GetTempDir() string {
	tempGaugeDir := filepath.Join(os.TempDir(), "gauge_temp")
	tempGaugeDir += strconv.FormatInt(time.Now().UnixNano(), 10)
	if !exists(tempGaugeDir) {
		os.MkdirAll(tempGaugeDir, NewDirectoryPermissions)
	}
	return tempGaugeDir
}

func Remove(path string) error {
	return os.RemoveAll(path)
}

func exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func UnzipArchive(zipFile string, dest string) (string, error) {
	if !FileExists(zipFile) {
		return "", fmt.Errorf("ZipFile %s does not exist", zipFile)
	}

	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		error := func() error {
			defer rc.Close()

			path := filepath.Join(dest, f.Name)
			os.MkdirAll(filepath.Dir(path), NewDirectoryPermissions)
			if f.FileInfo().IsDir() {
				os.MkdirAll(path, f.Mode())
			} else {
				f, err := os.OpenFile(
					path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
				if err != nil {
					return err
				}
				defer f.Close()

				_, err = io.Copy(f, rc)
				if err != nil {
					return err
				}
			}
			return nil
		}()
		if error != nil {
			return "", error

		}
	}

	return dest, nil
}

func SaveFile(filePath, contents string, takeBackup bool) error {
	backupFile := ""
	if takeBackup {
		tmpDir := os.TempDir()
		fileName := fmt.Sprintf("%s_%v", filepath.Base(filePath), GetUniqueID())
		backupFile = filepath.Join(tmpDir, fileName)
		err := CopyFile(filePath, backupFile)
		if err != nil {
			return fmt.Errorf("Failed to make backup for '%s': %s", filePath, err.Error())
		}
	}
	err := ioutil.WriteFile(filePath, []byte(contents), NewFilePermissions)
	if err != nil {
		return fmt.Errorf("Failed to write to '%s': %s", filePath, err.Error())
	}

	return nil
}

func getUserHome() (string, error) {
	usr, err := user.Current()
	if err != nil {
		homeFromEnv := getUserHomeFromEnv()
		if homeFromEnv != "" {
			return homeFromEnv, nil
		}
		return "", fmt.Errorf("Could not get the home directory")
	}
	return usr.HomeDir, nil
}

func getUserHomeFromEnv() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}

func TrimTrailingSpace(str string) string {
	var r = regexp.MustCompile(`[ \t]+$`)
	return r.ReplaceAllString(str, "")
}

func (property *Property) String() string {
	return fmt.Sprintf("#%s\n%s = %s", property.Comment, property.Name, property.DefaultValue)
}

func UrlExists(url string) (bool, error) {
	resp, err := http.Head(url)
	if err != nil {
		return false, fmt.Errorf("Failed to resolve host.")
	}
	if resp.StatusCode == 404 {
		return false, fmt.Errorf("File does not exist.")
	}
	return true, nil
}

func GetPluginProperties(jsonPropertiesFile string) (map[string]interface{}, error) {
	pluginPropertiesJSON, err := ioutil.ReadFile(jsonPropertiesFile)
	if err != nil {
		return nil, fmt.Errorf("Could not read %s: %s\n", filepath.Base(jsonPropertiesFile), err)
	}
	var pluginJSON interface{}
	if err = json.Unmarshal([]byte(pluginPropertiesJSON), &pluginJSON); err != nil {
		return nil, fmt.Errorf("Could not read %s: %s\n", filepath.Base(jsonPropertiesFile), err)
	}
	return pluginJSON.(map[string]interface{}), nil
}

func GetGaugePluginVersion(pluginName string) (string, error) {
	pluginProperties, err := GetPluginProperties(fmt.Sprintf("%s.json", pluginName))
	if err != nil {
		return "", fmt.Errorf("Failed to get gauge %s properties file. %s", pluginName, err)
	}
	return pluginProperties["version"].(string), nil
}

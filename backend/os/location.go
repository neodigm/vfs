package os

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/c2fo/vfs"
	"github.com/c2fo/vfs/utils"
)

//Location implements the vfs.Location interface specific to OS fs.
type Location struct {
	name       string
	fileSystem vfs.FileSystem
}

// NewFile uses the properties of the calling location to generate a vfs.File (backed by an os.File). A string
// argument is expected to be a relative path to the location's current path.
func (l *Location) NewFile(fileName string) (vfs.File, error) {
	return l.fileSystem.NewFile(l.Volume(), filepath.Join(l.Path(), fileName))
}

// DeleteFile deletes the file of the given name at the location. This is meant to be a short cut for instantiating a
// new file and calling delete on that with all the necessary error handling overhead.
func (l *Location) DeleteFile(fileName string) error {
	file, err := l.NewFile(fileName)
	if err != nil {
		return err
	}

	return file.Delete()
}

type fileTest func(fileName string) bool

// List returns a slice of all files in the top directory of of the location.
func (l *Location) List() ([]string, error) {
	return l.fileList(func(name string) bool { return true })
}

// ListByPrefix returns a slice of all files starting with "prefix" in the top directory of of the location.
func (l *Location) ListByPrefix(prefix string) ([]string, error) {
	if err := utils.ValidateFilePrefix(prefix); err != nil {
		return nil, err
	}
	return l.fileList(func(name string) bool {
		return strings.HasPrefix(name, prefix)
	})
}

// ListByRegex returns a slice of all files matching the regex in the top directory of of the location.
func (l *Location) ListByRegex(regex *regexp.Regexp) ([]string, error) {
	return l.fileList(func(name string) bool {
		return regex.MatchString(name)
	})
}

func (l *Location) fileList(testEval fileTest) ([]string, error) {
	files := make([]string, 0)
	exists, err := l.Exists()
	if err != nil {
		return files, err
	}

	// Function should return an empty slice if the directory doesn't exist. This is to match behavior of remote
	// systems. If the user cares about the distinction between directories that are empty, vs non-existent then
	// Location.Exists() should be used first.
	if exists {
		entries, err := ioutil.ReadDir(l.Path())
		if err != nil {
			return files, err
		}

		for _, info := range entries {
			if !info.IsDir() && testEval(info.Name()) {
				files = append(files, info.Name())
			}
		}
	}

	return files, nil
}

// Volume returns the volume, if any, of the location. Given "C:\foo\bar" it returns "C:" on Windows. On other platforms it returns "".
func (l *Location) Volume() string {
	return filepath.VolumeName(l.name)
}

// Path returns the location path.
func (l *Location) Path() string {
	return l.name
}

// Exists returns true if the location exists, and the calling user has the appropriate
// permissions. Will receive false without an error if the location simply doesn't exist. Otherwise could receive
// false and any errors passed back from the OS.
func (l *Location) Exists() (bool, error) {
	_, err := os.Stat(l.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// URI returns the Location's URI as a string.
func (l *Location) URI() string {
	return utils.GetLocationURI(l)
}

// String implement fmt.Stringer, returning the location's URI as the default string.
func (l *Location) String() string {
	return l.URI()
}

// NewLocation makes a copy of the underlying Location, then modifies its path by calling ChangeDir with the
// relativePath argument, returning the resulting location. The only possible errors come from the call to
// ChangeDir.
func (l *Location) NewLocation(relativePath string) (vfs.Location, error) {
	fullPath, err := filepath.Abs(filepath.Join(l.name, relativePath))
	if err != nil {
		return nil, err
	}

	fullPath = utils.AddTrailingSlash(fullPath)
	return &Location{
		name:       fullPath,
		fileSystem: l.fileSystem,
	}, nil
}

// ChangeDir takes a relative path, and modifies the underlying Location's path. The caller is modified by this
// so the only return is any error. For this implementation there are no errors.
func (l *Location) ChangeDir(relativePath string) error {
	l.name = filepath.Join(l.name, relativePath)
	return nil
}

// FileSystem returns a vfs.FileSystem interface of the location's underlying fileSystem.
func (l *Location) FileSystem() vfs.FileSystem {
	return l.fileSystem
}

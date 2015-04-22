package fs

// GenericFileError - generic file error
type GenericFileError struct {
	path string
}

// FileNotFound (ENOENT) - file not found
type FileNotFound GenericFileError

func (e FileNotFound) Error() string {
	return "Requested file " + e.path + " not found"
}

// FileISDir (EISDIR) - accessed file is a directory
type FileISDir GenericFileError

func (e FileISDir) Error() string {
	return "Requested file " + e.path + " is a directory"
}

// FileNotDir (ENOTDIR) - accessed file is not a directory
type FileNotDir GenericFileError

func (e FileNotDir) Error() string {
	return "Requested file " + e.path + " is not a directory"
}

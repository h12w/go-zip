// Copyright 2013, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package go-zip is a wrapper around C library libzip. It tries to mimic standard library "archive/zip" but provides ability to modify existing ZIP archives.

SEE: http://www.nih.at/libzip/index.html
*/
package zip

import (
	"io"
)

// Archive provides ability for reading, creating and modifying a ZIP archive.
type Archive struct {
	file string
	z    pzip
	last *fileWriter
}

// Open a ZIP archive, create a new one if not exists.
func Open(file string) (*Archive, error) {
	a := &Archive{
		file: file,
		last: nil}
	e := a.open_z()
	if e != nil {
		return nil, e
	}
	return a, nil
}

// Close z ZIP archive, and modifications get written to the disk when closing.
func (a *Archive) Close() error {
	err := a.flush()
	if err != nil {
		return err
	}
	return a.close_z()
}

// Add a file or directory in the ZIP archive.
// If name ends with '/', a directory will be added, and returned w will be nil.
// Otherwise, a file will be added, and w is a Writer to which the file contents should be written. 
func (a *Archive) Create(name string) (w io.Writer, err error) {
	err = a.flush()
	if err != nil {
		return nil, err
	}

	if len(name) > 0 && name[len(name)-1] == '/' {
		return a.createDirectory(name)
	}
	return a.createFile(name)
}

func (a *Archive) createDirectory(name string) (io.Writer, error) {
	_, err := zip_add_dir(a.z, name)
	return nil, err
}

func (a *Archive) createFile(name string) (w io.Writer, err error) {
	a.last, err = newFileWriter()
	if err != nil {
		return nil, err
	}

	go func() {
		err := zip_add_fd(a.z, name, a.last.rpipe.Fd())
		if err != nil {
			goto Return
		}
		err = a.reopen_z()
	Return:
		a.last.done <- err
	}()
	return a.last, nil
}

func (a *Archive) flush() error {
	if a.last != nil {
		err := a.last.Close()
		a.last = nil
		return err
	}
	return nil
}

func (a *Archive) open_z() (err error) {
	if a.z == nil {
		a.z, err = zip_open(a.file, _ZIP_CREATE)
	}
	return
}

func (a *Archive) close_z() error {
	if a.z != nil {
		err := zip_close(a.z)
		a.z = nil
		return err
	}
	return nil
}

func (a *Archive) reopen_z() error {
	err := a.close_z()
	if err != nil {
		return err
	}
	return a.open_z()
}

// ZIP entry count in the ZIP archive.
func (a *Archive) Count() int64 {
	c, err := zip_get_num_entries(a.z, 0)
	if err != nil {
		return 0
	}
	return c
}

// Delete a ZIP entry in the ZIP archive.
func (a *Archive) Delete(name string) error {
	index, err := zip_name_locate(a.z, name, 0)
	if err != nil {
		return err
	}
	return zip_delete(a.z, uint64(index))
}

// Rename a ZIP entry in the ZIP archive.
func (a *Archive) Rename(from, to string) error {
	index, err := zip_name_locate(a.z, from, 0)
	if err != nil {
		return err
	}
	return zip_rename(a.z, uint64(index), to)
}

// Get a file from the ZIP archive.
func (a *Archive) File(index uint64) (*File, error) {
	h, err := zip_file_header(a.z, index)
	if err != nil {
		return nil, err
	}
	return &File{h, a}, nil
}


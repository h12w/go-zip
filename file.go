// Copyright 2013, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zip

import (
	"io"
	"os"
)

// File provides ability to read a file in the ZIP archive.
type File struct {
	*FileHeader
	a *Archive
}

// Open returns a ReadCloser that provides access to the File's contents.
func (f *File) Open() (rc io.ReadCloser, err error) {
	fr, err := zip_fopen(f.a.z, f.Name, 0)
	if err != nil {
		return nil, err
	}
	return &fileReader{fr}, nil
}

type fileWriter struct {
	rpipe *os.File
	wpipe *os.File
	done  chan error
}

func newFileWriter() (w *fileWriter, err error) {
	w = &fileWriter{
		rpipe: nil,
		wpipe: nil,
		done:  make(chan error)}
	w.rpipe, w.wpipe, err = os.Pipe()
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (w *fileWriter) Write(p []byte) (nn int, err error) {
	return w.wpipe.Write(p)
}

func (w *fileWriter) Close() error {
	w.wpipe.Close()
	w.wpipe = nil
	err := <-w.done
	w.rpipe.Close()
	w.rpipe = nil
	return err
}

type fileReader struct {
	f pzip_file
}

func (r *fileReader) Read(b []byte) (int, error) {
	n, err := zip_fread(r.f, b)
	return int(n), err
}

func (r *fileReader) Close() error {
	return zip_fclose(r.f)
}


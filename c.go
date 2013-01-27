// Copyright 2013, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zip

/*
#cgo LDFLAGS: -lzip
#include <zip.h>
#include <stdlib.h>
*/
import "C"

import (
	"syscall"
	"time"
	"unsafe"
)

// Compression methods.
const (
	Store   uint16 = C.ZIP_CM_STORE
	Deflate uint16 = C.ZIP_CM_DEFLATE
)

type (
	pzip        *C.struct_zip
	pzip_source *C.struct_zip_source
	pzip_file   *C.struct_zip_file
)

type FileHeader struct {
	Name             string
	Flags            uint32
	Method           uint16
	ModifiedTime     time.Time
	CRC32            uint32
	CompressedSize   uint64
	UncompressedSize uint64
}

// Flags of zip_open
const (
	_ZIP_CREATE = C.ZIP_CREATE
)

func zip_open(path string, flags C.int) (pzip, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var ze C.int
	z, err := C.zip_open(cpath, flags, &ze)
	if z == nil {
		se := error_to_errno(err)
		return nil, ze_se_to_error(ze, se)
	}
	return z, nil
}

func zip_close(z pzip) error {
	if -1 == C.zip_close(z) {
		return zip_error(z)
	}
	return nil
}

func zip_source_file(z pzip, fname string, start uint64, length int64) (pzip_source, error) {
	cfname := C.CString(fname)
	defer C.free(unsafe.Pointer(cfname))
	s := C.zip_source_file(z, cfname, C.zip_uint64_t(start), C.zip_int64_t(length))
	if s == nil {
		return nil, zip_error(z)
	}
	return s, nil
}

func zip_source_filep(z pzip, file *C.FILE, start uint64, length int64) (pzip_source, error) {
	s := C.zip_source_filep(z, file, C.zip_uint64_t(start), C.zip_int64_t(length))
	if s == nil {
		return nil, zip_error(z)
	}
	return s, nil
}

// If not added successfully, free it. Otherwise, don't call it.
func zip_source_free(s pzip_source) {
	C.zip_source_free(s)
}

func zip_add(z pzip, name string, s pzip_source) (int64, error) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	index := int64(C.zip_add(z, cname, s))
	if index < 0 {
		return index, zip_error(z)
	}
	return index, nil
}

func zip_add_fd(z pzip, name string, fd uintptr) error {
	mode := [...]C.char{'r', 0}
	file := C.fdopen(C.int(fd), &mode[0])
	s, err := zip_source_filep(z, file, 0, -1)
	if s == nil {
		return err
	}
	index, err := zip_add(z, name, s)
	if index < 0 {
		zip_source_free(s)
		return err
	}
	return nil
}

func zip_add_dir(z pzip, name string) (int64, error) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	index := int64(C.zip_add_dir(z, cname))
	if index < 0 {
		return index, zip_error(z)
	}
	return index, nil
}

func zip_get_num_entries(z pzip, flags C.zip_flags_t) (int64, error) {
	num := int64(C.zip_get_num_entries(z, flags))
	if num < 0 {
		return num, zip_error(z)
	}
	return num, nil
}

func zip_name_locate(z pzip, fname string, flags C.zip_flags_t) (int64, error) {
	cfname := C.CString(fname)
	defer C.free(unsafe.Pointer(cfname))
	index := int64(C.zip_name_locate(z, cfname, flags))
	if index < 0 {
		return index, zip_error(z)
	}
	return index, nil
}

func zip_delete(z pzip, index uint64) error {
	if 0 != C.zip_delete(z, C.zip_uint64_t(index)) {
		return zip_error(z)
	}
	return nil
}

func zip_rename(z pzip, index uint64, name string) error {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	if 0 != C.zip_rename(z, C.zip_uint64_t(index), cname) {
		return zip_error(z)
	}
	return nil
}

func zip_fopen_index(z pzip, index uint64, flags C.zip_flags_t) (pzip_file, error) {
	f := C.zip_fopen_index(z, C.zip_uint64_t(index), flags)
	if f == nil {
		return nil, zip_error(z)
	}
	return f, nil
}

func zip_fopen(z pzip, name string, flags C.zip_flags_t) (pzip_file, error) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	f := C.zip_fopen(z, cname, flags)
	if f == nil {
		return nil, zip_error(z)
	}
	return f, nil
}

func zip_fread(f pzip_file, b []byte) (int64, error) {
	n := C.zip_fread(f, unsafe.Pointer(&b[0]), C.zip_uint64_t(len(b)))
	if n == -1 {
		return 0, zip_file_error(f)
	}
	return int64(n), nil
}

func zip_fclose(f pzip_file) error {
	ze, err := C.zip_fclose(f)
	if ze != 0 {
		se := error_to_errno(err)
		return ze_se_to_error(ze, se)
	}
	return nil
}

func zip_stat_index(z pzip, index uint64, flags C.zip_flags_t) (*C.struct_zip_stat, error) {
	var s C.struct_zip_stat
	if 0 != C.zip_stat_index(z, C.zip_uint64_t(index), flags, &s) {
		return nil, zip_error(z)
	}
	return &s, nil
}

func zip_file_header(z pzip, index uint64) (*FileHeader, error) {
	s, err := zip_stat_index(z, index, 0)
	if err != nil {
		return nil, err
	}

	h := &FileHeader{}
	if C.ZIP_STAT_NAME&s.valid != 0 {
		h.Name = C.GoString(s.name)
	}
	if C.ZIP_STAT_FLAGS&s.valid != 0 {
		h.Flags = uint32(s.flags)
	}
	if C.ZIP_STAT_COMP_METHOD&s.valid != 0 {
		h.Method = uint16(s.comp_method)
	}
	if C.ZIP_STAT_MTIME&s.valid != 0 {
		h.ModifiedTime = time.Unix(int64(s.mtime), 0)
	}
	if C.ZIP_STAT_CRC&s.valid != 0 {
		h.CRC32 = uint32(s.crc)
	}
	if C.ZIP_STAT_COMP_SIZE&s.valid != 0 {
		h.CompressedSize = uint64(s.comp_size)
	}
	if C.ZIP_STAT_SIZE&s.valid != 0 {
		h.UncompressedSize = uint64(s.size)
	}
	return h, nil
}

/*
error handling
==============

After each call return, firstly check the return value:
	1. If valid, return nil error without converting the error code
	2. If invalid, convert the error code to error
Because sometimes, the error code is mistakenly set even if the return value is good. e.g.
	_zip_file_replace calls _zip_set_name which set the error code.
*/

// Error wraps all the errors returned by C library libzip.
type Error struct {
	ze, se C.int
	s      string
}

func (e Error) Error() string {
	return e.s
}

func ze_se_to_error(ze, se C.int) error {
	buf := []C.char{0}
	bufLen := C.zip_error_to_str(&buf[0], 1, ze, se) + 1
	buf = make([]C.char, bufLen)
	C.zip_error_to_str(&buf[0], C.zip_uint64_t(bufLen), ze, se)
	return Error{ze, se, C.GoString(&buf[0])}
}

func error_to_errno(err error) C.int {
	if syserr, ok := err.(syscall.Errno); ok {
		return C.int(syserr)
	}
	return 0
}

func zip_error(z pzip) error {
	ze, se := zip_error_get(z)
	if ze == C.ZIP_ER_OK {
		return nil
	}
	return ze_se_to_error(ze, se)
}

func zip_file_error(f pzip_file) error {
	ze, se := zip_file_error_get(f)
	if ze == C.ZIP_ER_OK {
		return nil
	}
	return ze_se_to_error(ze, se)
}

func zip_error_get(z pzip) (ze, se C.int) {
	C.zip_error_get(z, &ze, &se)
	return
}

func zip_file_error_get(f pzip_file) (ze, se C.int) {
	C.zip_file_error_get(f, &ze, &se)
	return
}

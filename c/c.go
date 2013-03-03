// Copyright 2013, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package c

/*
#cgo LDFLAGS: -lzip
#include <zip.h>
#include <stdlib.h>
*/
import "C"

import (
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// Compression methods.
const (
	Store   uint16 = C.ZIP_CM_STORE
	Deflate uint16 = C.ZIP_CM_DEFLATE
)

type Zip struct {
	Path string
	p    *C.struct_zip
	mu   sync.Mutex
}

type zipSource struct {
	p *C.struct_zip_source
}

type ZipFile struct {
	p *C.struct_zip_file
	z *Zip
}

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

// notes: all the exported methods should be locked.

func (z *Zip) Open() (err error) {
	z.lock()
	defer z.unlock()
	cpath := C.CString(z.Path)
	defer C.free(unsafe.Pointer(cpath))
	var ze C.int
	z.p, err = C.zip_open(cpath, _ZIP_CREATE, &ze)
	if z == nil {
		se := error_to_errno(err)
		return ze_se_to_error(ze, se)
	}
	return nil
}

func (z *Zip) lock() {
	z.mu.Lock()
}

func (z *Zip) unlock() {
	z.mu.Unlock()
}

func (z *Zip) Close() error {
	z.lock()
	defer z.unlock()
	if z.p != nil {
		if -1 == C.zip_close(z.p) {
			return z.error()
		}
	}
	return nil
}

func (z *Zip) error() error {
	// should not lock
	var ze, se C.int
	C.zip_error_get(z.p, &ze, &se)
	if ze == C.ZIP_ER_OK {
		return nil
	}
	return ze_se_to_error(ze, se)
}

func (z *Zip) sourceFileP(file *C.FILE, start uint64, length int64) (*zipSource, error) {
	s := C.zip_source_filep(z.p, file, C.zip_uint64_t(start), C.zip_int64_t(length))
	if s == nil {
		return nil, z.error()
	}
	return &zipSource{s}, nil
}

// If not added successfully, free it. Otherwise, don't call it.
func (s *zipSource) free() {
	C.zip_source_free(s.p)
}

func (z *Zip) add(name string, s *zipSource) (int64, error) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	index := int64(C.zip_add(z.p, cname, s.p))
	if index < 0 {
		return index, z.error()
	}
	return index, nil
}

func (z *Zip) AddFd(name string, fd uintptr) error {
	z.lock()
	defer z.unlock()
	mode := [...]C.char{'r', 0}
	file := C.fdopen(C.int(fd), &mode[0])
	s, err := z.sourceFileP(file, 0, -1)
	if s == nil {
		return err
	}
	index, err := z.add(name, s)
	if index < 0 {
		s.free()
		return err
	}
	return nil
}

func (z *Zip) AddDir(name string) (int64, error) {
	z.lock()
	defer z.unlock()
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	index := int64(C.zip_add_dir(z.p, cname))
	if index < 0 {
		return index, z.error()
	}
	return index, nil
}

func (z *Zip) GetNumEntries(flags C.zip_flags_t) (int64, error) {
	z.lock()
	defer z.unlock()
	num := int64(C.zip_get_num_entries(z.p, flags))
	if num < 0 {
		return num, z.error()
	}
	return num, nil
}

func (z *Zip) NameLocate(fname string, flags C.zip_flags_t) (int64, error) {
	z.lock()
	defer z.unlock()
	cfname := C.CString(fname)
	defer C.free(unsafe.Pointer(cfname))
	index := int64(C.zip_name_locate(z.p, cfname, flags))
	if index < 0 {
		return index, z.error()
	}
	return index, nil
}

func (z *Zip) Delete(index uint64) error {
	z.lock()
	defer z.unlock()
	if 0 != C.zip_delete(z.p, C.zip_uint64_t(index)) {
		return z.error()
	}
	return nil
}

func (z *Zip) Rename(index uint64, name string) error {
	z.lock()
	defer z.unlock()
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	if 0 != C.zip_rename(z.p, C.zip_uint64_t(index), cname) {
		return z.error()
	}
	return nil
}

func (z *Zip) FopenIndex(index uint64, flags C.zip_flags_t) (*ZipFile, error) {
	z.lock()
	defer z.unlock()
	f := C.zip_fopen_index(z.p, C.zip_uint64_t(index), flags)
	if f == nil {
		return nil, z.error()
	}
	return &ZipFile{f, z}, nil
}

func (z *Zip) Fopen(name string, flags C.zip_flags_t) (*ZipFile, error) {
	z.lock()
	defer z.unlock()
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	f := C.zip_fopen(z.p, cname, flags)
	if f == nil {
		return nil, z.error()
	}
	return &ZipFile{f, z}, nil
}

func (f *ZipFile) Read(b []byte) (int64, error) {
	f.z.lock()
	defer f.z.unlock()
	n := C.zip_fread(f.p, unsafe.Pointer(&b[0]), C.zip_uint64_t(len(b)))
	if n == -1 {
		return 0, f.error()
	}
	return int64(n), nil
}

func (f *ZipFile) Close() error {
	f.z.lock()
	defer f.z.unlock()
	ze, err := C.zip_fclose(f.p)
	if ze != 0 {
		se := error_to_errno(err)
		return ze_se_to_error(ze, se)
	}
	return nil
}

func (f *ZipFile) error() error {
	var ze, se C.int
	C.zip_file_error_get(f.p, &ze, &se)
	if ze == C.ZIP_ER_OK {
		return nil
	}
	return ze_se_to_error(ze, se)
}

func (z *Zip) statIndex(index uint64, flags C.zip_flags_t) (*C.struct_zip_stat, error) {
	var s C.struct_zip_stat
	if 0 != C.zip_stat_index(z.p, C.zip_uint64_t(index), flags, &s) {
		return nil, z.error()
	}
	return &s, nil
}

func (z *Zip) FileHeader(index uint64) (*FileHeader, error) {
	z.lock()
	defer z.unlock()
	s, err := z.statIndex(index, 0)
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

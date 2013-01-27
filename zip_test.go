// Copyright 2013, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zip

import (
	"os"
	"path"
	"testing"
)

func TestWriteRead(t *testing.T) {
	file := testFile()

	// Write
	func() {
		a, e := Open(file)
		c(e, t)
		defer a.Close()
		fw, e := a.Create("a/file")
		c(e, t)
		fw.Write([]byte("Hello, world!\n"))
		fw.Write([]byte("你好, 世界!\n"))
	}()

	// Read
	func() {
		a, e := Open(file)
		c(e, t)
		defer a.Close()
		f, e := a.File(0)
		c(e, t)

		if f.Name != "a/file" {
			t.Fatalf("Expected file name a/file, got %s", f.Name)
		}

		r, e := f.Open()
		c(e, t)
		defer r.Close()

		buf := make([]byte, 100)
		n, e := r.Read(buf)
		text := string(buf[:n])
		if text != "Hello, world!\n你好, 世界!\n" {
			t.Fatalf("Text mismatch, got %s", text)
		}
	}()
}

func c(e error, t *testing.T) {
	if e != nil {
		p(e)
		t.FailNow()
	}
}

func testFile() string {
	file := path.Join(os.TempDir(), "test-go-zip.zip")
	os.Remove(file)
	return file
}

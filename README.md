GO-ZIP
======

A wrapper around C library libzip. It tries to mimic standard library "archive/zip" but provides ability to modify existing ZIP archives. 

libzip official site: http://www.nih.at/libzip/index.html

Quick Start
-----------

###Install libzip
Please install the development version from the trunk, rather than current stable libzip 0.10.1 released on 2012-03-20. Otherwise, the writing will not function correctly.
####In Arch Linux
    packer -S libzip-hg
####In Others
    ......
###Get the package

    go get -u "github.com/hailiang/go-zip"

###Import the package

    import "github.com/hailiang/go-zip"

###Open, read, modify, close

####Open

    a, err := Open("a.zip")

####Read

    for i := 0; i < a.Count(); i++ {
        file, err := a.File(i)
        // ......
    }

####Modify

    fw, err := a.Create("afile")
    fw.Write("Hello world!")

####Close

    a.Close()

##Complete Documentation
http://go.pkgdoc.org/github.com/hailiang/go-zip


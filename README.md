GO-ZIP
======

A wrapper around C library libzip. It tries to mimic standard library "archive/zip" but provides ability to modify existing ZIP archives. 

Quick Start
-----------

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


package main

/*
#cgo LDFLAGS: -lGL -lX11
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <GL/gl.h>
#include <GL/glx.h>
#include <stdlib.h>

int Test() {
	return 1;
}

*/
import "C"
import (
	"fmt"
	"log"
	"runtime"
)

func main() {
	runtime.LockOSThread()

	display := C.XOpenDisplay(nil)
	if display == nil {
		log.Fatal("Cannot connect to X server")
	}
	defer C.XCloseDisplay(display)

	result := C.Test()
	fmt.Println(result)
}

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

// GLX Context Extensions (for OpenGL versions > 2.1)
typedef struct {
    int contextMajorVersion;
    int contextMinorVersion;
    int contextFlags;
    int profileMask;
} GLXContextAttributes;

PFNGLXCREATECONTEXTATTRIBSARBPROC glXCreateContextAttribsARB = 0;

GLXContext createGLXContext(Display *display, GLXFBConfig config, GLXContext shareList, Bool direct, GLXContextAttributes attribs) {
    int attribList[] = {
        0x2091, attribs.contextMajorVersion, // GLX_CONTEXT_MAJOR_VERSION_ARB
        0x2092, attribs.contextMinorVersion, // GLX_CONTEXT_MINOR_VERSION_ARB
        0x9126, attribs.profileMask,         // GLX_CONTEXT_PROFILE_MASK_ARB
        0
    };

    if (!glXCreateContextAttribsARB) {
        glXCreateContextAttribsARB = (PFNGLXCREATECONTEXTATTRIBSARBPROC)glXGetProcAddressARB((const GLubyte *) "glXCreateContextAttribsARB");
    }

    return glXCreateContextAttribsARB(display, config, shareList, direct, attribList);
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

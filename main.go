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
	"github.com/go-gl/gl/v3.3-core/gl"
	"log"
	"runtime"
	"unsafe"
)

func main() {
	runtime.LockOSThread()

	display := C.XOpenDisplay(nil)
	if display == nil {
		log.Fatal("Cannot connect to X server")
	}
	defer C.XCloseDisplay(display)

	// Get the default screen
	screen := C.XDefaultScreen(display)

	// Define GLX attributes
	visualAttribs := []C.int{
		C.GLX_RGBA,           // Request RGBA colors
		C.GLX_DEPTH_SIZE, 24, // Depth size
		C.GLX_DOUBLEBUFFER, 1, // Double-buffered rendering
		0, // Null-terminate
	}

	// Retrieve framebuffer configs
	var fbCount C.int
	fbConfigs := C.glXChooseFBConfig(display, screen, &visualAttribs[0], &fbCount)
	if fbConfigs == nil || fbCount == 0 {
		log.Fatal("Failed to retrieve framebuffer config")
	}

	// Cast the pointer to an array and access the first framebuffer config
	fbConfig := (*[1 << 28]C.GLXFBConfig)(unsafe.Pointer(fbConfigs))[:fbCount:fbCount][0]

	// Get a visual from the framebuffer config
	visualInfo := C.glXGetVisualFromFBConfig(display, fbConfig)
	if visualInfo == nil {
		log.Fatal("Failed to get visual info")
	}
	defer C.XFree(unsafe.Pointer(visualInfo))

	//GetOpenGLMax(display, screen)
	result := C.Test()
	fmt.Println(result)
}

// isGeometryShaderSupported checks for geometry shader support in OpenGL
func isGeometryShaderSupported() {
	// Fallback: Check for ARB_geometry_shader4 or EXT_geometry_shader4 extension in older OpenGL versions
	extensions := gl.GoStr(gl.GetString(gl.EXTENSIONS))

	fmt.Println(extensions)
}

func GetOpenGLMax(display *C.Display, screen C.int) {
	// Set minimal visual attributes for OpenGL context
	attributes := []C.int{
		C.GLX_RGBA,
		C.GLX_DEPTH_SIZE, 24,
		C.GLX_DOUBLEBUFFER,
		0,
	}
	visual := C.glXChooseVisual(display, C.int(screen), &attributes[0])
	if visual == nil {
		log.Fatal("No appropriate visual found")
	}

	ctx := C.glXCreateContext(display, visual, nil, C.True)
	if ctx == nil {
		log.Fatal("Failed to create an OpenGL context")
	}
	defer C.glXDestroyContext(display, ctx)

	// Create a dummy window
	root := C.XRootWindow(display, C.int(screen))
	win := C.XCreateSimpleWindow(display, root, 0, 0, 1, 1, 0, 0, 0)
	C.glXMakeCurrent(display, C.GLXDrawable(win), ctx)
	defer C.XDestroyWindow(display, win)

	// Get OpenGL version
	version := C.GoString((*C.char)(unsafe.Pointer(C.glGetString(C.GL_VERSION))))
	shaderVersion := C.GoString((*C.char)(unsafe.Pointer(C.glGetString(C.GL_SHADING_LANGUAGE_VERSION))))
	renderer := C.GoString((*C.char)(unsafe.Pointer(C.glGetString(C.GL_RENDERER))))
	vendor := C.GoString((*C.char)(unsafe.Pointer(C.glGetString(C.GL_VENDOR))))

	// Display OpenGL version information
	log.Printf("OpenGL Version: %s", version)
	log.Printf("GLSL Version: %s", shaderVersion)
	log.Printf("Renderer: %s", renderer)
	log.Printf("Vendor: %s", vendor)
	isGeometryShaderSupported()
}

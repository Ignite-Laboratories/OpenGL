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
	"strings"
	"time"
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
		C.GLX_X_RENDERABLE, 1, // Ensure renderable
		C.GLX_RENDER_TYPE, C.GLX_RGBA_BIT,
		C.GLX_DRAWABLE_TYPE, C.GLX_WINDOW_BIT,
		C.GLX_X_VISUAL_TYPE, C.GLX_TRUE_COLOR,
		C.GLX_RED_SIZE, 8,
		C.GLX_GREEN_SIZE, 8,
		C.GLX_BLUE_SIZE, 8,
		C.GLX_DEPTH_SIZE, 24,
		C.GLX_DOUBLEBUFFER, 1,
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

	// Create a window using the visual info
	root := C.XRootWindow(display, C.int(screen))
	swa := C.XSetWindowAttributes{
		colormap:   C.XCreateColormap(display, root, visualInfo.visual, C.AllocNone),
		event_mask: C.ExposureMask | C.KeyPressMask,
	}

	width, height := 800, 600
	win := C.XCreateWindow(
		display,
		root,
		0, 0,
		C.uint(width), C.uint(height),
		0, C.int(visualInfo.depth),
		C.InputOutput,
		visualInfo.visual,
		C.CWColormap|C.CWEventMask,
		&swa,
	)
	C.XMapWindow(display, win)

	// Create a GLX context for OpenGL ES 3.2
	contextAttribs := C.GLXContextAttributes{
		contextMajorVersion: 3,   // Request OpenGL ES major version 3
		contextMinorVersion: 1,   // Request OpenGL ES minor version 2
		profileMask:         0x4, // GLX_CONTEXT_ES2_PROFILE_BIT_EXT for OpenGL ES
	}
	glxContext := C.createGLXContext(display, fbConfig, nil, C.True, contextAttribs)
	if glxContext == nil {
		log.Fatal("Failed to create OpenGL 3.3 Core context")
	}

	// Make the context current
	if ok := C.glXMakeCurrent(display, C.GLXDrawable(win), glxContext); ok == 0 {
		log.Fatal("Failed to make OpenGL context current")
	}

	// Initialize OpenGL
	if err := gl.Init(); err != nil {
		log.Fatalf("Failed to initialize OpenGL: %v", err)
	}
	log.Printf("OpenGL initialized. Version: %s", gl.GoStr(gl.GetString(gl.VERSION)))

	ver := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println(ver)

	numExtensions := int32(0)
	gl.GetIntegerv(gl.NUM_EXTENSIONS, &numExtensions)

	for i := int32(0); i < numExtensions; i++ {
		extension := gl.GoStr(gl.GetStringi(gl.EXTENSIONS, uint32(i)))
		if strings.Contains(extension, "geometry") {
			fmt.Println(extension)
		}
	}

	// Render loop
	for {
		// Clear the screen
		gl.ClearColor(0.2, 0.3, 0.4, 1.0) // RGB color
		gl.Clear(gl.COLOR_BUFFER_BIT)

		// Swap buffers
		C.glXSwapBuffers(display, C.GLXDrawable(win))

		// Wait a bit (~60 FPS)
		time.Sleep(16 * time.Millisecond)
	}
}

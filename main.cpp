#include <GL/glew.h>
#include <GLFW/glfw3.h>
#include <iostream>

int main()
{
    // Initialize GLFW
    if (!glfwInit())
    {
        std::cerr << "GLFW initialization failed!" << std::endl;
        return -1;
    }

      // Create a GLFW windowed mode window and its OpenGL context
    GLFWwindow* window = glfwCreateWindow(800,600, "OpenGL Setup", nullptr, nullptr);
    if (!window)
    {
        std::cerr << "Failed to open GLFW window!" << std::endl;
        glfwTerminate();
        return -1;
    }

    // Make the window's context current
    glfwMakeContextCurrent(window);
    glfwSwapInterval(1); // Enable vsync

    // Initialize GLEW
    if (glewInit() != GLEW_OK)
    {
        std::cerr << "GLEW initialization failed!" << std::endl;
        return -1;
    }

    // Rendering loop
    while (!glfwWindowShouldClose(window))
    {
        glClearColor(0.2f, 0.3f, 0.3f, 1.0f); // Set clear color
        glClear(GL_COLOR_BUFFER_BIT); // Clear the screen

        // Draw here

        glfwSwapBuffers(window); // Swap buffers
        glfwPollEvents(); // Poll for input events
    }

    glfwTerminate(); // Clean up and close the window
    return 0;
}
cmake_minimum_required(VERSION 3.30)
project(OpenGLProject)

set(CMAKE_CXX_STANDARD 20)

add_executable(OpenGLProject main.cpp)
find_package(GLEW REQUIRED)
find_package(PkgConfig REQUIRED)
pkg_search_module(GLFW REQUIRED glfw3)
find_package(OpenGL REQUIRED)
find_package(glfw3 REQUIRED)
target_include_directories(${PROJECT_NAME} PRIVATE ${GLEW_INCLUDE_DIRS} ${GLFW_INCLUDE_DIRS})
target_link_libraries(${PROJECT_NAME} PRIVATE ${GLEW_LIBRARIES} ${GLFW_LIBRARIES} ${OPENGL_LIBRARIES})
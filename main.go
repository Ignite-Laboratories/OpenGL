package main

/*
int Test() {
	return 1;
}

*/
import "C"
import "fmt"

func main() {
	result := C.Test()
	fmt.Println(result)
}

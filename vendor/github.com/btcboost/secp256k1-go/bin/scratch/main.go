package main

/*
#include <stdio.h>
#include <stdlib.h>
void doitnowman(unsigned char* key) {
	key[0] = 0x41;
}
*/
import "C"

import "unsafe"
import "log"

func modify(bytes []byte) {
	C.doitnowman((*C.uchar)(unsafe.Pointer(&bytes[0])))
}

func main() {
	buf := []byte(`0BCDABCD`)
	log.Println(buf)
	modify(buf)
	log.Println(buf)

}

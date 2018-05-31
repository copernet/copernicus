package main

/*
#include <stdio.h>
#include <stdlib.h>


void sum_int(int * result, const int * const *pubnonces, size_t n) {
	printf("SUM\n");
	int i;
	printf("aaaa\n");

	for (i = 0; i < n; i++) {
		printf("loop %d \n", i);
		printf("v %d \n", *pubnonces[i]);
		//printf("v %d \n", (int*)*pubnonces[i]);
	    *result += *pubnonces[i];
	}
	printf("nonce %d \n",*result);

}

static int**makeIntArray(int size) {
        return calloc(sizeof(int*), size);
}

static void setArrayInt(int **a, int *s, int n) {
        a[n] = s;
}

static void freeIntArray(int **a, int size) {
        int i;
        for (i = 0; i < size; i++)
                free(a[i]);
        free(a);
}


*/
import "C"

import "log"

func sum(ints []int) int {
	cargs := C.makeIntArray(C.int(len(ints)))

	for i := 0; i < len(ints); i++ {
		intval := C.int(ints[i])
		C.setArrayInt(cargs, &intval, C.int(i))
	}

	result := C.int(0)
	C.sum_int(&result, cargs, C.size_t(len(ints)))

	return int(result)
}

func main() {

	arr := []int{1, 2, 3}
	log.Println(arr)
	result := sum(arr)
	log.Println(arr)
	log.Println(result)

}

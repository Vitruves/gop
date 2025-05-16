/**
 * Sample C file for testing
 */
#include <stdio.h>
#include "./sample.h"

// TODO: Implement error handling
int add(int a, int b) {
    return a + b;
}

// FIXME: This function needs optimization
int multiply(int a, int b) {
    int result = 0;
    for (int i = 0; i < b; i++) {
        result += a;
    }
    return result;
}

// This is a duplicate function
void process_data(int* data, int size) {
    for (int i = 0; i < size; i++) {
        data[i] *= 2;
    }
}

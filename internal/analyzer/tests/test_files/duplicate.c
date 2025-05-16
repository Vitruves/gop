/**
 * Duplicate file for testing
 */
#include <stdio.h>

// This is a duplicate function with minor changes
void process_data(int* data, int length) {
    // Same functionality but with different variable name
    for (int i = 0; i < length; i++) {
        data[i] *= 2;
    }
}

// Another function with a duplicated name
int add(float a, float b) {
    return (int)(a + b);
}

#include "test_duplicate.h"
#include <stdio.h>

// Function implementations - these should be prioritized over declarations
int duplicate_function(int a, int b) {
    return a + b;
}

void another_function(const char* str) {
    printf("Processing: %s\n", str);
}

double calculate_average(double* values, int count) {
    if (count == 0) return 0.0;
    
    double sum = 0.0;
    for (int i = 0; i < count; i++) {
        sum += values[i];
    }
    
    return sum / count;
} 
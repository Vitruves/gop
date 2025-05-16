/**
 * This file contains code that is a good candidate for refactoring.
 * It has inconsistent naming, duplicated code, and other issues.
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// Global variables with inconsistent naming
int MAX_BUFFER_size = 1024;
int min_buffer_SIZE = 128;
char* globalBuffer;
char* Global_Message = "This is a global message";

// Function with old_style_naming
int calculate_sum(int a, int b) {
    return a + b;
}

// Same function with camelCase naming
int calculateSum(int a, int b) {
    return a + b;
}

// Function with PascalCase naming
int CalculateSum(int a, int b) {
    return a + b;
}

// Duplicated code block 1
void process_data_v1(int* data, int size) {
    for (int i = 0; i < size; i++) {
        // Process each element
        if (data[i] < 0) {
            data[i] = 0;
        } else if (data[i] > 100) {
            data[i] = 100;
        }
        
        // Print the processed element
        printf("Processed element %d: %d\n", i, data[i]);
    }
}

// Duplicated code block 2
void process_data_v2(int* data, int size) {
    for (int i = 0; i < size; i++) {
        // Process each element
        if (data[i] < 0) {
            data[i] = 0;
        } else if (data[i] > 100) {
            data[i] = 100;
        }
        
        // Print the processed element
        printf("Processed element %d: %d\n", i, data[i]);
    }
}

// Function with inconsistent parameter naming
void print_array(int* arr, int size) {
    for (int i = 0; i < size; i++) {
        printf("%d ", arr[i]);
    }
    printf("\n");
}

// Similar function with different parameter naming
void display_array(int* data, int length) {
    for (int i = 0; i < length; i++) {
        printf("%d ", data[i]);
    }
    printf("\n");
}

// Function with hardcoded values
void allocate_buffer() {
    globalBuffer = (char*)malloc(1024);
    if (globalBuffer == NULL) {
        printf("Failed to allocate buffer of size 1024\n");
        return;
    }
    
    // Initialize buffer
    memset(globalBuffer, 0, 1024);
    printf("Allocated buffer of size 1024\n");
}

// Another function with the same hardcoded values
void reset_buffer() {
    if (globalBuffer != NULL) {
        memset(globalBuffer, 0, 1024);
        printf("Reset buffer of size 1024\n");
    }
}

// Function with inconsistent error handling
int process_file_v1(const char* filename) {
    FILE* file = fopen(filename, "r");
    if (file == NULL) {
        printf("Error: Could not open file %s\n", filename);
        return -1;
    }
    
    // Process file...
    
    fclose(file);
    return 0;
}

// Similar function with different error handling
int process_file_v2(const char* filename) {
    FILE* file = fopen(filename, "r");
    if (file == NULL) {
        fprintf(stderr, "Failed to open file: %s\n", filename);
        exit(1);
    }
    
    // Process file...
    
    fclose(file);
    return 0;
}

// Main function
int main() {
    // Allocate buffer
    allocate_buffer();
    
    // Create and process data
    int data[10] = {-5, 10, 50, 120, 30, -10, 80, 90, 110, 40};
    
    // Process data using v1
    process_data_v1(data, 10);
    
    // Print array
    print_array(data, 10);
    
    // Reset buffer
    reset_buffer();
    
    // Free buffer
    free(globalBuffer);
    
    return 0;
}

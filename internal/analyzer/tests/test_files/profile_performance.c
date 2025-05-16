/**
 * This file contains code with various performance characteristics
 * for testing the profile tool.
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

// Function with O(n) complexity
void linear_search(int* array, int size, int target) {
    for (int i = 0; i < size; i++) {
        if (array[i] == target) {
            printf("Found target %d at index %d\n", target, i);
            return;
        }
    }
    printf("Target %d not found\n", target);
}

// Function with O(log n) complexity
void binary_search(int* array, int size, int target) {
    int left = 0;
    int right = size - 1;
    
    while (left <= right) {
        int mid = left + (right - left) / 2;
        
        if (array[mid] == target) {
            printf("Found target %d at index %d\n", target, mid);
            return;
        }
        
        if (array[mid] < target) {
            left = mid + 1;
        } else {
            right = mid - 1;
        }
    }
    
    printf("Target %d not found\n", target);
}

// Function with O(n^2) complexity
void bubble_sort(int* array, int size) {
    for (int i = 0; i < size - 1; i++) {
        for (int j = 0; j < size - i - 1; j++) {
            if (array[j] > array[j + 1]) {
                // Swap elements
                int temp = array[j];
                array[j] = array[j + 1];
                array[j + 1] = temp;
            }
        }
    }
}

// Function with O(n log n) complexity
void quick_sort_impl(int* array, int low, int high) {
    if (low < high) {
        // Choose pivot
        int pivot = array[high];
        int i = low - 1;
        
        // Partition
        for (int j = low; j <= high - 1; j++) {
            if (array[j] < pivot) {
                i++;
                // Swap
                int temp = array[i];
                array[i] = array[j];
                array[j] = temp;
            }
        }
        
        // Swap pivot
        int temp = array[i + 1];
        array[i + 1] = array[high];
        array[high] = temp;
        
        int pivot_index = i + 1;
        
        // Recursive calls
        quick_sort_impl(array, low, pivot_index - 1);
        quick_sort_impl(array, pivot_index + 1, high);
    }
}

void quick_sort(int* array, int size) {
    quick_sort_impl(array, 0, size - 1);
}

// Function with memory allocation
void memory_intensive(int size) {
    printf("Allocating memory...\n");
    
    // Allocate a large array
    int** matrix = (int**)malloc(size * sizeof(int*));
    if (matrix == NULL) {
        printf("Memory allocation failed\n");
        return;
    }
    
    for (int i = 0; i < size; i++) {
        matrix[i] = (int*)malloc(size * sizeof(int));
        if (matrix[i] == NULL) {
            printf("Memory allocation failed\n");
            
            // Free previously allocated memory
            for (int j = 0; j < i; j++) {
                free(matrix[j]);
            }
            free(matrix);
            return;
        }
    }
    
    // Initialize matrix
    for (int i = 0; i < size; i++) {
        for (int j = 0; j < size; j++) {
            matrix[i][j] = i * j;
        }
    }
    
    // Use the matrix
    long sum = 0;
    for (int i = 0; i < size; i++) {
        for (int j = 0; j < size; j++) {
            sum += matrix[i][j];
        }
    }
    
    printf("Sum of matrix elements: %ld\n", sum);
    
    // Free memory
    for (int i = 0; i < size; i++) {
        free(matrix[i]);
    }
    free(matrix);
}

// Function with CPU-intensive computation
void cpu_intensive(int iterations) {
    printf("Starting CPU-intensive computation...\n");
    
    double result = 0.0;
    
    // Compute approximation of pi using Leibniz formula
    for (int i = 0; i < iterations; i++) {
        double term = (i % 2 == 0) ? 1.0 : -1.0;
        term /= (2 * i + 1);
        result += term;
    }
    
    result *= 4;
    
    printf("Approximation of pi after %d iterations: %.10f\n", iterations, result);
}

// Function with inefficient string operations
void string_operations(int iterations) {
    printf("Starting string operations...\n");
    
    char* str = (char*)malloc(iterations + 1);
    if (str == NULL) {
        printf("Memory allocation failed\n");
        return;
    }
    
    // Initialize string
    memset(str, 'A', iterations);
    str[iterations] = '\0';
    
    // Inefficient string manipulation
    for (int i = 0; i < iterations / 10; i++) {
        int pos = rand() % iterations;
        str[pos] = 'B' + (i % 26);
    }
    
    // Count occurrences of each character
    int counts[26] = {0};
    for (int i = 0; i < iterations; i++) {
        if (str[i] >= 'A' && str[i] <= 'Z') {
            counts[str[i] - 'A']++;
        }
    }
    
    // Print counts
    for (int i = 0; i < 26; i++) {
        if (counts[i] > 0) {
            printf("%c: %d\n", 'A' + i, counts[i]);
        }
    }
    
    free(str);
}

// Function with recursive calls
int fibonacci(int n) {
    if (n <= 1) {
        return n;
    }
    return fibonacci(n - 1) + fibonacci(n - 2);
}

void recursive_function(int n) {
    printf("Computing Fibonacci(%d)...\n", n);
    clock_t start = clock();
    int result = fibonacci(n);
    clock_t end = clock();
    double time_spent = (double)(end - start) / CLOCKS_PER_SEC;
    printf("Fibonacci(%d) = %d (computed in %.6f seconds)\n", n, result, time_spent);
}

// Main function
int main(int argc, char* argv[]) {
    // Seed random number generator
    srand(time(NULL));
    
    // Parse command line arguments
    int size = 1000;
    int iterations = 1000000;
    
    if (argc > 1) {
        size = atoi(argv[1]);
    }
    
    if (argc > 2) {
        iterations = atoi(argv[2]);
    }
    
    printf("Running performance tests with size=%d, iterations=%d\n", size, iterations);
    
    // Create and initialize array
    int* array = (int*)malloc(size * sizeof(int));
    if (array == NULL) {
        printf("Memory allocation failed\n");
        return 1;
    }
    
    for (int i = 0; i < size; i++) {
        array[i] = rand() % (size * 10);
    }
    
    // Run sorting algorithms
    int* array_copy = (int*)malloc(size * sizeof(int));
    if (array_copy == NULL) {
        printf("Memory allocation failed\n");
        free(array);
        return 1;
    }
    
    // Test bubble sort
    memcpy(array_copy, array, size * sizeof(int));
    clock_t start = clock();
    bubble_sort(array_copy, size);
    clock_t end = clock();
    double time_spent = (double)(end - start) / CLOCKS_PER_SEC;
    printf("Bubble sort completed in %.6f seconds\n", time_spent);
    
    // Test quick sort
    memcpy(array_copy, array, size * sizeof(int));
    start = clock();
    quick_sort(array_copy, size);
    end = clock();
    time_spent = (double)(end - start) / CLOCKS_PER_SEC;
    printf("Quick sort completed in %.6f seconds\n", time_spent);
    
    // Test search algorithms
    int target = array_copy[size / 2]; // Pick a value that exists in the array
    
    start = clock();
    linear_search(array_copy, size, target);
    end = clock();
    time_spent = (double)(end - start) / CLOCKS_PER_SEC;
    printf("Linear search completed in %.6f seconds\n", time_spent);
    
    start = clock();
    binary_search(array_copy, size, target);
    end = clock();
    time_spent = (double)(end - start) / CLOCKS_PER_SEC;
    printf("Binary search completed in %.6f seconds\n", time_spent);
    
    // Test memory-intensive operations
    memory_intensive(size / 10);
    
    // Test CPU-intensive operations
    cpu_intensive(iterations);
    
    // Test string operations
    string_operations(iterations / 100);
    
    // Test recursive function
    recursive_function(30);
    
    // Clean up
    free(array);
    free(array_copy);
    
    return 0;
}

/**
 * Sample C++ header file for testing
 */
#ifndef SAMPLE_CPP_H
#define SAMPLE_CPP_H

#include <vector>

class Calculator {
public:
    Calculator();
    
    int getValue() const;
    void setValue(int value);
    
    int add(int a, int b);
    int multiply(int a, int b);
    
    // This method is declared but not implemented
    int divide(int a, int b);
};

// This function is declared but not implemented
void sortVector(std::vector<int>& data);

// This function is declared but implemented in the cpp file
void processVector(std::vector<int>& data);

#endif /* SAMPLE_CPP_H */

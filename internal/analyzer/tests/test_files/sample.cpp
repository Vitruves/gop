/**
 * Sample C++ file for testing
 */
#include <vector>
#include "./sample_cpp.h"

// TODO: Add proper error handling
class Calculator {
private:
    int m_value;

public:
    Calculator() : m_value(0) {}
    
    int getValue() const {
        return m_value;
    }
    
    void setValue(int value) {
        m_value = value;
    }
    
    // FIXME: This method could be optimized
    int add(int a, int b) {
        m_value = a + b;
        return m_value;
    }
    
    int multiply(int a, int b) {
        m_value = a * b;
        return m_value;
    }
};

// This is a utility function
void processVector(std::vector<int>& data) {
    for (auto& item : data) {
        item *= 2;
    }
}

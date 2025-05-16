#include <iostream>
#include <string>
#include <vector>

// File with Unicode characters and unusual syntax patterns
// to test edge cases for the refactor tool

// Unicode variable names
int 变量1 = 42;  // Chinese characters
double π = 3.14159;  // Greek letter pi
std::string résumé = "CV";  // Accented characters

// Function with Unicode name
void 打印消息(const std::string& 消息) {
    std::cout << "消息: " << 消息 << std::endl;
}

// Class with emoji in comments
class DataProcessor {
public:
    // 🚀 Constructor
    DataProcessor() : initialized_(false) {}
    
    // 🔍 Process data
    void process(const std::vector<int>& data) {
        // 📊 Processing logic
        for (int value : data) {
            processedData_.push_back(value * 2);
        }
        initialized_ = true;
    }
    
    // 📋 Get results
    std::vector<int> getResults() const {
        return processedData_;
    }
    
private:
    bool initialized_;  // ✅ Initialization flag
    std::vector<int> processedData_;  // 📝 Processed data
};

// Unusual syntax patterns
#define STRANGE_MACRO(x) do { \
    if (x > 0) { \
        std::cout << "Positive: " << x << std::endl; \
    } else if (x < 0) { \
        std::cout << "Negative: " << x << std::endl; \
    } else { \
        std::cout << "Zero" << std::endl; \
    } \
} while(0)

// Function with unusual formatting
int
calculate
(
    int a,
    int b
)
{
    return
        a
        +
        b;
}

// Template with complex nesting
template<typename T, template<typename> class Container>
class ComplexTemplate {
public:
    template<typename U>
    struct NestedTemplate {
        U value;
        
        template<typename V>
        V convert(V multiplier) {
            return static_cast<V>(value) * multiplier;
        }
    };
    
    Container<NestedTemplate<T>> data;
};

// Function with mixed tabs and spaces
void mixedIndentation() {
	std::cout << "This line uses tabs" << std::endl;
    std::cout << "This line uses spaces" << std::endl;
	    std::cout << "This line uses both" << std::endl;
}

// String with escape sequences to test refactoring
const char* complexString = "Line 1\n"
                            "Line \"2\" with \"quotes\"\n"
                            "Line \\3\\ with \\backslashes\\\n"
                            "Line 4 with \t tabs and \r returns";

// Main function
int main() {
    // Print Unicode variables
    std::cout << "变量1 = " << 变量1 << std::endl;
    std::cout << "π = " << π << std::endl;
    std::cout << "résumé = " << résumé << std::endl;
    
    // Call Unicode function
    打印消息("Hello, World!");
    
    // Use DataProcessor class
    DataProcessor processor;
    processor.process({1, 2, 3, 4, 5});
    
    // Use unusual syntax
    STRANGE_MACRO(42);
    STRANGE_MACRO(-7);
    STRANGE_MACRO(0);
    
    // Call function with unusual formatting
    int result = calculate(10, 20);
    std::cout << "Result: " << result << std::endl;
    
    // Use mixed indentation function
    mixedIndentation();
    
    // Print complex string
    std::cout << complexString << std::endl;
    
    return 0;
}

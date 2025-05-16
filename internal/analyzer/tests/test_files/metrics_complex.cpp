#include <iostream>
#include <vector>
#include <algorithm>
#include <string>

// Complex class with nested structures and high cyclomatic complexity
class ComplexDataProcessor {
private:
    std::vector<int> data;
    int threshold;
    bool initialized;
    
    // Nested structure
    struct ProcessingOptions {
        bool normalize;
        bool filter;
        int filterThreshold;
        std::string outputFormat;
    };
    
    ProcessingOptions options;

public:
    // Constructor with initialization list
    ComplexDataProcessor(int t) : 
        threshold(t), 
        initialized(false) {
        // Empty constructor body
    }
    
    // Method with high cyclomatic complexity
    void processData(const std::vector<int>& input) {
        // Initialize data
        data = input;
        initialized = true;
        
        // Complex processing with multiple branches
        if (options.normalize) {
            // Find min and max for normalization
            auto minmax = std::minmax_element(data.begin(), data.end());
            int min = *minmax.first;
            int max = *minmax.second;
            
            // Avoid division by zero
            if (max == min) {
                std::cout << "Warning: Cannot normalize, all values are equal" << std::endl;
            } else {
                // Normalize data
                for (auto& val : data) {
                    val = static_cast<int>(100.0 * (val - min) / (max - min));
                }
            }
        }
        
        // Apply filtering if enabled
        if (options.filter) {
            std::vector<int> filtered;
            for (auto val : data) {
                if (val > options.filterThreshold) {
                    filtered.push_back(val);
                } else if (val < -options.filterThreshold) {
                    // Handle negative values differently
                    filtered.push_back(-val);
                } else if (val == 0) {
                    // Special case for zero
                    if (options.filterThreshold == 0) {
                        filtered.push_back(0);
                    }
                }
            }
            data = filtered;
        }
        
        // Apply threshold
        for (size_t i = 0; i < data.size(); ++i) {
            if (data[i] > threshold) {
                data[i] = threshold;
            } else if (data[i] < -threshold) {
                data[i] = -threshold;
            }
        }
    }
    
    // Method with nested loops
    void analyzePatterns() {
        if (!initialized || data.empty()) {
            std::cout << "Error: Data not initialized or empty" << std::endl;
            return;
        }
        
        // Find patterns with nested loops
        for (size_t i = 0; i < data.size(); ++i) {
            int count = 0;
            for (size_t j = 0; j < data.size(); ++j) {
                if (i != j) {
                    // Check for similar values
                    if (std::abs(data[i] - data[j]) < threshold / 10) {
                        count++;
                    }
                    
                    // Look for sequences
                    if (j > 0 && j < data.size() - 1) {
                        if (data[j-1] < data[j] && data[j] < data[j+1]) {
                            // Increasing sequence
                            std::cout << "Increasing sequence at " << j-1 << "-" << j+1 << std::endl;
                        } else if (data[j-1] > data[j] && data[j] > data[j+1]) {
                            // Decreasing sequence
                            std::cout << "Decreasing sequence at " << j-1 << "-" << j+1 << std::endl;
                        }
                    }
                }
            }
            
            if (count > data.size() / 3) {
                std::cout << "Value at " << i << " is similar to many others" << std::endl;
            }
        }
    }
    
    // Setter for options
    void setOptions(bool normalize, bool filter, int filterThreshold, const std::string& format) {
        options.normalize = normalize;
        options.filter = filter;
        options.filterThreshold = filterThreshold;
        options.outputFormat = format;
    }
    
    // Getter for processed data
    const std::vector<int>& getProcessedData() const {
        return data;
    }
};

// Global function with moderate complexity
int findOptimalThreshold(const std::vector<int>& data, int minThreshold, int maxThreshold) {
    int bestThreshold = minThreshold;
    int bestScore = -1;
    
    for (int threshold = minThreshold; threshold <= maxThreshold; threshold++) {
        int score = 0;
        
        // Calculate score for this threshold
        for (auto val : data) {
            if (val <= threshold) {
                score++;
            } else {
                score--;
            }
            
            // Bonus for values close to threshold
            if (std::abs(val - threshold) < 5) {
                score += 2;
            }
        }
        
        // Update best threshold if score is better
        if (score > bestScore) {
            bestScore = score;
            bestThreshold = threshold;
        }
    }
    
    return bestThreshold;
}

// Simple utility function
void printVector(const std::vector<int>& vec) {
    std::cout << "[ ";
    for (auto val : vec) {
        std::cout << val << " ";
    }
    std::cout << "]" << std::endl;
}

// Main function
int main() {
    // Create test data
    std::vector<int> testData = {15, 7, 42, 23, 8, 16, 4, 11, 29};
    
    // Find optimal threshold
    int threshold = findOptimalThreshold(testData, 5, 30);
    std::cout << "Optimal threshold: " << threshold << std::endl;
    
    // Process data
    ComplexDataProcessor processor(threshold);
    processor.setOptions(true, true, 5, "standard");
    processor.processData(testData);
    
    // Print results
    std::cout << "Processed data: ";
    printVector(processor.getProcessedData());
    
    // Analyze patterns
    processor.analyzePatterns();
    
    return 0;
}

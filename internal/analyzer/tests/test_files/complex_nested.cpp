#include <iostream>
#include <vector>
#include <map>
#include <string>
#include <algorithm>
#include <functional>
#include <memory>
#include <stdexcept>

// A file with extremely complex nested structures and high cyclomatic complexity

namespace gop {
namespace testing {
namespace complexity {

// Forward declarations
template<typename T> class ComplexDataStructure;
template<typename K, typename V> class NestedProcessor;

// Enum for processing modes
enum class ProcessingMode {
    SIMPLE,
    NORMAL,
    COMPLEX,
    ADVANCED,
    EXPERT
};

// Struct with nested types
struct ConfigOptions {
    bool enableLogging;
    int maxDepth;
    double threshold;
    std::string outputFormat;
    
    struct ValidationRules {
        bool strictMode;
        int errorTolerance;
        
        enum class Severity {
            LOW,
            MEDIUM,
            HIGH,
            CRITICAL
        };
        
        Severity defaultSeverity;
        std::map<std::string, Severity> customSeverities;
        
        bool validate(const std::string& input, Severity* outSeverity = nullptr) const {
            // Complex validation logic with high cyclomatic complexity
            if (input.empty()) {
                if (outSeverity) *outSeverity = Severity::LOW;
                return strictMode ? false : true;
            }
            
            if (input.length() < 5) {
                if (strictMode) {
                    if (outSeverity) *outSeverity = Severity::MEDIUM;
                    return false;
                } else {
                    if (errorTolerance > 2) {
                        if (outSeverity) *outSeverity = Severity::LOW;
                        return true;
                    } else {
                        if (outSeverity) *outSeverity = Severity::MEDIUM;
                        return false;
                    }
                }
            }
            
            // Check for custom severities
            for (const auto& pair : customSeverities) {
                if (input.find(pair.first) != std::string::npos) {
                    if (outSeverity) *outSeverity = pair.second;
                    
                    switch (pair.second) {
                        case Severity::LOW:
                            return true;
                        case Severity::MEDIUM:
                            return errorTolerance > 1;
                        case Severity::HIGH:
                            return errorTolerance > 2;
                        case Severity::CRITICAL:
                            return false;
                        default:
                            return false;
                    }
                }
            }
            
            // Default case
            if (outSeverity) *outSeverity = defaultSeverity;
            return true;
        }
    };
    
    ValidationRules validationRules;
    
    // Nested function object
    struct Processor {
        std::function<void(const std::string&)> preProcess;
        std::function<std::string(const std::string&)> process;
        std::function<void(const std::string&)> postProcess;
        
        std::string operator()(const std::string& input) const {
            if (preProcess) preProcess(input);
            
            std::string result;
            if (process) {
                result = process(input);
            } else {
                result = input;
            }
            
            if (postProcess) postProcess(result);
            return result;
        }
    };
    
    Processor processor;
};

// Class with extremely high cyclomatic complexity
class ComplexAlgorithm {
public:
    ComplexAlgorithm(int maxIterations, double convergenceThreshold)
        : maxIterations_(maxIterations), 
          convergenceThreshold_(convergenceThreshold),
          initialized_(false) {}
    
    bool initialize(const std::vector<double>& initialData) {
        if (initialData.empty()) {
            lastError_ = "Empty initial data";
            return false;
        }
        
        data_ = initialData;
        initialized_ = true;
        return true;
    }
    
    // Method with extremely high cyclomatic complexity
    std::vector<double> process(ProcessingMode mode, bool normalize = false) {
        if (!initialized_) {
            throw std::runtime_error("Algorithm not initialized");
        }
        
        std::vector<double> result = data_;
        double sum = 0.0;
        double min = std::numeric_limits<double>::max();
        double max = std::numeric_limits<double>::lowest();
        
        // Pre-processing
        for (size_t i = 0; i < result.size(); ++i) {
            // Handle different modes
            switch (mode) {
                case ProcessingMode::SIMPLE:
                    // Simple processing - just square the values
                    result[i] = result[i] * result[i];
                    break;
                    
                case ProcessingMode::NORMAL:
                    // Normal processing - apply some transformations
                    if (result[i] < 0) {
                        result[i] = -result[i] * 2;
                    } else if (result[i] > 100) {
                        result[i] = 100 + std::log(result[i] - 99);
                    } else {
                        result[i] = result[i] * 1.5;
                    }
                    break;
                    
                case ProcessingMode::COMPLEX:
                    // Complex processing with multiple branches
                    if (i % 3 == 0) {
                        // Every third element gets special treatment
                        if (result[i] < 0) {
                            result[i] = 0;
                        } else if (result[i] < 50) {
                            result[i] = result[i] * 2;
                        } else if (result[i] < 100) {
                            result[i] = 100;
                        } else {
                            result[i] = 100 + (result[i] - 100) / 2;
                        }
                    } else if (i % 3 == 1) {
                        // Different treatment for remainder 1
                        if (result[i] >= 0 && result[i] <= 100) {
                            result[i] = result[i] / 2;
                        } else if (result[i] > 100) {
                            result[i] = 50 + result[i] / 10;
                        } else {
                            result[i] = 0;
                        }
                    } else {
                        // Different treatment for remainder 2
                        if (result[i] < 0) {
                            result[i] = -std::sqrt(-result[i]);
                        } else {
                            result[i] = std::sqrt(result[i]);
                        }
                    }
                    break;
                    
                case ProcessingMode::ADVANCED:
                    // Advanced processing with nested conditions
                    if (i > 0 && i < result.size() - 1) {
                        // Not first or last element
                        double prev = result[i-1];
                        double curr = result[i];
                        double next = result[i+1];
                        
                        if (prev < curr && curr < next) {
                            // Increasing sequence
                            if (curr - prev < next - curr) {
                                // Acceleration
                                result[i] = curr * 1.5;
                            } else {
                                // Deceleration
                                result[i] = curr * 1.2;
                            }
                        } else if (prev > curr && curr > next) {
                            // Decreasing sequence
                            if (prev - curr < curr - next) {
                                // Acceleration
                                result[i] = curr * 0.5;
                            } else {
                                // Deceleration
                                result[i] = curr * 0.8;
                            }
                        } else if (prev < curr && curr > next) {
                            // Peak
                            result[i] = (prev + next) / 2;
                        } else if (prev > curr && curr < next) {
                            // Valley
                            result[i] = curr * 2;
                        } else {
                            // Equal values
                            result[i] = curr;
                        }
                    } else if (i == 0) {
                        // First element
                        if (result.size() > 1) {
                            if (result[i] < result[i+1]) {
                                result[i] = 0;
                            } else {
                                result[i] = result[i] * 2;
                            }
                        }
                    } else {
                        // Last element
                        if (result[i] < result[i-1]) {
                            result[i] = 0;
                        } else {
                            result[i] = result[i] * 2;
                        }
                    }
                    break;
                    
                case ProcessingMode::EXPERT:
                    // Expert mode with extremely complex logic
                    {
                        double factor = 1.0;
                        
                        // Determine factor based on position
                        if (i == 0) {
                            factor = 2.0;
                        } else if (i == result.size() - 1) {
                            factor = 0.5;
                        } else {
                            factor = static_cast<double>(i) / result.size();
                        }
                        
                        // Apply different transformations based on value ranges
                        if (result[i] < -100) {
                            result[i] = -100;
                        } else if (result[i] < -50) {
                            result[i] = result[i] * factor;
                        } else if (result[i] < 0) {
                            if (i % 2 == 0) {
                                result[i] = -result[i];
                            } else {
                                result[i] = result[i] * 2;
                            }
                        } else if (result[i] < 50) {
                            if (i % 3 == 0) {
                                result[i] = result[i] * 3;
                            } else if (i % 3 == 1) {
                                result[i] = result[i] * 2;
                            } else {
                                result[i] = result[i] * 1.5;
                            }
                        } else if (result[i] < 100) {
                            if (i % 4 == 0) {
                                result[i] = 100;
                            } else if (i % 4 == 1) {
                                result[i] = 75;
                            } else if (i % 4 == 2) {
                                result[i] = 50;
                            } else {
                                result[i] = 25;
                            }
                        } else {
                            if (i % 2 == 0) {
                                result[i] = 100 + std::log(result[i] - 99);
                            } else {
                                result[i] = 100;
                            }
                        }
                    }
                    break;
            }
            
            // Update statistics
            sum += result[i];
            min = std::min(min, result[i]);
            max = std::max(max, result[i]);
        }
        
        // Post-processing
        if (normalize && max > min) {
            // Normalize to [0, 1] range
            for (size_t i = 0; i < result.size(); ++i) {
                result[i] = (result[i] - min) / (max - min);
            }
        }
        
        // Check convergence
        double mean = sum / result.size();
        double variance = 0.0;
        
        for (double val : result) {
            variance += (val - mean) * (val - mean);
        }
        
        variance /= result.size();
        
        if (variance < convergenceThreshold_) {
            std::cout << "Algorithm converged with variance: " << variance << std::endl;
        } else {
            std::cout << "Algorithm did not converge. Variance: " << variance << std::endl;
        }
        
        return result;
    }
    
    // Method with recursive complexity
    double recursiveProcess(double value, int depth) {
        if (depth <= 0) {
            return value;
        }
        
        // Different processing based on value ranges and depth
        if (value < 0) {
            if (depth % 2 == 0) {
                return recursiveProcess(-value / 2, depth - 1);
            } else {
                return recursiveProcess(value * 2, depth - 1);
            }
        } else if (value < 1) {
            if (depth % 3 == 0) {
                return recursiveProcess(value + 0.5, depth - 1);
            } else if (depth % 3 == 1) {
                return recursiveProcess(value * 3, depth - 1);
            } else {
                return recursiveProcess(std::sqrt(value), depth - 1);
            }
        } else if (value < 10) {
            if (depth > 5) {
                return recursiveProcess(value / 2, depth - 2);
            } else {
                return recursiveProcess(value * 1.5, depth - 1);
            }
        } else {
            if (depth % 2 == 0) {
                return recursiveProcess(std::log(value), depth - 1);
            } else {
                return recursiveProcess(std::sqrt(value), depth - 1);
            }
        }
    }
    
private:
    int maxIterations_;
    double convergenceThreshold_;
    bool initialized_;
    std::vector<double> data_;
    std::string lastError_;
};

// Template class with complex nested structure
template<typename T>
class ComplexDataStructure {
public:
    // Nested node structure
    struct Node {
        T value;
        std::vector<std::shared_ptr<Node>> children;
        std::weak_ptr<Node> parent;
        
        // Node processing with high complexity
        T process(int depth, bool recursive = true) {
            if (depth <= 0 || children.empty()) {
                return value;
            }
            
            T result = value;
            
            if (recursive) {
                // Recursive processing
                for (const auto& child : children) {
                    if (child) {
                        result = combine(result, child->process(depth - 1, recursive));
                    }
                }
            } else {
                // Non-recursive processing
                std::vector<std::shared_ptr<Node>> nodes = children;
                int currentDepth = 1;
                
                while (currentDepth < depth && !nodes.empty()) {
                    std::vector<std::shared_ptr<Node>> nextLevel;
                    
                    for (const auto& node : nodes) {
                        if (node) {
                            result = combine(result, node->value);
                            
                            for (const auto& child : node->children) {
                                if (child) {
                                    nextLevel.push_back(child);
                                }
                            }
                        }
                    }
                    
                    nodes = nextLevel;
                    currentDepth++;
                }
            }
            
            return result;
        }
        
        // Combine values with different strategies based on type
        T combine(const T& a, const T& b) {
            if constexpr (std::is_arithmetic<T>::value) {
                return a + b;
            } else if constexpr (std::is_same<T, std::string>::value) {
                return a + b;
            } else {
                // Default case, try to use operator+
                return a + b;
            }
        }
    };
    
    // Constructor
    ComplexDataStructure() : root_(std::make_shared<Node>()) {}
    
    // Add a value to the structure
    void add(const T& value, const std::vector<int>& path) {
        if (path.empty()) {
            root_->value = value;
            return;
        }
        
        std::shared_ptr<Node> current = root_;
        
        for (size_t i = 0; i < path.size(); ++i) {
            int index = path[i];
            
            // Ensure the path exists
            while (current->children.size() <= static_cast<size_t>(index)) {
                auto newNode = std::make_shared<Node>();
                newNode->parent = current;
                current->children.push_back(newNode);
            }
            
            current = current->children[index];
        }
        
        current->value = value;
    }
    
    // Process the entire structure
    T process(int maxDepth, bool recursive = true) {
        return root_->process(maxDepth, recursive);
    }
    
private:
    std::shared_ptr<Node> root_;
};

// Function with complex template metaprogramming
template<typename T, typename... Args>
auto complexProcess(T&& first, Args&&... args) {
    if constexpr (sizeof...(args) == 0) {
        // Base case: only one argument
        if constexpr (std::is_arithmetic<std::decay_t<T>>::value) {
            return first * 2;
        } else if constexpr (std::is_same<std::decay_t<T>, std::string>::value) {
            return first + first;
        } else {
            return first;
        }
    } else {
        // Recursive case: multiple arguments
        auto restResult = complexProcess(std::forward<Args>(args)...);
        
        if constexpr (std::is_same<decltype(first + restResult), decltype(first)>::value) {
            return first + restResult;
        } else if constexpr (std::is_arithmetic<std::decay_t<T>>::value && 
                            std::is_arithmetic<std::decay_t<decltype(restResult)>>::value) {
            return first * restResult;
        } else {
            // Default case: try to convert and combine
            return static_cast<std::common_type_t<std::decay_t<T>, std::decay_t<decltype(restResult)>>>(first) + 
                   static_cast<std::common_type_t<std::decay_t<T>, std::decay_t<decltype(restResult)>>>(restResult);
        }
    }
}

// Main function with complex control flow
int main() {
    try {
        // Create and use complex algorithm
        ComplexAlgorithm algorithm(100, 0.001);
        
        std::vector<double> data = {-50, -25, 0, 25, 50, 75, 100, 125, 150};
        
        if (!algorithm.initialize(data)) {
            std::cerr << "Failed to initialize algorithm" << std::endl;
            return 1;
        }
        
        // Process with different modes
        std::cout << "Simple processing:" << std::endl;
        auto simpleResult = algorithm.process(ProcessingMode::SIMPLE);
        for (double val : simpleResult) {
            std::cout << val << " ";
        }
        std::cout << std::endl;
        
        std::cout << "Complex processing:" << std::endl;
        auto complexResult = algorithm.process(ProcessingMode::COMPLEX, true);
        for (double val : complexResult) {
            std::cout << val << " ";
        }
        std::cout << std::endl;
        
        // Test recursive processing
        std::cout << "Recursive processing:" << std::endl;
        for (double val : data) {
            std::cout << "Original: " << val << ", Processed: " 
                      << algorithm.recursiveProcess(val, 5) << std::endl;
        }
        
        // Test complex data structure
        ComplexDataStructure<int> dataStructure;
        dataStructure.add(10, {});
        dataStructure.add(20, {0});
        dataStructure.add(30, {1});
        dataStructure.add(40, {0, 0});
        dataStructure.add(50, {0, 1});
        dataStructure.add(60, {1, 0});
        
        std::cout << "Data structure processing: " << dataStructure.process(3) << std::endl;
        
        // Test complex template function
        std::cout << "Complex process results:" << std::endl;
        std::cout << complexProcess(5) << std::endl;
        std::cout << complexProcess(std::string("Hello")) << std::endl;
        std::cout << complexProcess(1, 2, 3, 4, 5) << std::endl;
        std::cout << complexProcess(1.5, 2.5, 3.5) << std::endl;
        std::cout << complexProcess(std::string("Hello"), std::string(" World")) << std::endl;
        
        return 0;
    } catch (const std::exception& e) {
        std::cerr << "Exception: " << e.what() << std::endl;
        return 1;
    } catch (...) {
        std::cerr << "Unknown exception" << std::endl;
        return 1;
    }
}

} // namespace complexity
} // namespace testing
} // namespace gop

/**
 * @file docs_well_documented.h
 * @brief A header file with extensive documentation for testing the docs tool
 * @author GOP Team
 * @date 2025-05-16
 */

#ifndef DOCS_WELL_DOCUMENTED_H
#define DOCS_WELL_DOCUMENTED_H

#include <string>
#include <vector>
#include <functional>

namespace gop {
namespace docs {

/**
 * @brief Error codes used throughout the library
 * 
 * This enumeration contains all possible error codes that can be
 * returned by functions in this library.
 */
enum class ErrorCode {
    SUCCESS = 0,       /**< Operation completed successfully */
    INVALID_INPUT,     /**< Input parameters were invalid */
    FILE_NOT_FOUND,    /**< Requested file could not be found */
    PERMISSION_DENIED, /**< Permission denied when accessing resource */
    TIMEOUT,           /**< Operation timed out */
    UNKNOWN_ERROR      /**< An unknown error occurred */
};

/**
 * @brief Configuration options for the parser
 * 
 * This struct contains all configuration options that can be
 * passed to the Parser class to customize its behavior.
 */
struct ParserOptions {
    bool extractComments;       /**< Whether to extract comments */
    bool includePrivate;        /**< Whether to include private members */
    bool recursiveMode;         /**< Whether to process files recursively */
    int maxDepth;               /**< Maximum recursion depth (if recursiveMode is true) */
    std::vector<std::string> excludePatterns; /**< Patterns to exclude */
};

/**
 * @brief Default parser options with reasonable defaults
 * 
 * @return ParserOptions with default values set
 */
ParserOptions getDefaultOptions();

/**
 * @brief A class for parsing source code and extracting documentation
 * 
 * This class provides functionality to parse source code files
 * and extract documentation comments for classes, functions, etc.
 */
class Parser {
public:
    /**
     * @brief Constructor with default options
     * 
     * Creates a new Parser instance with default options.
     */
    Parser();
    
    /**
     * @brief Constructor with custom options
     * 
     * Creates a new Parser instance with the specified options.
     * 
     * @param options The options to use for this parser
     */
    explicit Parser(const ParserOptions& options);
    
    /**
     * @brief Destructor
     * 
     * Cleans up any resources used by the Parser.
     */
    ~Parser();
    
    /**
     * @brief Parse a single file
     * 
     * Parses the specified file and extracts documentation.
     * 
     * @param filePath Path to the file to parse
     * @return ErrorCode indicating success or failure
     * @throws std::runtime_error if the file cannot be read
     */
    ErrorCode parseFile(const std::string& filePath);
    
    /**
     * @brief Parse multiple files
     * 
     * Parses all the specified files and extracts documentation.
     * 
     * @param filePaths Paths to the files to parse
     * @return Number of files successfully parsed
     * @see parseFile
     */
    int parseFiles(const std::vector<std::string>& filePaths);
    
    /**
     * @brief Parse a directory
     * 
     * Parses all files in the specified directory that match the
     * configured patterns.
     * 
     * @param dirPath Path to the directory to parse
     * @param recursive Whether to recursively parse subdirectories
     * @return Number of files successfully parsed
     * @example
     * Parser parser;
     * int count = parser.parseDirectory("/path/to/source", true);
     * std::cout << "Parsed " << count << " files" << std::endl;
     */
    int parseDirectory(const std::string& dirPath, bool recursive = false);
    
    /**
     * @brief Get the extracted documentation
     * 
     * Returns the documentation extracted from the parsed files.
     * 
     * @return A string containing the extracted documentation
     */
    std::string getDocumentation() const;
    
    /**
     * @brief Export documentation to a file
     * 
     * Exports the extracted documentation to the specified file.
     * 
     * @param filePath Path to the output file
     * @param format Format of the output (md, html, txt)
     * @return ErrorCode indicating success or failure
     */
    ErrorCode exportDocumentation(const std::string& filePath, const std::string& format = "md");
    
    /**
     * @brief Register a custom documentation formatter
     * 
     * Registers a custom function to format documentation for a specific output format.
     * 
     * @param format The format identifier (e.g., "custom")
     * @param formatter Function that takes documentation string and returns formatted string
     * @return true if the formatter was registered successfully, false otherwise
     */
    bool registerFormatter(const std::string& format, 
                          std::function<std::string(const std::string&)> formatter);
    
private:
    ParserOptions options_; /**< Configuration options for this parser */
    std::string documentation_; /**< Extracted documentation */
    std::vector<std::string> parsedFiles_; /**< List of successfully parsed files */
    
    /**
     * @brief Internal method to process a file
     * 
     * @param filePath Path to the file to process
     * @return true if processing was successful, false otherwise
     */
    bool processFile_(const std::string& filePath);
    
    /**
     * @brief Internal method to extract comments from a file
     * 
     * @param content Content of the file
     * @return Extracted comments
     */
    std::string extractComments_(const std::string& content);
};

/**
 * @brief Utility function to check if a file has documentation
 * 
 * Checks if the specified file contains documentation comments.
 * 
 * @param filePath Path to the file to check
 * @return true if the file has documentation, false otherwise
 */
bool hasDocumentation(const std::string& filePath);

/**
 * @brief Utility function to count documentation lines
 * 
 * Counts the number of documentation comment lines in the specified file.
 * 
 * @param filePath Path to the file to check
 * @return Number of documentation lines, or -1 if the file cannot be read
 */
int countDocumentationLines(const std::string& filePath);

} // namespace docs
} // namespace gop

#endif // DOCS_WELL_DOCUMENTED_H

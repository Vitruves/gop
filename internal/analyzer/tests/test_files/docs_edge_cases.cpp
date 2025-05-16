/**
 * @file docs_edge_cases.cpp
 * This file contains edge cases for the documentation generator
 */

// Incomplete documentation comment
/**
 * This comment is missing a closing delimiter

// Documentation comment with unusual tags
/**
 * @CUSTOM_TAG This is a custom tag that might not be recognized
 * @param
 * @return with no description
 * @throws
 */
void functionWithUnusualTags() {
    // Function body
}

// Nested documentation comments
/**
 * Outer documentation comment
 * /**
 *  * Nested documentation comment
 *  */
 */
void functionWithNestedComments() {
    // Function body
}

// Documentation comment with code examples
/**
 * Function that demonstrates code examples in documentation
 * 
 * @example
 * ```cpp
 * int result = calculateSum(5, 10);
 * assert(result == 15);
 * ```
 * 
 * @example
 * ```
 * // This example has no language specified
 * calculateSum(-1, 1); // Should return 0
 * ```
 */
int calculateSum(int a, int b) {
    return a + b;
}

// Documentation for overloaded functions
/**
 * @overload
 */
void overloadedFunction(int x);

/**
 * @overload
 */
void overloadedFunction(double x);

/**
 * @overload
 */
void overloadedFunction(const char* x);

// Implementation of overloaded functions
void overloadedFunction(int x) {}
void overloadedFunction(double x) {}
void overloadedFunction(const char* x) {}

// Documentation with Unicode characters
/**
 * Function with Unicode characters in documentation: π, é, ñ, 你好, 😊
 * 
 * @param π The value of pi
 * @param résumé The résumé parameter
 * @return 你好 (Hello)
 */
double unicodeFunction(double π, const char* résumé) {
    return π * 2.0;
}

// Documentation with HTML-like tags
/**
 * Function with <b>HTML-like</b> tags in the documentation.
 * 
 * <ul>
 *   <li>Item 1</li>
 *   <li>Item 2</li>
 * </ul>
 * 
 * <pre>
 * This is preformatted text
 * that should be preserved
 * </pre>
 */
void htmlTagsFunction() {
    // Function body
}

// Documentation with markdown
/**
 * Function with **Markdown** formatting in the documentation.
 * 
 * # Heading 1
 * ## Heading 2
 * 
 * - List item 1
 * - List item 2
 * 
 * ```cpp
 * int x = 42;
 * ```
 * 
 * [Link text](https://example.com)
 */
void markdownFunction() {
    // Function body
}

// Empty documentation comment
/**
 */
void emptyDocumentation() {
    // Function body
}

// Documentation with extremely long lines
/**
 * This documentation comment contains an extremely long line that might cause issues with parsing or formatting. The line is intentionally very long to test how the documentation generator handles it. It continues for a while to ensure it exceeds any reasonable line length limits that might be in place. This is to simulate real-world documentation where developers might not always follow best practices for line length. The documentation generator should be able to handle this gracefully without crashing or producing invalid output.
 */
void longLineFunction() {
    // Function body
}

// Documentation with special characters
/**
 * Function with special characters: \, ", \n, \t, %, $, #, @, *, &, ^, !, ~
 */
void specialCharactersFunction() {
    // Function body
}
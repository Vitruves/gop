/**
 * This file contains edge cases for metrics analysis
 * Including:
 * - Empty functions
 * - Nested functions
 * - High complexity functions
 * - Unusual comment styles
 * - Macros that look like functions
 */

// Empty function
void empty_function() {
    // This function is empty
}

// Function with high cyclomatic complexity
int high_complexity(int a, int b, int c, int d, int e) {
    int result = 0;
    
    // Multiple conditionals to increase complexity
    if (a > 0) {
        if (b > 0) {
            result += a * b;
        } else if (c > 0) {
            result += a * c;
        } else {
            result += a;
        }
    } else if (b > 0) {
        if (c > 0) {
            result += b * c;
        } else if (d > 0) {
            result += b * d;
        } else {
            result += b;
        }
    } else if (c > 0) {
        if (d > 0) {
            result += c * d;
        } else if (e > 0) {
            result += c * e;
        } else {
            result += c;
        }
    } else if (d > 0) {
        result += d;
    } else if (e > 0) {
        result += e;
    }
    
    // Switch statement to add complexity
    switch (a) {
        case 1:
            result *= 2;
            break;
        case 2:
            result *= 3;
            break;
        case 3:
            result *= 4;
            break;
        case 4:
            result *= 5;
            break;
        default:
            result *= 1;
            break;
    }
    
    // Loops to add complexity
    for (int i = 0; i < a; i++) {
        if (i % 2 == 0) {
            result += i;
        } else {
            result -= i;
        }
    }
    
    return result;
}

// Unusual comment styles
/*
    Multi-line comment
    with multiple lines
    and indentation
*/

/********************************************
 * Block comment with stars
 ********************************************/

///////////////////////////////////////
// Comment with slashes
///////////////////////////////////////

// Nested struct definitions
struct OuterStruct {
    int x;
    int y;
    
    struct InnerStruct {
        int a;
        int b;
        
        struct DeepStruct {
            int value;
        } deep;
    } inner;
};

// Macro that looks like a function
#define FUNCTION_LIKE_MACRO(x, y) ((x) * (y))

// Actual function that uses the macro
int use_macro(int a, int b) {
    return FUNCTION_LIKE_MACRO(a, b);
}

// Function with a long name to test name handling
int this_is_a_very_long_function_name_that_might_cause_issues_with_formatting_or_display_in_some_tools(int x) {
    return x * 2;
}

// One-liner function
int one_liner(int x) { return x * 3; }

// Function with unusual whitespace
int    unusual    whitespace    (   int    x   ,   int    y   )   {
    return    x    +    y   ;
}

// Main function
int main() {
    int a = 5, b = 10;
    int result = high_complexity(a, b, 15, 20, 25);
    result += use_macro(a, b);
    result += one_liner(a);
    result += unusual whitespace(a, b);
    return result;
}

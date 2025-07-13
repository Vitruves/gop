package concatenate

import (
	"path/filepath"
	"regexp"
	"strings"
)

type CppProcessor struct{}

func (cpp *CppProcessor) GetExtensions() []string {
	return []string{".cpp", ".cxx", ".cc", ".hpp", ".hxx", ".hh", ".h++", ".c++"}
}

func (cpp *CppProcessor) IsTestFile(path string) bool {
	filename := filepath.Base(path)
	
	testPatterns := []string{
		"test_*.cpp", "*_test.cpp", "test*.cpp",
		"test_*.cxx", "*_test.cxx", "test*.cxx", 
		"test_*.cc", "*_test.cc", "test*.cc",
		"test_*.hpp", "*_test.hpp", "test*.hpp",
		"test_*.hxx", "*_test.hxx", "test*.hxx",
		"test_*.hh", "*_test.hh", "test*.hh",
	}
	
	for _, pattern := range testPatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}
	}
	
	testDirs := []string{"tests", "test", "unit_tests", "gtest"}
	for _, testDir := range testDirs {
		if strings.Contains(path, testDir) {
			return true
		}
	}
	
	return false
}

func (cpp *CppProcessor) RemoveComments(content string) string {
	singleLineRegex := regexp.MustCompile(`//.*$`)
	lines := strings.Split(content, "\n")
	var result []string
	
	for _, line := range lines {
		processed := singleLineRegex.ReplaceAllString(line, "")
		result = append(result, processed)
	}
	
	content = strings.Join(result, "\n")
	
	multiLineRegex := regexp.MustCompile(`(?s)/\*.*?\*/`)
	content = multiLineRegex.ReplaceAllString(content, "")
	
	return content
}

func (cpp *CppProcessor) RemoveTestCode(content string) string {
	gtestTestRegex := regexp.MustCompile(`(?s)TEST\s*\([^)]*\)\s*\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}`)
	content = gtestTestRegex.ReplaceAllString(content, "")
	
	gtestTestFRegex := regexp.MustCompile(`(?s)TEST_F\s*\([^)]*\)\s*\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}`)
	content = gtestTestFRegex.ReplaceAllString(content, "")
	
	gtestTestPRegex := regexp.MustCompile(`(?s)TEST_P\s*\([^)]*\)\s*\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}`)
	content = gtestTestPRegex.ReplaceAllString(content, "")
	
	catchTestRegex := regexp.MustCompile(`(?s)TEST_CASE\s*\([^)]*\)\s*\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}`)
	content = catchTestRegex.ReplaceAllString(content, "")
	
	testFixtureRegex := regexp.MustCompile(`(?s)class\s+\w+\s*:\s*public\s+::testing::Test\s*\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}`)
	content = testFixtureRegex.ReplaceAllString(content, "")
	
	gtestIncludeRegex := regexp.MustCompile(`(?m)^[ \t]*#include\s+<gtest/gtest\.h>.*\n`)
	content = gtestIncludeRegex.ReplaceAllString(content, "")
	
	catchIncludeRegex := regexp.MustCompile(`(?m)^[ \t]*#include\s+<catch2?/catch\.hpp>.*\n`)
	content = catchIncludeRegex.ReplaceAllString(content, "")
	
	boostTestIncludeRegex := regexp.MustCompile(`(?m)^[ \t]*#include\s+<boost/test/.*\.hpp>.*\n`)
	content = boostTestIncludeRegex.ReplaceAllString(content, "")
	
	testMainRegex := regexp.MustCompile(`(?s)int\s+main\s*\([^)]*\)\s*\{[^{}]*testing::InitGoogleTest[^{}]*\}`)
	content = testMainRegex.ReplaceAllString(content, "")
	
	expectAssertRegex := regexp.MustCompile(`(?m)^[ \t]*(EXPECT_|ASSERT_)\w+\s*\(.*\)\s*;.*\n`)
	content = expectAssertRegex.ReplaceAllString(content, "")
	
	requireRegex := regexp.MustCompile(`(?m)^[ \t]*REQUIRE\s*\(.*\)\s*;.*\n`)
	content = requireRegex.ReplaceAllString(content, "")
	
	return content
}

func (cpp *CppProcessor) SupportsSpecialFiles() map[string]bool {
	return map[string]bool{
		"Makefile":       true,
		"CMakeLists.txt": true,
		"configure.ac":   true,
		"configure.in":   true,
		"config.h.in":    true,
		"meson.build":    true,
		"conanfile.txt":  true,
		"conanfile.py":   true,
		"vcpkg.json":     true,
	}
}

func (cpp *CppProcessor) IsHeaderFile(path string) bool {
	ext := filepath.Ext(path)
	headerExts := []string{".hpp", ".hxx", ".hh", ".h++"}
	for _, headerExt := range headerExts {
		if ext == headerExt {
			return true
		}
	}
	return false
}
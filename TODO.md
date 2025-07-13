gop <command> [options] 

gop is a tool to provide utilities to help code with AI

[global options]
-l/--language <python,rust,go,c,cpp>
-i/--include <dir or file> (only concat the dir or files, can be used multiple file for multiple dir or files, also support wildcard like for example code-*)
-e/--exclude <dir or file> (inverse of include)
-R/--recursive (concat all code going down all dirs)
-d/--depth <N> (depth of recursive)
-j/--jobs <N> (number of cpu core to use, default is all)
-v/--verbose
-h/--help

gop concatenate [options] (concatenate all code matching language extension in current dir)
--remove-tests (especially for rust when tests are in config test in same file as source, but also exclude classic tests path/patterns in other languages)
--remove-comments
--add-line-numbers (per script exact line numbering at start of each line)
--add-headers (clearly separate each script with name and path)
-o/--output <.txt> (if not enabled, output to console)


gop function-registry [options] (create a registry of all functions in codebase with all info like usage, availability (private,public) etc. - ensure to perform exact duplicate filtering. ensure to output as concise as possible while being precise)
-o/--output <.md or .txt or .yaml>
--by-script (clearly group and separate function by script name, in other word create per script function registry and concat all registries to output)
code specific c/cpp : --only-header-files
--add-relations (tell if function is called by other files)
--only-dead-code


gop placeholders [options] (search and highlight placeholders generated )


gop stats [options] (generate codebase stats : number of scripts, lines, functions, comments, etc.)
-o/--output <.txt>


organize structure so it can be installed with go install {to determine}/gop@latest

create tests for all functions and options
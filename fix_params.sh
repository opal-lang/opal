#!/bin/bash

# Fix parameter struct literals in test files
file="runtime/execution/plan/generator_test.go"

# Replace {Name: "", with {ParamName: "", ParamValue: 
sed -i 's/{Name: "", /{ParamName: "", ParamValue: /g' "$file"

# Replace {Name: "param", with {ParamName: "param", ParamValue: 
sed -i 's/{Name: "\([^"]*\)", /{ParamName: "\1", ParamValue: /g' "$file"

echo "Fixed parameter literals in $file"
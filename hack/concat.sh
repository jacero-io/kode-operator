#!/bin/bash

# This script concatenates files from your project directory based on include and exclude patterns.
#
# Usage: ./script.sh [options]
#
# Options:
#   -i: Specify include patterns (comma-separated)
#   -e: Specify exclude patterns (comma-separated)
#
# Example:
#   ./script.sh -i "*.go,controllers/*" -e "*_test.go,*_mock.go"
#
# This example will include all .go files and everything in the controllers directory,
# but exclude test files and mock files.

# Output file
output_file="concatenated_output.txt"

# Create or empty the output file
> "$output_file"

# Exclude patterns
exclude_patterns=( 'zz_generated.deepcopy.go' 'entrypoint_integration_test.go' )

# Include patterns (initially empty)
include_patterns=( 'v1alpha2/*' 'cleanup/*' 'controllers/kode/*' 'test/integration/controller/*' )

# Function to process files
process_files() {
  local dir=$1
  local find_command="find \"$dir\" -type f"
  
  # Apply exclude patterns
  for pattern in "${exclude_patterns[@]}"; do
    find_command+=" ! -name \"$pattern\""
  done
  
  # Apply include patterns if any
  if [ ${#include_patterns[@]} -gt 0 ]; then
    find_command+=" \( "
    for pattern in "${include_patterns[@]}"; do
      if [[ $pattern == *"/"* ]]; then
        # If pattern contains a slash, treat it as a path
        find_command+=" -path \"*/$pattern\" -o"
      else
        # Otherwise, treat it as a filename pattern
        find_command+=" -name \"$pattern\" -o"
      fi
    done
    find_command=${find_command% -o} # Remove the last -o
    find_command+=" \)"
  fi

  eval "$find_command"
}

# Parse command-line options
while getopts "i:e:" opt; do
  case $opt in
    i)
      IFS=',' read -ra ADDR <<< "$OPTARG"
      for i in "${ADDR[@]}"; do
        include_patterns+=("$i")
      done
      ;;
    e)
      IFS=',' read -ra ADDR <<< "$OPTARG"
      for i in "${ADDR[@]}"; do
        exclude_patterns+=("$i")
      done
      ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      exit 1
      ;;
    :)
      echo "Option -$OPTARG requires an argument." >&2
      exit 1
      ;;
  esac
done

# Process the entire project directory
process_files "../" | while read -r file; do
  echo "#####################################" >> "$output_file"
  echo "// $file" >> "$output_file"
  echo "" >> "$output_file"  # Add a newline for separation
  
  # Remove multiline comments and output the result
  sed '/\/\*/,/\*\//d' "$file" >> "$output_file"
  
  echo "" >> "$output_file"  # Add a newline for separation
done

echo "Concatenation complete. Output written to $output_file"

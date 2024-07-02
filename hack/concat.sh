#!/bin/bash

# Output file
output_file="concatenated_output.txt"

# Create or empty the output file
> "$output_file"

# Exclude patterns
exclude_patterns=('*deepcopy.go' '*_test.go' 'entrypoint*' '*validator*')

# Function to process files
process_files() {
  local dir=$1
  local find_command="find \"../$dir\" -type f"
  for pattern in "${exclude_patterns[@]}"; do
    find_command+=" ! -name \"$pattern\""
  done

  eval "$find_command" | while read -r file; do
    echo "#################################################################################" >> "$output_file"
    txtFile="${file#../}"
    echo "// $txtFile" >> "$output_file"
    echo "" >> "$output_file"  # Add a newline for separation
    cat "$file" >> "$output_file"
    echo "" >> "$output_file"  # Add a newline for separation
  done
}

# Process directories
process_files "api"
process_files "internal"

echo "Concatenation complete. Output written to $output_file"

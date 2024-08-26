#!/bin/bash

# Check if two arguments are provided
if [ $# -lt 2 ]; then
    echo "Error: Insufficient arguments provided."
    echo "Usage: $0 <directory_path> <output_file>"
    exit 1
fi

directory="$1"
output_file="$2"

# Check if the provided directory path is empty
if [ -z "$directory" ]; then
    echo "Error: The provided directory path is empty."
    echo "Usage: $0 <directory_path> <output_file>"
    exit 1
fi

# Check if the directory exists
if [ ! -d "$directory" ]; then
    echo "Error: The directory '$directory' does not exist."
    exit 1
fi

# Check if the directory is empty
if [ -z "$(ls -A "$directory")" ]; then
    echo "The directory '$directory' is empty. No files to concatenate."
    exit 0
fi

# Clear the output file if it exists, or create it if it doesn't
> "$output_file"

# Concatenate files
for file in "$directory"/*; do
    if [ -f "$file" ]; then
        echo -e "\n# File: ${file#$directory/}\n" >> "$output_file"
        cat "$file" >> "$output_file"
        echo -e "\n#==== End of file: ${file#$directory/} ====\n" >> "$output_file"
    fi
done

echo "Files have been concatenated and saved to $output_file"

import json
import sys

def beautify_json(input_file, output_file):
    try:
        with open(input_file, 'r') as file:
            data = json.load(file)
        
        with open(output_file, 'w') as file:
            json.dump(data, file, indent=2, ensure_ascii=False)
        
        print(f"JSON file has been successfully beautified and saved to {output_file}")
    except json.JSONDecodeError:
        print("The input file is not a valid JSON format")
    except IOError:
        print("An error occurred while reading or writing the file")

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python script.py <input_file> <output_file>")
    else:
        input_file = sys.argv[1]
        output_file = sys.argv[2]
        beautify_json(input_file, output_file)
import json
import sys

def compare_json_files(file1, file2):
    try:
        with open(file1, 'r') as f1, open(file2, 'r') as f2:
            data1 = json.load(f1)
            data2 = json.load(f2)

        if not isinstance(data1, list) or not isinstance(data2, list):
            print("Both JSON files should contain lists as the top-level structure")
            return

        for i, (item1, item2) in enumerate(zip(data1, data2)):
            if item1 != item2:
                print(f"First difference found at index {i}:")
                diff_keys = set(item1.keys()) ^ set(item2.keys())
                common_keys = set(item1.keys()) & set(item2.keys())
                
                for key in diff_keys:
                    if key in item1:
                        print(f"  Key unique to file1: {key}: {item1[key]}")
                    else:
                        print(f"  Key unique to file2: {key}: {item2[key]}")
                
                for key in common_keys:
                    if item1[key] != item2[key]:
                        print(f"  Values differ for key '{key}':")
                        print(f"    File1: {item1[key]}")
                        print(f"    File2: {item2[key]}")
                
                # 格式化输出两个item
                print("  Item from file1:")
                print(json.dumps(item1, indent=4, ensure_ascii=False))
                print("  Item from file2:")
                print(json.dumps(item2, indent=4, ensure_ascii=False))
                return  # Return immediately after finding the first difference

        if len(data1) != len(data2):
            print(f"File lengths differ. File1 length: {len(data1)}, File2 length: {len(data2)}")
        else:
            print("The two files are identical")

    except json.JSONDecodeError:
        print("One or both input files are not valid JSON format")
    except IOError:
        print("An error occurred while reading the files")

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python compare_json.py <file1> <file2>")
    else:
        file1 = sys.argv[1]
        file2 = sys.argv[2]
        compare_json_files(file1, file2)
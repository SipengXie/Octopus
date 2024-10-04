import json
import sys

def beautify_json(input_file, output_file):
    try:
        with open(input_file, 'r') as file:
            data = json.load(file)
        
        with open(output_file, 'w') as file:
            json.dump(data, file, indent=2, ensure_ascii=False)
        
        print(f"JSON文件已成功美化并保存到 {output_file}")
    except json.JSONDecodeError:
        print("输入文件不是有效的JSON格式")
    except IOError:
        print("读取或写入文件时发生错误")

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("使用方法: python script.py <输入文件> <输出文件>")
    else:
        input_file = sys.argv[1]
        output_file = sys.argv[2]
        beautify_json(input_file, output_file)
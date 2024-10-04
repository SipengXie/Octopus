import json
import sys

def compare_json_files(file1, file2):
    try:
        with open(file1, 'r') as f1, open(file2, 'r') as f2:
            data1 = json.load(f1)
            data2 = json.load(f2)

        if not isinstance(data1, list) or not isinstance(data2, list):
            print("两个JSON文件都应该包含列表作为顶层结构")
            return

        for i, (item1, item2) in enumerate(zip(data1, data2)):
            if item1 != item2:
                print(f"在索引 {i} 处发现第一个差异:")
                diff_keys = set(item1.keys()) ^ set(item2.keys())
                common_keys = set(item1.keys()) & set(item2.keys())
                
                for key in diff_keys:
                    if key in item1:
                        print(f"  文件1中独有的键: {key}: {item1[key]}")
                    else:
                        print(f"  文件2中独有的键: {key}: {item2[key]}")
                
                for key in common_keys:
                    if item1[key] != item2[key]:
                        print(f"  键 '{key}' 的值不同:")
                        print(f"    文件1: {item1[key]}")
                        print(f"    文件2: {item2[key]}")
                
                return  # 找到第一个差异后立即返回

        if len(data1) != len(data2):
            print(f"文件长度不同。文件1的长度: {len(data1)}, 文件2的长度: {len(data2)}")
        else:
            print("两个文件完全相同")

    except json.JSONDecodeError:
        print("一个或两个输入文件不是有效的JSON格式")
    except IOError:
        print("读取文件时发生错误")

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("使用方法: python compare_json.py <文件1> <文件2>")
    else:
        file1 = sys.argv[1]
        file2 = sys.argv[2]
        compare_json_files(file1, file2)
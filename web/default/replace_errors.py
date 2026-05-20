import os
import re
import glob

def process_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()

    # 500 -> 30000 (server error)
    content = re.sub(r'nacosV3Err\(c,\s*500,', r'nacosV3Err(c, 30000,', content)
    
    # 404 -> 20004 (resource not found) generally
    content = re.sub(r'nacosV3Err\(c,\s*404,\s*"命名空间不存在"\)', r'nacosV3Err(c, 22001, "命名空间不存在")', content)
    content = re.sub(r'nacosV3Err\(c,\s*404,', r'nacosV3Err(c, 20004,', content)

    # 401, 403 -> 10001 (access denied)
    content = re.sub(r'nacosV3Err\(c,\s*401,', r'nacosV3Err(c, 10001,', content)
    content = re.sub(r'nacosV3Err\(c,\s*403,', r'nacosV3Err(c, 10001,', content)

    # 405 -> 20002 (parameter validate error)
    content = re.sub(r'nacosV3Err\(c,\s*405,', r'nacosV3Err(c, 20002,', content)

    # 400 -> 10000 (parameter missing) for messages with "必填" or "缺少"
    content = re.sub(r'nacosV3Err\(c,\s*400,\s*("[^"]*(必填|缺少)[^"]*")\)', r'nacosV3Err(c, 10000, \1)', content)

    # 400 -> 20002 (parameter validate error) for the rest
    content = re.sub(r'nacosV3Err\(c,\s*400,', r'nacosV3Err(c, 20002,', content)

    with open(filepath, 'w') as f:
        f.write(content)

for filepath in glob.glob('/opt/workspaces/uphone/k3d/xcph/one-api/controller/nacos_*.go'):
    process_file(filepath)

print("Done replacing.")

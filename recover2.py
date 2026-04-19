with open("/home/phucle/.gemini/antigravity/brain/78e2f01d-f5e7-4748-b420-e35d186e3b7f/.system_generated/logs/overview.txt", "r") as f:
    text = f.read()

import re
matches = re.findall(r'File Path: `file:///home/phucle/Desktop/new/aurora-controlplane/internal/iam/route\.go`.*?Total Lines: \d+.*?Showing lines 1 to \d+.*?(?:<line_number>: <original_line>\..*?)\n(.*?)The above content shows the entire, complete file', text, re.DOTALL)
if matches:
    content = matches[-1]
    lines = []
    for line in content.split('\n'):
        if line and ':' in line and line.split(':', 1)[0].isdigit():
            lines.append(line.split(':', 1)[1][1:])
        elif line == "":
            lines.append("")
    with open("recovered_route.go", "w") as out:
        out.write("\n".join(lines))
    print("Recovered!")
else:
    print("Not found full content, falling back to writing from scratch")

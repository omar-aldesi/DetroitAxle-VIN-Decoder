import os
import re
import subprocess

os.chdir(os.path.join(os.path.dirname(__file__), ".."))

with open("helpers/fitment.go", "r", encoding="utf-8") as f:
    content = f.read()

content = re.sub(r"/\* [─═][\s\S]*?\*/\n*", "", content)

parts = re.split(r"\n(?=func )", content)
header = parts[0].strip() + "\n\n"
chunks = [p.strip() for p in parts[1:] if p.strip()]

assign = {
    "fitment_resolve.go": {"normalizeKey", "resolveField"},
    "fitment_callout.go": {"evaluateCallout"},
    "fitment_match.go": {
        "ciEqual",
        "normModelStr",
        "containsWord",
        "matchesModelTokens",
        "matchesTrimList",
        "normDriveType",
        "displMatches",
    },
    "fitment_eval.go": {"EvaluateRule", "BestFitForPart", "FitResultString"},
}

files = {"fitment_types.go": [header]}
for k in assign:
    files[k] = []

for chunk in chunks:
    m = re.match(r"func (\w+)", chunk)
    name = m.group(1) if m else ""
    placed = False
    for fname, names in assign.items():
        if name in names:
            files[fname].append(chunk)
            placed = True
            break
    if not placed and name == "evaluateCallout":
        files["fitment_callout.go"].append(chunk)

# calloutStatus type lives before evaluateCallout — keep in callout file from header tail
callout_type = re.search(
    r"type calloutStatus int[\s\S]*?calloutMissing\s+calloutStatus = 2\n\)",
    content,
)
if callout_type:
    files["fitment_callout.go"].insert(0, callout_type.group(0))

for fname, parts_list in files.items():
    body = "\n\n".join(parts_list).strip() + "\n"
    body = re.sub(r"^package helpers\n*", "", body)
    body = re.sub(r"import \([^)]*\)\n*", "", body, count=1, flags=re.S)
    path = "helpers/" + fname
    with open(path, "w", encoding="utf-8", newline="\n") as out:
        out.write("package helpers\n\n" + body)
    print("wrote", path, len(parts_list), "parts")

os.remove("helpers/fitment.go")
subprocess.run(
    [
        "go",
        "run",
        "golang.org/x/tools/cmd/goimports@latest",
        "-w",
        "helpers/fitment_types.go",
        "helpers/fitment_resolve.go",
        "helpers/fitment_callout.go",
        "helpers/fitment_match.go",
        "helpers/fitment_eval.go",
    ],
    check=True,
)
print("removed helpers/fitment.go")

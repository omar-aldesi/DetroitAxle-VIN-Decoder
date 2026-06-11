import os
import re
import subprocess

os.chdir(os.path.join(os.path.dirname(__file__), ".."))

with open("services/fetch_gm.go", "r", encoding="utf-8") as f:
    content = f.read()

content = re.sub(
    r"// fetch_gm\.go —[\s\S]*?//     demand and shown read-only\. See FormatAsJSON \(served by /api/gm/decode\)\.\n\n",
    "",
    content,
)

parts = re.split(r"\n// --- ", content)
chunks = [parts[0]]
for p in parts[1:]:
    chunks.append("// --- " + p)

assign = {
    "gm_fetch.go": ["response types", "http fetch"],
    "gm_format.go": ["live formatter"],
    "gm_apply.go": ["persisted enrichment"],
    "gm_parse.go": ["brand detection", "shared utilities"],
}

files = {k: [] for k in assign}
files["gm_fetch.go"].append(chunks[0])

for chunk in chunks[1:]:
    name = chunk.split("\n", 1)[0].lower()
    for fname, keys in assign.items():
        if any(k in name for k in keys):
            files[fname].append(chunk)
            break

for fname, parts_list in files.items():
    body = "\n\n".join(parts_list).strip() + "\n"
    body = re.sub(r"^package services\n*", "", body)
    body = re.sub(r"import \([^)]*\)\n*", "", body, count=1, flags=re.S)
    path = "services/" + fname
    with open(path, "w", encoding="utf-8", newline="\n") as out:
        out.write("package services\n\n" + body)
    print("wrote", path)

os.remove("services/fetch_gm.go")
subprocess.run(
    [
        "go",
        "run",
        "golang.org/x/tools/cmd/goimports@latest",
        "-w",
        "services/gm_fetch.go",
        "services/gm_format.go",
        "services/gm_apply.go",
        "services/gm_parse.go",
    ],
    check=True,
)
print("removed services/fetch_gm.go")

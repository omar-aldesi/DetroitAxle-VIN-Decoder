import re
import os
import subprocess

os.chdir(os.path.join(os.path.dirname(__file__), ".."))

with open("services/fetch_vin.go", "r", encoding="utf-8") as f:
    content = f.read()

parts = re.split(r"\n// ─── ", content)
chunks = [parts[0]]
for p in parts[1:]:
    chunks.append("// ─── " + p)

assign = {
    "vin_types.go": ["auto.dev", "nhtsa", "result wrappers"],
    "vin_decode.go": ["entry point", "auto.dev fetch", "nhtsa fetch"],
    "vin_map.go": ["mapping"],
    "vin_parse.go": ["engine string parser"],
    "vin_normalize.go": ["field normalizers"],
    "vin_helpers.go": ["identity helpers", "helpers"],
}

files = {k: [] for k in assign}

for chunk in chunks:
    name = chunk.split("\n", 1)[0].lower()
    placed = False
    for fname, keys in assign.items():
        if any(k in name for k in keys):
            files[fname].append(chunk)
            placed = True
            break
    if not placed:
        files["vin_types.go"].append(chunk)

for fname, parts_list in files.items():
    body = "\n\n".join(parts_list).strip()
    body = re.sub(r"^package services\n*", "", body)
    body = re.sub(r"import \([^)]*\)\n*", "", body, count=1, flags=re.S)
    path = "services/" + fname
    with open(path, "w", encoding="utf-8", newline="\n") as out:
        out.write("package services\n\n" + body + "\n")
    print("wrote", path)

os.remove("services/fetch_vin.go")
subprocess.run(
    ["go", "run", "golang.org/x/tools/cmd/goimports@latest", "-w"]
    + [f"services/{k}" for k in assign],
    check=True,
)
print("removed services/fetch_vin.go")

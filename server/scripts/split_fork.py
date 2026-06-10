import re
import os

os.chdir(os.path.join(os.path.dirname(__file__), ".."))

with open("services/fork.go", "r", encoding="utf-8") as f:
    content = f.read()

content = re.sub(
    r"^package services\n\n// fork\.go.*?\n\n",
    "package services\n\n",
    content,
    flags=re.S,
)

parts = re.split(r"\n// ─── ", content)
chunks = []
if parts:
    chunks.append(parts[0])
    for p in parts[1:]:
        chunks.append("// ─── " + p)

files = {
    "fork_types.go": [],
    "fork_engine.go": [],
    "fork_resolve.go": [],
    "fork_internal.go": [],
    "fork_store.go": [],
    "fork_integration.go": [],
    "fork_notes.go": [],
    "fork_seed.go": [],
}

for chunk in chunks:
    name = chunk.split("\n", 1)[0].lower()
    if "recordpoint" in name or "recordrange" in name or "normalization" in name:
        files["fork_engine.go"].append(chunk)
    elif "resolve" in name and "edit-path" not in name:
        files["fork_resolve.go"].append(chunk)
    elif "helpers" in name and "edit-path" not in name:
        files["fork_internal.go"].append(chunk)
    elif "gorm-backed" in name:
        files["fork_store.go"].append(chunk)
    elif "edit-path" in name:
        files["fork_integration.go"].append(chunk)
    elif "note applicability" in name:
        files["fork_notes.go"].append(chunk)
    elif "seeding" in name:
        files["fork_seed.go"].append(chunk)
    else:
        files["fork_types.go"].append(chunk)

imports = """package services

import (
\t\"math\"
\t\"sort\"
\t\"strconv\"
\t\"strings\"
\t\"time\"

\t\"main/models\"

\t\"gorm.io/gorm\"
)
"""

for fname, parts_list in files.items():
    body = "\n\n".join(parts_list).strip()
    if not body or body == "package services":
        continue
    body = re.sub(r"^package services\n*", "", body)
    body = re.sub(r"import \([^)]*\)\n*", "", body, count=1, flags=re.S)
    path = "services/" + fname
    with open(path, "w", encoding="utf-8", newline="\n") as out:
        out.write(imports + "\n" + body + "\n")
    print("wrote", path, len(body.splitlines()), "lines")

os.remove("services/fork.go")
print("removed services/fork.go")

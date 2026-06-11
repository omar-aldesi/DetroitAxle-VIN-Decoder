import os
import re

root = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))

def clean(content):
    lines = content.splitlines(keepends=True)
    out = []
    skip_block = False
    for line in lines:
        stripped = line.strip()
        if stripped.startswith("/*") and any(c in stripped for c in "═─"):
            skip_block = "*/" not in stripped
            continue
        if skip_block:
            if "*/" in stripped:
                skip_block = False
            continue
        if re.match(r"^// [─═\-]{2,}", stripped):
            continue
        if re.match(r"^// ──", stripped):
            continue
        line = re.sub(r" // (unverified|held).*$", "", line.rstrip("\r\n")) + "\n"
        line = re.sub(r" // caller should treat as a build-key field", "", line)
        out.append(line)
    content = "".join(out)
    content = re.sub(r"\n{3,}", "\n\n", content)
    return content

for dirpath, _, filenames in os.walk(root):
    if os.path.basename(dirpath) == "scripts":
        continue
    for fn in filenames:
        if not fn.endswith(".go"):
            continue
        path = os.path.join(dirpath, fn)
        with open(path, "r", encoding="utf-8") as f:
            original = f.read()
        updated = clean(original)
        if updated != original:
            with open(path, "w", encoding="utf-8", newline="\n") as f:
                f.write(updated)
            print("cleaned", os.path.relpath(path, root))

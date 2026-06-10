import glob
import os
import re

os.chdir(os.path.join(os.path.dirname(__file__), ".."))

targets = []
targets += glob.glob("services/fork_*.go")
targets += glob.glob("services/vin_*.go")
targets += glob.glob("handlers/dnr_*.go")
targets += glob.glob("handlers/parts_*.go")
targets += ["handlers/fork.go", "services/fork_test.go"]


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
        line = re.sub(r" // (unverified|held).*$", "", line.rstrip("\r\n")) + "\n"
        out.append(line)
    content = "".join(out)
    return re.sub(r"\n{3,}", "\n\n", content)


for path in targets:
    with open(path, "r", encoding="utf-8") as f:
        original = f.read()
    updated = clean(original)
    if updated != original:
        with open(path, "w", encoding="utf-8", newline="\n") as f:
            f.write(updated)
        print("cleaned", path)

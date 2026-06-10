import os
import subprocess

os.chdir(os.path.join(os.path.dirname(__file__), ".."))

with open("handlers/parts.go", "r", encoding="utf-8") as f:
    lines = f.readlines()

imports_end = next(i for i, l in enumerate(lines) if l.strip() == ")") + 1
imports = "".join(lines[: imports_end + 1]) + "\n"


def write_file(name, ranges):
    body = ""
    for start, end in ranges:
        body += "".join(lines[start:end])
    body = body.strip() + "\n"
    path = f"handlers/{name}"
    with open(path, "w", encoding="utf-8", newline="\n") as out:
        out.write(imports + body)
    print("wrote", path)


write_file("parts_handler.go", [(20, 23)])
write_file("parts_helpers.go", [(24, 29), (339, 376)])
write_file("parts_crud.go", [(34, 265)])
write_file("parts_rules.go", [(270, 338)])
write_file("parts_fitment.go", [(381, len(lines))])

os.remove("handlers/parts.go")
files = [
    "handlers/parts_handler.go",
    "handlers/parts_helpers.go",
    "handlers/parts_crud.go",
    "handlers/parts_rules.go",
    "handlers/parts_fitment.go",
]
subprocess.run(
    ["go", "run", "golang.org/x/tools/cmd/goimports@latest", "-w", *files],
    check=True,
)
print("removed handlers/parts.go")

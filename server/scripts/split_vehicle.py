import os
import subprocess

os.chdir(os.path.join(os.path.dirname(__file__), ".."))

with open("handlers/vehicle.go", "r", encoding="utf-8") as f:
    lines = f.readlines()

imports_end = next(i for i, l in enumerate(lines) if l.strip() == ")") + 1
imports = "".join(lines[: imports_end + 1]) + "\n"


def write_file(name, ranges):
    body = ""
    for start, end in ranges:
        body += "".join(lines[start:end])
    path = f"handlers/{name}"
    with open(path, "w", encoding="utf-8", newline="\n") as out:
        out.write(imports + body.strip() + "\n")
    print("wrote", path)


write_file("vehicle_handler.go", [(20, 23)])
write_file("vehicle_get.go", [(24, 169)])
write_file("vehicle_update.go", [(169, 351)])
write_file("vehicle_list.go", [(351, len(lines))])

os.remove("handlers/vehicle.go")
files = [
    "handlers/vehicle_handler.go",
    "handlers/vehicle_get.go",
    "handlers/vehicle_update.go",
    "handlers/vehicle_list.go",
]
subprocess.run(
    ["go", "run", "golang.org/x/tools/cmd/goimports@latest", "-w", *files],
    check=True,
)
print("removed handlers/vehicle.go")

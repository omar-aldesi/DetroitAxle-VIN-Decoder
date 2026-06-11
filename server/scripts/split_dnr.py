import os
import subprocess

os.chdir(os.path.join(os.path.dirname(__file__), ".."))

with open("handlers/dnr.go", "r", encoding="utf-8") as f:
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


write_file("dnr_spec.go", [(21, 125)])
write_file("dnr_queue.go", [(126, 276)])
write_file("dnr_stats.go", [(277, 332)])
write_file("dnr_vehicle.go", [(333, 376)])
write_file("dnr_propagate.go", [(377, len(lines))])

os.remove("handlers/dnr.go")
files = [
    "handlers/dnr_spec.go",
    "handlers/dnr_queue.go",
    "handlers/dnr_stats.go",
    "handlers/dnr_vehicle.go",
    "handlers/dnr_propagate.go",
]
subprocess.run(
    ["go", "run", "golang.org/x/tools/cmd/goimports@latest", "-w", *files],
    check=True,
)
print("removed handlers/dnr.go")

import os
import re

root = os.path.join(os.path.dirname(__file__), "..")
src = os.path.join(root, "src", "pages", "VehiclePage.jsx")
out_dir = os.path.join(root, "src", "components", "vehicle")
os.makedirs(out_dir, exist_ok=True)

with open(src, "r", encoding="utf-8") as f:
    lines = f.readlines()

RANGES = {
    "constants.js": (41, 143),
    "CopyBtn.jsx": (146, 174),
    "SpecRow.jsx": (177, 301),
    "SpecSection.jsx": (304, 368),
    "CustomFieldsSection.jsx": (371, 522),
    "gmUtils.js": (527, 696),
    "GMLiveSection.jsx": (700, 834),
    "BuildNumberSpecs.jsx": (837, 1321),
    "RecordMeta.jsx": (1324, 1350),
    "HeroBadge.jsx": (1353, 1361),
    "QuickVinSearch.jsx": (1364, 1407),
}

for fname, (start, end) in RANGES.items():
    chunk = "".join(lines[start - 1 : end])
    chunk = re.sub(r"/\* [═─][\s\S]*?\*/\n*", "", chunk)
    path = os.path.join(out_dir, fname)
    if fname.endswith(".js"):
        chunk = chunk.replace("const ", "export const ")
        chunk = chunk.replace("function isGMVehicle", "export function isGMVehicle")
        chunk = chunk.replace("function formatGMText", "export function formatGMText")
        chunk = chunk.replace("function parseRPO", "export function parseRPO")
        chunk = chunk.replace("function gmVinFor", "export function gmVinFor")
    else:
        chunk = chunk.replace("function ", "export function ")
    with open(path, "w", encoding="utf-8", newline="\n") as out:
        out.write(chunk.strip() + "\n")
    print("wrote", fname)

page_start = 1412
page = "".join(lines[page_start - 1 :])
page = re.sub(r"/\* [═─][\s\S]*?\*/\n*", "", page)
page = """import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeft,
  Fingerprint,
  RefreshCw,
  AlertCircle,
  WifiOff,
  ChevronsUpDown,
} from "lucide-react";
import { getVehicle } from "../api/vehicles";
import ThemeToggle from "../components/ThemeToggle";
import NotesPanel from "../components/NotesPanel";
import { SPEC_SECTIONS } from "../components/vehicle/constants";
import { CopyBtn } from "../components/vehicle/CopyBtn";
import { SpecSection } from "../components/vehicle/SpecSection";
import { CustomFieldsSection } from "../components/vehicle/CustomFieldsSection";
import { BuildNumberSpecs } from "../components/vehicle/BuildNumberSpecs";
import { GMLiveSection } from "../components/vehicle/GMLiveSection";
import { HeroBadge } from "../components/vehicle/HeroBadge";
import { QuickVinSearch } from "../components/vehicle/QuickVinSearch";

""" + page.replace("export default function VehiclePage", "export default function VehiclePage")

with open(src, "w", encoding="utf-8", newline="\n") as out:
    out.write(page.strip() + "\n")
print("rewrote VehiclePage.jsx")

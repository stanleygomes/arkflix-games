"""Confere as regras de busca do ES-DE local: XML válido e todo caminho existe."""

import sys
import xml.etree.ElementTree as ET
from pathlib import Path

rules = ET.parse(sys.argv[1]).getroot()
missing = [
  e.text for e in rules.iter("entry") if not e.text or not Path(e.text).exists()
]
for path in missing:
  print(f"❌ não existe: {path}")
print("✅ regras do ES-DE ok" if not missing else "rode make build / make wrappers")
sys.exit(1 if missing else 0)

#!/usr/bin/env python3
"""Build the Windows icon + VERSIONINFO COFF resource for OpenAvionicsBridge.

Requirements: Python 3 and clang with the x86_64-w64-windows-gnu target.
Run from the repository root:
    python tools/build-windows-resources.py

Set OAB_VERSION to override the default version string used in VERSIONINFO.
"""
from pathlib import Path
import os
import shutil
import struct
import subprocess

ROOT = Path(__file__).resolve().parent.parent
TOOLS = ROOT / "tools"
GEN = TOOLS / ".generated-resources"
ICON = ROOT / "OpenAvionicsBridge.ico"
OUT = ROOT / "resource_windows_amd64.syso"
VERSION = os.environ.get("OAB_VERSION", "1.0.0-rc3")


def pad4(buf: bytearray) -> None:
    while len(buf) % 4:
        buf.append(0)


def wstr(value: str) -> bytes:
    return value.encode("utf-16le") + b"\x00\x00"


def version_parts(text: str):
    # Numeric Windows file version; prerelease suffixes are represented in string fields.
    nums = []
    for token in text.split("-")[0].split("."):
        try:
            nums.append(int(token))
        except ValueError:
            nums.append(0)
    nums = (nums + [0, 0, 0, 0])[:4]
    return nums


def block(key: str, value: bytes = b"", value_length: int = 0, value_type: int = 1, children=()):
    b = bytearray(b"\x00" * 6)
    b += wstr(key)
    pad4(b)
    b += value
    pad4(b)
    for child in children:
        b += child
        pad4(b)
    struct.pack_into("<HHH", b, 0, len(b), value_length, value_type)
    return bytes(b)


def string_block(key: str, value: str):
    raw = wstr(value)
    # For text values, wValueLength is measured in UTF-16 code units including NUL.
    return block(key, raw, len(raw) // 2, 1)


def build_version_blob(version: str) -> bytes:
    a, b, c, d = version_parts(version)
    fixed = struct.pack(
        "<13I",
        0xFEEF04BD,  # dwSignature
        0x00010000,  # dwStrucVersion
        (a << 16) | b,
        (c << 16) | d,
        (a << 16) | b,
        (c << 16) | d,
        0x0000003F,  # dwFileFlagsMask
        0x00000002 if "-" in version else 0,  # VS_FF_PRERELEASE
        0x00040004,  # VOS_NT_WINDOWS32
        0x00000001,  # VFT_APP
        0,
        0,
        0,
    )
    strings = [
        ("CompanyName", "leboerF"),
        ("FileDescription", "OpenAvionicsBridge"),
        ("FileVersion", version),
        ("InternalName", "OpenAvionicsBridge"),
        ("LegalCopyright", "© 2026 leboerF"),
        ("OriginalFilename", "OpenAvionicsBridge.exe"),
        ("ProductName", "OpenAvionicsBridge"),
        ("ProductVersion", version),
        ("Comments", "MSFS avionics hardware bridge · MIT License"),
    ]
    table = block("040904B0", children=[string_block(k, v) for k, v in strings])
    sfi = block("StringFileInfo", children=[table])
    translation = block("Translation", struct.pack("<HH", 0x0409, 0x04B0), 4, 0)
    vfi = block("VarFileInfo", children=[translation])
    return block("VS_VERSION_INFO", fixed, len(fixed), 0, [sfi, vfi])


def parse_icon(path: Path):
    raw = path.read_bytes()
    reserved, typ, count = struct.unpack_from("<HHH", raw, 0)
    if (reserved, typ) != (0, 1) or count < 1:
        raise SystemExit("Not a valid Windows ICO file")
    entries = []
    for i in range(count):
        off = 6 + i * 16
        w, h, cc, res, planes, bpp, size, ofs = struct.unpack_from("<BBBBHHII", raw, off)
        entries.append({
            "id": i + 1,
            "w": w,
            "h": h,
            "cc": cc,
            "res": res,
            "planes": planes,
            "bpp": bpp,
            "size": size,
            "data": raw[ofs:ofs + size],
        })
    return entries


def normalize_coff_resource_relocs(path: Path):
    data = bytearray(path.read_bytes())
    machine, nsec, _ts, symptr, nsym, optsz, _chars = struct.unpack_from("<HHIIIHH", data, 0)
    if machine != 0x8664:
        raise RuntimeError("Expected AMD64 COFF object")
    sec_base = 20 + optsz
    sections = []
    for i in range(nsec):
        off = sec_base + i * 40
        name = bytes(data[off:off+8]).split(b"\0", 1)[0].decode("ascii", "replace")
        size = struct.unpack_from("<I", data, off+16)[0]
        rawptr = struct.unpack_from("<I", data, off+20)[0]
        relptr = struct.unpack_from("<I", data, off+24)[0]
        nrel = struct.unpack_from("<H", data, off+32)[0]
        sections.append((name, size, rawptr, relptr, nrel))
    try:
        _, _, rawptr, relptr, nrel = next(x for x in sections if x[0] == ".rsrc")
    except StopIteration:
        raise RuntimeError("No .rsrc section in COFF object")

    strtab_off = symptr + nsym * 18
    symbols = {}
    i = 0
    while i < nsym:
        off = symptr + i * 18
        rawname = bytes(data[off:off+8])
        if rawname[:4] == b"\0\0\0\0":
            stroff = struct.unpack_from("<I", rawname, 4)[0]
            end = data.find(0, strtab_off + stroff)
            name = bytes(data[strtab_off+stroff:end]).decode("ascii", "replace")
        else:
            name = rawname.split(b"\0", 1)[0].decode("ascii", "replace")
        value = struct.unpack_from("<I", data, off+8)[0]
        aux = data[off+17]
        symbols[i] = (name, value)
        i += 1 + aux
    rsrc_idx = next(idx for idx, (name, _v) in symbols.items() if name == ".rsrc")

    for j in range(nrel):
        roff = relptr + j * 10
        vaddr, symidx, rtype = struct.unpack_from("<IIH", data, roff)
        if rtype != 0x0003:  # IMAGE_REL_AMD64_ADDR32NB
            continue
        name, addend = symbols.get(symidx, ("", 0))
        if not name.startswith(("icon_blob_", "group_blob_", "version_blob_")):
            continue
        struct.pack_into("<I", data, rawptr + vaddr, addend)
        struct.pack_into("<I", data, roff + 4, rsrc_idx)
    path.write_bytes(data)


def main():
    if not ICON.exists():
        raise SystemExit(f"Missing {ICON.name}")
    if shutil.which("clang") is None:
        raise SystemExit("clang not found")

    entries = parse_icon(ICON)
    if GEN.exists():
        shutil.rmtree(GEN)
    GEN.mkdir(parents=True)
    for e in entries:
        (GEN / f'icon_{e["id"]}.bin').write_bytes(e["data"])
    version_blob = build_version_blob(VERSION)
    (GEN / "version_info.bin").write_bytes(version_blob)

    L = []
    a = L.append
    a('.section .rsrc,"dr"')
    a('.balign 4')
    a('rsrc_start:')
    a('  .long 0, 0')
    a('  .short 0, 0, 0, 3')
    a('  .long 3')
    a('  .long 0x80000000 + icon_type_dir - rsrc_start')
    a('  .long 14')
    a('  .long 0x80000000 + group_type_dir - rsrc_start')
    a('  .long 16')
    a('  .long 0x80000000 + version_type_dir - rsrc_start')

    a('icon_type_dir:')
    a('  .long 0, 0')
    a(f'  .short 0, 0, 0, {len(entries)}')
    for e in entries:
        a(f'  .long {e["id"]}')
        a(f'  .long 0x80000000 + icon_lang_{e["id"]} - rsrc_start')

    a('group_type_dir:')
    a('  .long 0, 0')
    a('  .short 0, 0, 0, 1')
    a('  .long 1')
    a('  .long 0x80000000 + group_lang_1 - rsrc_start')

    a('version_type_dir:')
    a('  .long 0, 0')
    a('  .short 0, 0, 0, 1')
    a('  .long 1')
    a('  .long 0x80000000 + version_lang_1 - rsrc_start')

    for e in entries:
        a(f'icon_lang_{e["id"]}:')
        a('  .long 0, 0')
        a('  .short 0, 0, 0, 1')
        a('  .long 0x0409')
        a(f'  .long icon_data_entry_{e["id"]} - rsrc_start')

    a('group_lang_1:')
    a('  .long 0, 0')
    a('  .short 0, 0, 0, 1')
    a('  .long 0x0409')
    a('  .long group_data_entry_1 - rsrc_start')

    a('version_lang_1:')
    a('  .long 0, 0')
    a('  .short 0, 0, 0, 1')
    a('  .long 0x0409')
    a('  .long version_data_entry_1 - rsrc_start')

    for e in entries:
        a(f'icon_data_entry_{e["id"]}:')
        a(f'  .rva icon_blob_{e["id"]}')
        a(f'  .long {e["size"]}, 0, 0')
    a('group_data_entry_1:')
    a('  .rva group_blob_1')
    a(f'  .long {6 + 14 * len(entries)}, 0, 0')
    a('version_data_entry_1:')
    a('  .rva version_blob_1')
    a(f'  .long {len(version_blob)}, 0, 0')

    for e in entries:
        a('  .balign 4')
        a(f'icon_blob_{e["id"]}:')
        a(f'  .incbin ".generated-resources/icon_{e["id"]}.bin"')

    a('  .balign 4')
    a('group_blob_1:')
    a(f'  .short 0, 1, {len(entries)}')
    for e in entries:
        a(f'  .byte {e["w"]}, {e["h"]}, {e["cc"]}, {e["res"]}')
        a(f'  .short {e["planes"]}, {e["bpp"]}')
        a(f'  .long {e["size"]}')
        a(f'  .short {e["id"]}')

    a('  .balign 4')
    a('version_blob_1:')
    a('  .incbin ".generated-resources/version_info.bin"')
    a('  .balign 4')

    asm = TOOLS / 'resource_windows_amd64.s'
    asm.write_text('\n'.join(L) + '\n', encoding='utf-8')
    subprocess.run([
        'clang', '--target=x86_64-w64-windows-gnu', '-c', asm.name,
        '-o', str(OUT)
    ], cwd=TOOLS, check=True)
    normalize_coff_resource_relocs(OUT)
    print(f'Built {OUT.name}: {len(entries)} icon sizes + VERSIONINFO {VERSION}')


if __name__ == '__main__':
    main()

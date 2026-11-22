#!/usr/bin/env python3
"""Собирает веб-версию в один файл Pyatimirye.html.

Внутри: wasm_exec.js и game.wasm (сжатый gzip, в base64). Такой файл
открывается двойным щелчком прямо с диска (file://), без веб-сервера:
игра распаковывается в браузере через DecompressionStream.

    python3 scripts/make-html.py dist/web dist/Pyatimirye.html
"""
import base64, gzip, sys, pathlib

src, out = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
wasm = base64.b64encode(gzip.compress((src / "game.wasm").read_bytes(), 9)).decode()
js = (src / "wasm_exec.js").read_text()
out.write_text(f"""<!doctype html><html lang="ru"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1"><title>Пятимирье</title>
<style>html,body{{margin:0;height:100%;background:#0b0a12;color:#d8cfe8;overflow:hidden}}
#l{{position:fixed;inset:0;display:flex;align-items:center;justify-content:center;font:20px sans-serif}}</style>
</head><body><div id="l">Пятимирье загружается…</div>
<script>{js}</script>
<script id="w" type="application/octet-stream">{wasm}</script>
<script>
(async () => {{
  const bin = Uint8Array.from(atob(document.getElementById("w").textContent), c => c.charCodeAt(0));
  const raw = await new Response(new Blob([bin]).stream().pipeThrough(new DecompressionStream("gzip"))).arrayBuffer();
  const go = new Go();
  const r = await WebAssembly.instantiate(raw, go.importObject);
  document.getElementById("l").remove();
  go.run(r.instance);
}})().catch(e => {{ document.getElementById("l").textContent = "Не удалось запустить: " + e; }});
</script></body></html>""", encoding="utf-8")
print(out, out.stat().st_size // 1024, "КБ")

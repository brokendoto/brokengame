#!/usr/bin/env python3
"""Собирает веб-версию в один файл Pyatimirye.html.

Внутри: wasm_exec.js и game.wasm (сжатый deflate, в base64). Такой файл
открывается двойным щелчком прямо с диска (file://), без веб-сервера,
и так же работает, если выложить его на любой хостинг (GitHub Pages и т. п.).

Что делает загрузчик, чтобы игра запускалась везде:
  * распаковывает игру встроенным в браузер DecompressionStream, а если его
    нет (старые Safari/Firefox) — своим распаковщиком на JavaScript;
  * заранее проверяет WebAssembly и WebGL 2 и, если чего-то нет, пишет
    понятным языком, что делать, вместо чёрного экрана;
  * при открытии с диска отключает AudioWorklet (из file:// он не грузится,
    и звука бы не было) — звук идёт через ScriptProcessor;
  * на телефоне: запрещает прокрутку и масштабирование страницы жестами,
    по первому касанию разворачивает игру на весь экран (где браузер умеет)
    и просит повернуть телефон горизонтально.

    python3 scripts/make-html.py dist/web dist/Pyatimirye.html [--pages]

--pages добавляет ссылку на manifest.webmanifest (установка на главный экран);
сам манифест и иконку кладёт рядом scripts/make-pages.sh.
"""
import base64, sys, pathlib, zlib

args = [a for a in sys.argv[1:] if not a.startswith("--")]
pages = "--pages" in sys.argv
src, out = pathlib.Path(args[0]), pathlib.Path(args[1])
here = pathlib.Path(__file__).parent

co = zlib.compressobj(9, zlib.DEFLATED, -15)  # «сырой» deflate без заголовков
raw = (src / "game.wasm").read_bytes()
wasm = base64.b64encode(co.compress(raw) + co.flush()).decode()
js = (src / "wasm_exec.js").read_text()
icon = base64.b64encode((here.parent / "internal/assets/icon_256.png").read_bytes()).decode()
loader = (here / "loader.js").read_text()
manifest = '<link rel="manifest" href="manifest.webmanifest">' if pages else ""

out.write_text(f"""<!doctype html><html lang="ru"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1,user-scalable=no,viewport-fit=cover">
<meta name="mobile-web-app-capable" content="yes"><meta name="apple-mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-status-bar-style" content="black"><meta name="theme-color" content="#0b0a12">
<meta name="description" content="Пятимирье — аркада о пяти мирах: собери всё и успей к выходу.">
<title>Пятимирье</title><link rel="icon" href="data:image/png;base64,{icon}">
<link rel="apple-touch-icon" href="data:image/png;base64,{icon}">{manifest}
<style>
html,body{{margin:0;height:100%;background:#0b0a12;color:#d8cfe8;overflow:hidden;
  overscroll-behavior:none;touch-action:none;-webkit-user-select:none;user-select:none;
  -webkit-touch-callout:none;-webkit-tap-highlight-color:transparent;font-family:system-ui,sans-serif}}
canvas{{touch-action:none}}
#l{{position:fixed;inset:0;display:flex;flex-direction:column;gap:18px;align-items:center;justify-content:center;
  text-align:center;padding:24px;box-sizing:border-box;z-index:10;background:radial-gradient(circle at 50% 40%,#1d1630,#0b0a12 70%)}}
#l img{{width:96px;height:96px;border-radius:22px;box-shadow:0 0 40px #6ad8ff55}}
#l b{{font-size:30px;letter-spacing:3px;color:#f0e8dc}}
#bar{{width:min(320px,70vw);height:6px;border-radius:3px;background:#ffffff18;overflow:hidden}}
#bar i{{display:block;height:100%;width:0;background:linear-gradient(90deg,#6ad8ff,#c87aff);transition:width .2s}}
#msg{{font-size:16px;color:#a99fbb;max-width:560px;line-height:1.45}}
#msg.err{{color:#ffb0b0}}
#rot{{position:fixed;inset:0;display:none;flex-direction:column;gap:14px;align-items:center;justify-content:center;
  background:#0b0a12ee;z-index:9;font-size:18px;text-align:center;padding:24px}}
#rot span{{font-size:64px;animation:r 2s ease-in-out infinite}}
@keyframes r{{0%,30%{{transform:rotate(0)}}60%,100%{{transform:rotate(90deg)}}}}
#rot u{{color:#6ad8ff;font-size:15px}}
@media (orientation:portrait) and (pointer:coarse){{body.run:not(.okp) #rot{{display:flex}}}}
</style></head><body>
<div id="l"><img src="data:image/png;base64,{icon}" alt=""><b>ПЯТИМИРЬЕ</b><div id="bar"><i></i></div><div id="msg">Загрузка…</div></div>
<div id="rot"><span>📱</span>Поверните телефон горизонтально<u>или нажмите, чтобы играть так</u></div>
<script>{js}</script>
<script id="w" type="application/octet-stream" data-size="{len(raw)}">{wasm}</script>
<script>{loader}</script>
</body></html>""", encoding="utf-8")
print(out, out.stat().st_size // 1024, "КБ")

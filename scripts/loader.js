// Загрузчик веб-версии Пятимирья. Вставляется в Pyatimirye.html скриптом
// make-html.py. Пишется на «старом» JavaScript без сборщика, чтобы работать
// и в старых браузерах телефонов.
(function () {
  "use strict";
  var $ = function (id) { return document.getElementById(id); };
  var msg = $("msg"), bar = $("bar").firstChild, box = $("l");
  var log = []; // последние строки консоли — показать, если игра упадёт
  ["log", "warn", "error"].forEach(function (k) {
    var orig = console[k];
    console[k] = function () {
      try { log.push(Array.prototype.join.call(arguments, " ")); if (log.length > 12) log.shift(); } catch (e) {}
      return orig.apply(console, arguments);
    };
  });
  function step(text, p) { msg.textContent = text; bar.style.width = p + "%"; }
  function fail(html, details) {
    if (!box.parentNode) document.body.appendChild(box);
    box.style.display = "flex";
    bar.parentNode.style.display = "none";
    msg.className = "err";
    msg.innerHTML = html + (details ? '<pre style="text-align:left;font-size:11px;white-space:pre-wrap;opacity:.7;max-height:30vh;overflow:auto">' +
      String(details).replace(/[<&]/g, function (c) { return c === "<" ? "&lt;" : "&amp;"; }) + "</pre>" : "") +
      '<p><button onclick="location.reload()" style="font-size:16px;padding:8px 18px;border-radius:10px;border:0;background:#6ad8ff;color:#0b0a12">Перезапустить</button></p>';
  }
  var frame = function () { return new Promise(function (r) { (window.requestAnimationFrame || setTimeout)(function () { setTimeout(r, 0); }); }); };

  // --- base64 → байты (atob на 8 МБ строки медленный и съедает память) ---
  function unbase64(s) {
    var T = new Uint8Array(128), A = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    for (var i = 0; i < 64; i++) T[A.charCodeAt(i)] = i;
    var n = s.length; while (n && (s.charCodeAt(n - 1) === 61 || s.charCodeAt(n - 1) <= 32)) n--;
    var b = 0; while (b < n && s.charCodeAt(b) <= 32) b++;
    var len = ((n - b) * 3) >> 2, out = new Uint8Array(len), o = 0;
    for (i = b; i + 3 < n; i += 4) {
      var v = T[s.charCodeAt(i)] << 18 | T[s.charCodeAt(i + 1)] << 12 | T[s.charCodeAt(i + 2)] << 6 | T[s.charCodeAt(i + 3)];
      out[o++] = v >> 16; out[o++] = v >> 8 & 255; out[o++] = v & 255;
    }
    var rest = n - i;
    if (rest > 1) {
      v = T[s.charCodeAt(i)] << 18 | T[s.charCodeAt(i + 1)] << 12 | (rest > 2 ? T[s.charCodeAt(i + 2)] << 6 : 0);
      out[o++] = v >> 16; if (rest > 2) out[o++] = v >> 8 & 255;
    }
    return out;
  }

  // --- распаковка deflate на JavaScript, если браузер не умеет сам ---
  function inflate(src, size) {
    var out = new Uint8Array(size), op = 0, bp = 0, bb = 0, bc = 0;
    function bits(n) {
      while (bc < n) { bb |= src[bp++] << bc; bc += 8; }
      var v = bb & ((1 << n) - 1); bb >>>= n; bc -= n; return v;
    }
    function build(lens, n) {
      var counts = new Uint16Array(16), offs = new Uint16Array(16), syms = new Uint16Array(n), i;
      for (i = 0; i < n; i++) counts[lens[i]]++;
      counts[0] = 0;
      for (i = 1; i < 16; i++) offs[i] = offs[i - 1] + counts[i - 1];
      for (i = 0; i < n; i++) if (lens[i]) syms[offs[lens[i]]++] = i;
      return { c: counts, s: syms };
    }
    function decode(t) {
      var code = 0, first = 0, index = 0;
      for (var len = 1; len < 16; len++) {
        code |= bits(1);
        var c = t.c[len];
        if (code - c < first) return t.s[index + (code - first)];
        index += c; first += c; first <<= 1; code <<= 1;
      }
      throw new Error("повреждённые данные");
    }
    var LB = [3, 4, 5, 6, 7, 8, 9, 10, 11, 13, 15, 17, 19, 23, 27, 31, 35, 43, 51, 59, 67, 83, 99, 115, 131, 163, 195, 227, 258],
      LE = [0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5, 0],
      DB = [1, 2, 3, 4, 5, 7, 9, 13, 17, 25, 33, 49, 65, 97, 129, 193, 257, 385, 513, 769, 1025, 1537, 2049, 3073, 4097, 6145, 8193, 12289, 16385, 24577],
      DE = [0, 0, 0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10, 10, 11, 11, 12, 12, 13, 13],
      ORD = [16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15];
    var last, i;
    do {
      last = bits(1);
      var type = bits(2), lt, dt;
      if (type === 0) {
        bb = 0; bc = 0;
        var len = src[bp] | src[bp + 1] << 8; bp += 4;
        out.set(src.subarray(bp, bp + len), op); op += len; bp += len;
        continue;
      } else if (type === 1) {
        var l = new Uint8Array(288);
        for (i = 0; i < 144; i++) l[i] = 8;
        for (; i < 256; i++) l[i] = 9;
        for (; i < 280; i++) l[i] = 7;
        for (; i < 288; i++) l[i] = 8;
        lt = build(l, 288);
        var d = new Uint8Array(30); for (i = 0; i < 30; i++) d[i] = 5;
        dt = build(d, 30);
      } else if (type === 2) {
        var hlit = bits(5) + 257, hdist = bits(5) + 1, hclen = bits(4) + 4;
        var cl = new Uint8Array(19);
        for (i = 0; i < hclen; i++) cl[ORD[i]] = bits(3);
        var ct = build(cl, 19), lens = new Uint8Array(hlit + hdist);
        for (i = 0; i < hlit + hdist;) {
          var sym = decode(ct);
          if (sym < 16) lens[i++] = sym;
          else {
            var rep, val = 0;
            if (sym === 16) { val = lens[i - 1]; rep = 3 + bits(2); }
            else if (sym === 17) rep = 3 + bits(3);
            else rep = 11 + bits(7);
            while (rep--) lens[i++] = val;
          }
        }
        lt = build(lens.subarray(0, hlit), hlit);
        dt = build(lens.subarray(hlit), hdist);
      } else throw new Error("повреждённые данные");
      for (;;) {
        var s = decode(lt);
        if (s < 256) out[op++] = s;
        else if (s === 256) break;
        else {
          s -= 257;
          var n = LB[s] + bits(LE[s]), ds = decode(dt), dist = DB[ds] + bits(DE[ds]);
          for (var k = 0; k < n; k++, op++) out[op] = out[op - dist];
        }
      }
    } while (!last);
    return out;
  }

  function unpack(bin, size) {
    if (typeof DecompressionStream === "function" && typeof Response === "function") {
      try {
        var ds = new DecompressionStream("deflate-raw");
        return new Response(new Blob([bin]).stream().pipeThrough(ds)).arrayBuffer()
          .catch(function () { return inflate(bin, size); });
      } catch (e) { /* формат deflate-raw не поддерживается — распакуем сами */ }
    }
    return frame().then(function () { return inflate(bin, size); });
  }

  // Во встроенных окнах (iframe на itch.io, превью, мессенджеры) политика
  // страницы может запрещать геймпады: тогда getGamepads бросает ошибку, и
  // Ebitengine падает. Подменяем на безопасную версию — без геймпадов игра
  // просто работает с клавиатурой и касаниями. Так же страхуем вибрацию.
  try {
    var nav = navigator, gp = nav.getGamepads || nav.webkitGetGamepads;
    nav.getGamepads = function () {
      try { return gp ? gp.call(nav) || [] : []; } catch (e) { return []; }
    };
  } catch (e) {}
  try {
    var kb = navigator.keyboard;
    if (kb && kb.getLayoutMap) {
      var glm = kb.getLayoutMap;
      kb.getLayoutMap = function () {
        try { return glm.call(kb); } catch (e) { return Promise.reject(e); }
      };
    }
  } catch (e) {}
  try {
    var vib = navigator.vibrate;
    if (vib) navigator.vibrate = function () { try { return vib.apply(navigator, arguments); } catch (e) { return false; } };
  } catch (e) {}

  // Из file:// модуль AudioWorklet не загружается (у страницы нет «origin»),
  // и звука не было бы совсем. Прячем AudioWorklet — звук пойдёт старым путём.
  if (location.protocol === "file:") {
    [window.BaseAudioContext, window.AudioContext, window.webkitAudioContext].forEach(function (C) {
      try {
        if (C && C.prototype && "audioWorklet" in C.prototype)
          Object.defineProperty(C.prototype, "audioWorklet", { get: function () { return undefined; }, configurable: true });
      } catch (e) {}
    });
  }

  // На телефоне: первое касание разворачивает игру на весь экран
  // и поворачивает горизонтально (Android). iPhone так не умеет — там
  // помогает «На экран Домой».
  var coarse = window.matchMedia && matchMedia("(pointer: coarse)").matches;
  var standalone = window.matchMedia && (matchMedia("(display-mode: fullscreen)").matches || matchMedia("(display-mode: standalone)").matches);
  if (coarse && !standalone) {
    var goFull = function () {
      window.removeEventListener("pointerup", goFull, true);
      window.removeEventListener("touchend", goFull, true);
      var d = document.documentElement, req = d.requestFullscreen || d.webkitRequestFullscreen;
      if (!req || document.fullscreenElement || document.webkitFullscreenElement) return;
      try {
        var p = req.call(d, { navigationUI: "hide" });
        if (p && p.then) p.then(function () {
          if (screen.orientation && screen.orientation.lock) screen.orientation.lock("landscape").catch(function () {});
        }).catch(function () {});
      } catch (e) {}
    };
    window.addEventListener("pointerup", goFull, true);
    window.addEventListener("touchend", goFull, true);
  }
  $("rot").addEventListener("click", function () { document.body.classList.add("okp"); });
  // Никакой прокрутки, масштабирования и контекстного меню поверх игры.
  ["gesturestart", "dblclick", "contextmenu"].forEach(function (e) {
    document.addEventListener(e, function (ev) { ev.preventDefault(); }, { passive: false });
  });
  document.addEventListener("touchmove", function (ev) { ev.preventDefault(); }, { passive: false });

  (function () {
    if (typeof WebAssembly !== "object" || typeof Promise !== "function")
      return Promise.reject({ ui: "Этот браузер слишком старый: в нём нет WebAssembly.<br>Откройте игру в свежем Chrome, Safari, Firefox, Edge или Яндекс Браузере." });
    var gl = null;
    try { gl = document.createElement("canvas").getContext("webgl2"); } catch (e) {}
    if (!gl)
      return Promise.reject({ ui: "Браузеру недоступен WebGL 2, без него игра не рисуется.<br>" +
        "Обновите браузер, включите «аппаратное ускорение» в его настройках или откройте игру в другом браузере. На iPhone и iPad нужна iOS 15 или новее." });
    var ext = gl.getExtension("WEBGL_lose_context"); if (ext) ext.loseContext();
    step("Распаковка…", 10);
    return frame().then(function () {
      var el = $("w"), size = +el.getAttribute("data-size");
      var bin = unbase64(el.textContent);
      el.parentNode.removeChild(el); // освобождаем память — важно на iPhone
      step("Распаковка…", 35);
      return unpack(bin, size);
    }).then(function (raw) {
      step("Запуск…", 75);
      var go = new Go();
      return WebAssembly.instantiate(raw, go.importObject).then(function (r) {
        step("Запуск…", 100);
        return frame().then(function () {
          box.parentNode.removeChild(box);
          document.body.classList.add("run");
          var crashed = false;
          go.exit = function (code) {
            if (code !== 0 && !crashed) {
              crashed = true;
              fail("Игра неожиданно остановилась. Нажмите «Перезапустить» — прогресс сохранён.", log.join("\n"));
            }
          };
          go.run(r.instance);
        });
      });
    });
  })().catch(function (e) {
    if (e && e.ui) return fail(e.ui);
    var t = String(e && e.message || e);
    if (/memory|Memory|RangeError/.test(t))
      return fail("Не хватило памяти, чтобы запустить игру. Закройте другие вкладки и приложения и попробуйте снова.", t);
    fail("Не удалось запустить игру.", t + "\n" + log.join("\n"));
  });
})();

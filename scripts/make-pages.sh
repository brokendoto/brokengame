#!/bin/sh
# Сайт для GitHub Pages: одна страница с игрой + манифест для установки
# на главный экран телефона. Вызывается из .github/workflows/pages.yml.
#   ./scripts/make-pages.sh dist/web dist/site
set -e
cd "$(dirname "$0")/.."
src="$1"; out="$2"
mkdir -p "$out"
python3 scripts/make-html.py "$src" "$out/index.html" --pages
cp internal/assets/icon_256.png "$out/icon-256.png"
cat > "$out/manifest.webmanifest" <<'JSON'
{
  "name": "Пятимирье",
  "short_name": "Пятимирье",
  "lang": "ru",
  "start_url": "./",
  "display": "fullscreen",
  "orientation": "landscape",
  "background_color": "#0b0a12",
  "theme_color": "#0b0a12",
  "icons": [{ "src": "icon-256.png", "sizes": "256x256", "type": "image/png", "purpose": "any maskable" }]
}
JSON
touch "$out/.nojekyll"
echo "Сайт готов: $out"

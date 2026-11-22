#!/bin/sh
# Сборка для Steam: Windows и Linux (Steam Deck) с тегом steam.
# Результат кладётся в steam/content/{windows,linux}, откуда его забирает
# SteamPipe (steamcmd +run_app_build steam/scripts/app_build.vdf).
#
#   STEAM_APPID=1234560 STEAM_SDK=~/steamworks_sdk ./scripts/build-steam.sh
#
# STEAM_SDK нужен только для Windows: из него копируется steam_api64.dll.
# Linux-сборке нужны библиотеки X11/GL/ALSA (см. README) — её лучше делать
# на Linux или в CI.
set -e
cd "$(dirname "$0")/.."
APPID="${STEAM_APPID:-480}"
LD="-s -w -X github.com/brokendoto/brokengame/internal/platform.SteamAppID=$APPID"
OUT=steam/content
rm -rf "$OUT" && mkdir -p "$OUT/windows" "$OUT/linux"

go test ./internal/...

GOOS=windows GOARCH=amd64 go build -tags steam -trimpath -ldflags "$LD -H=windowsgui" \
	-o "$OUT/windows/Pyatimirye.exe" ./cmd/pyatimirye
if [ -n "$STEAM_SDK" ]; then
	cp "$STEAM_SDK/redistributable_bin/win64/steam_api64.dll" "$OUT/windows/"
else
	echo "Внимание: STEAM_SDK не задан, steam_api64.dll не скопирована"
fi

if [ "$(uname -s)" = Linux ]; then
	GOOS=linux GOARCH=amd64 go build -tags steam -trimpath -ldflags "$LD" \
		-o "$OUT/linux/pyatimirye" ./cmd/pyatimirye
else
	echo "Linux-сборку пропускаю: собирайте на Linux или берите из CI"
fi
echo "Готово: $OUT (AppID $APPID)"

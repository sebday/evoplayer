#!/usr/bin/env bash
# Measure evoplayer daemon RSS after play/skip and large current-playlist save.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exe="${root}/.build/evoplayer"
if [[ ! -x "${exe}" ]]; then
  echo "build evoplayer first: go build -o .build/evoplayer ./cmd/evoplayer"
  exit 1
fi

runtime="$(mktemp -d)"
state="${runtime}/state"
cache="${runtime}/cache"
socket="${runtime}/evoplayer.sock"
export XDG_RUNTIME_DIR="${runtime}"
export XDG_STATE_HOME="${state}"
export XDG_CACHE_HOME="${cache}"
export EVOPLAYER_SOCKET="${socket}"

playlist_dir="${state}/evoplayer/playlists"
mkdir -p "${playlist_dir}"

"${exe}" serve >/dev/null 2>&1 &
pid=$!
cleanup() { kill "${pid}" 2>/dev/null || true; rm -rf "${runtime}"; }
trap cleanup EXIT

deadline=$((SECONDS + 10))
while ! [[ -S "${socket}" ]] && (( SECONDS < deadline )); do sleep 0.05; done

rss_kb() { ps -o rss= -p "${pid}" 2>/dev/null | tr -d ' ' || echo 0; }

ipc() {
  python3 -c "
import json, socket, sys
sock = sys.argv[1]
method = sys.argv[2]
params = json.loads(sys.argv[3]) if len(sys.argv) > 3 and sys.argv[3] else None
s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
s.connect(sock)
payload = {'id': 1, 'method': method}
if params is not None:
    payload['params'] = params
s.sendall((json.dumps(payload) + '\n').encode())
buf = b''
while b'\n' not in buf:
    buf += s.recv(65536)
s.close()
print(buf.decode().strip())
" "${socket}" "$@"
}

wav="$(mktemp --suffix=.wav)"
ffmpeg -y -loglevel error -f lavfi -i "sine=frequency=440:duration=8" -ar 48000 -ac 2 "${wav}" 2>/dev/null

paths_json='['
for i in $(seq 1 200); do
  paths_json+='"'"${wav}"'",'
done
paths_json="${paths_json%,}]"

echo "rss_kb_start=$(rss_kb)"

ipc queue.replace "{\"paths\":[\"${wav}\"],\"start_path\":\"${wav}\"}"
sleep 1
echo "rss_kb_playing=$(rss_kb)"

for _ in $(seq 1 20); do
  ipc playback.next "{}"
  sleep 0.15
done
echo "rss_kb_after_skip=$(rss_kb)"

ipc library.current.save "{\"paths\":${paths_json}}"
echo "rss_kb_after_save=$(rss_kb)"

echo "done pid=${pid}"

import asyncio
import json
import os
import sys
import threading
import time

out = os.fdopen(os.dup(1), "w", buffering=1)
os.dup2(2, 1)
sys.stdout = sys.stderr

import structlog

structlog.configure(
    processors=[structlog.processors.add_log_level, structlog.dev.ConsoleRenderer(colors=False)],
    logger_factory=structlog.PrintLoggerFactory(file=sys.stderr),
    wrapper_class=structlog.make_filtering_bound_logger(20),
)

from gamdl.api import AppleMusicApi
from gamdl.downloader import (
    AppleMusicBaseDownloader,
    AppleMusicDownloader,
    AppleMusicMusicVideoDownloader,
    AppleMusicSongDownloader,
    AppleMusicUploadedVideoDownloader,
)
from gamdl.interface import (
    AppleMusicBaseInterface,
    AppleMusicInterface,
    AppleMusicMusicVideoInterface,
    AppleMusicSongInterface,
    AppleMusicUploadedVideoInterface,
)
from gamdl.interface.enums import SongCodec


def reply(obj):
    out.write(json.dumps(obj) + "\n")
    out.flush()


def log(msg):
    sys.stderr.write(f"gamdl worker: {msg}\n")
    sys.stderr.flush()


def read_requests(loop, queue):
    buf = b""
    while True:
        chunk = os.read(0, 65536)
        if not chunk:
            loop.call_soon_threadsafe(queue.put_nowait, None)
            return
        buf += chunk
        while b"\n" in buf:
            line, buf = buf.split(b"\n", 1)
            if line.strip():
                loop.call_soon_threadsafe(queue.put_nowait, line.decode())


async def handle(interface, temp_path, req, sem):
    async with sem:
        started = time.monotonic()
        log(f"start {req['id']} {req['url']}")
        try:
            base = AppleMusicBaseDownloader(
                interface=interface,
                output_path=req["output_path"],
                temp_path=temp_path,
                silent=True,
            )
            downloader = AppleMusicDownloader(
                song=AppleMusicSongDownloader(base=base),
                music_video=AppleMusicMusicVideoDownloader(base=base),
                uploaded_video=AppleMusicUploadedVideoDownloader(base=base),
            )
            paths = []
            async for item in downloader.get_download_item_from_url(req["url"]):
                await downloader.download(item)
                if item.final_path:
                    paths.append(item.final_path)
            log(f"done {req['id']} in {time.monotonic() - started:.1f}s -> {paths}")
            reply({"id": req["id"], "ok": True, "paths": paths})
        except Exception as e:
            log(f"error {req['id']} after {time.monotonic() - started:.1f}s: {type(e).__name__}: {e}")
            reply({"id": req["id"], "ok": False, "error": f"{type(e).__name__}: {e}"})


async def main():
    cookies_path = sys.argv[1]
    codec = sys.argv[2] if len(sys.argv) > 2 and sys.argv[2] else "aac-web"
    temp_path = sys.argv[3] if len(sys.argv) > 3 and sys.argv[3] else "."
    workers = int(sys.argv[4]) if len(sys.argv) > 4 else 3

    try:
        api = await AppleMusicApi.create_from_netscape_cookies(cookies_path=cookies_path)
        if not api.active_subscription:
            reply({"event": "fatal", "error": "no active apple music subscription"})
            return
        base_interface = await AppleMusicBaseInterface.create(apple_music_api=api)
        priority = [SongCodec(c.strip()) for c in codec.split(",") if c.strip()]
        interface = AppleMusicInterface(
            song=AppleMusicSongInterface(base=base_interface, codec_priority=priority),
            music_video=AppleMusicMusicVideoInterface(base=base_interface),
            uploaded_video=AppleMusicUploadedVideoInterface(base=base_interface),
        )
    except Exception as e:
        reply({"event": "fatal", "error": f"{type(e).__name__}: {e}"})
        return

    reply({"event": "ready"})
    sem = asyncio.Semaphore(workers)
    loop = asyncio.get_running_loop()
    requests = asyncio.Queue()
    threading.Thread(target=read_requests, args=(loop, requests), daemon=True).start()
    tasks = set()
    while True:
        line = await requests.get()
        if line is None:
            break
        try:
            req = json.loads(line)
        except Exception:
            continue
        task = asyncio.create_task(handle(interface, temp_path, req, sem))
        tasks.add(task)
        task.add_done_callback(tasks.discard)
    if tasks:
        await asyncio.gather(*tasks, return_exceptions=True)


asyncio.run(main())

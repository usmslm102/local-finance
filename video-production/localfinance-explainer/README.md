# LocalFinance / Your money, made clear

60 seconds · 1080×1920 · 30fps · Remotion 4.0.538.

Original project work is MIT; fonts, speech models and optional rendering tools retain their own terms. See [third-party notices](THIRD_PARTY_NOTICES.md), including Remotion’s separate license.

The shared creative contract is `production/timeline.json`. Read it with the script, storyboard, claims ledger, art direction and audio plan. The illustrations reconstruct verified product capabilities using fictional data; they are not screen recordings.

## Preview and export

From `remotion/`, use Node 24 and pnpm 11:

```sh
pnpm install --frozen-lockfile
pnpm run studio
pnpm run typecheck
pnpm run preview
pnpm run render
# After inspecting the master:
pnpm run sync-site
```

Studio uses port 3000; rendering uses port 3004. On macOS, the config prefers an already-installed Google Chrome because downloaded Headless Shell 157 timed out on the production host. Set `REMOTION_BROWSER_EXECUTABLE` to override the browser. Other hosts retain Remotion's default browser. Webpack is enabled for compatibility.

The export script renders three 600-frame picture segments at concurrency 2, then joins them and adds the continuous mastered soundtrack with FFmpeg. This avoids a browser stall seen in a longer full-resolution run, with no audio seams.

The final export is `renders/localfinance-explainer.mp4`. The homepage copy, poster and WebVTT live in `../../docs/media/`. After a creative revision, render and inspect the new master before replacing the homepage copy. Keep captions, transcript and poster synchronized. All playback assets are self-hosted; the site needs no Node/Remotion runtime.

## Soundtrack

All final lossless FLAC stems and the final mix are supplied under `remotion/public/`, so rendering requires no TTS service or model download. Narration uses local Kokoro v1.0 `af_heart` through kokoro-onnx 0.4.9 (Apache-2.0 model, MIT engine). The score and data pings are original procedural synthesis. Font license is bundled beside Inter. See `production/asset-manifest.json`.

To regenerate audio, create an isolated Python environment with `kokoro-onnx==0.4.9`, `numpy` and `soundfile`. Download `kokoro-v1.0.int8.onnx` and `voices-v1.0.bin` from the [official model release](https://github.com/thewh1teagle/kokoro-onnx/releases/tag/model-files-v1.1) into a local model directory, then run:

```sh
LOCALFINANCE_TTS_DIR=/absolute/path/to/models python scripts/produce-audio.py
```

This updates measured narration windows, phrase captions, SRT, WebVTT and the mix. It makes no service calls. `prepare.py` is the initial preproduction generator; do not run it over an edited/locked timeline because it resets measured timing and asset readiness.

## Verify

With Pillow installed, `python scripts/inspect-export.py` validates the master and makes exact-frame contact sheets. `qa-report.md` records the actual inspected output and refinements. Full decoding and audio loudness can also be checked with FFmpeg.

For another renderer: read `production/timeline.json` and the referenced production docs; implement faithfully with the chosen engine, render and QA. Preserve the creative concept and 1800-frame duration.

The video workspace pins Zod 4.6.6 to address GHSA-mrhx-c7xc-m3w5 in the renderer’s transitive dependency graph. `pnpm audit --audit-level=high` and `pnpm run typecheck` verify that isolated toolchain.

# QA report

Verified 2026-10-25. Final master: `renders/localfinance-explainer.mp4`; identical homepage copy: `../../docs/media/localfinance-explainer.mp4`.

- 60.000000 seconds, 1080×1920, 30fps, exactly 1800 frames.
- H.264 High, yuv420p limited range; stereo AAC-LC, 48kHz. Fast-start MP4; 4,916,373 bytes.
- Final encoded audio measured with FFmpeg ebur128: -16.4 LUFS integrated, 3.6 LU loudness range, -4.4 dBFS true peak. No clipping. Narration ends at 57.430s; final brand hold continues to 60s.
- Blueprint validation: PASS, strict asset readiness, zero warnings. Remotion TypeScript check: PASS. Script syntax checks and `git diff --check`: PASS.
- Full FFmpeg decode: all 1800 frames, no decode errors. Master and homepage MP4 SHA-256 match: `05f1445b170d1fdc30098a438b3a51dd678efc4ac947d222d6cb9800106dfa3b`.

## Visual inspection and refinements

First rendered 540×960 preview and inspected all seven scene heroes. Found the spending budget footer crowded the illustrative-data label; moved that scene's content up and tightened spacing. Simplified the organize narration after measuring speech. Shortened the closing line to create a 2.57-second final hold. Replaced temporary system-voice narration with local Kokoro af_heart before the final export.

Inspected 39 exact final-export frames through contact sheets, including the opening 0–2 seconds, every scene boundary at -2/0/+2/+30 frames, each scene's resolved content, high-motion entries, and final frames 1740, 1770, 1798 and 1799. Evidence: `production/previews/final-scenes.jpg`, `final-boundaries-1.jpg`, `final-boundaries-2.jpg`, `final-boundaries-3.jpg`, and reproducible individual frames in the ignored `production/previews/final/` directory. No cropped headlines, caption collisions, missing assets, budget/footer overlap or blank tail remained. Brief chapter-entry fades are intentional. Frame 0 shows the persistent brand, followed by the opening reveal.

Played the updated composition in Studio through a full loop, and verified the actual finished MP4 in the homepage's native player. Browser reported duration 60, readyState 4, 1080×1920 video, paused false and currentTime advancing. No media error. Desktop page has no horizontal overflow. Screenshot: `production/previews/homepage-desktop.png`. Responsive CSS stacks the player below the introduction on smaller screens; a separate mobile-device playback pass was not performed.

Captions are short phrase captions, distributed within measured scene-local speech windows, with matching JSON/SRT/WebVTT. They are not word-level forced alignments. No subjective speaker/headphone listening pass was available; synchronization was checked through measured audio duration, scheduled starts, visible captions, playback and stream metadata.

## Renderer adaptations

Remotion 4.0.538 with local Chrome and Webpack. Downloaded Chrome Headless Shell 157 timed out loading the root on this host; an installed Chrome worked. A long 1080p render later stalled at frame 1279. The final reproducible export renders three 600-frame picture segments at concurrency 2, joins them, normalizes JPEG-derived full-range picture to limited-range yuv420p, and muxes the continuous mastered soundtrack once. This preserves all timeline frames and avoids AAC seams. No creative timing changes were made for the renderer.

The film uses original graphics rather than captured app screens and labels fictional data. Existing product mark and Inter font are reused, with font license included. Final narration uses Apache-2.0 Kokoro weights / MIT kokoro-onnx. Music and SFX are original procedural synthesis. Source-backed product claims and compatibility qualifications are recorded in `production/research.md`.

## Homepage integration and reproduction

The existing `docs/index.html` hero now includes a native self-hosted player, poster, optional English caption track, readable transcript and MP4 download. No autoplay, external player, remote media or player dependency. The video uses `preload="none"`; its poster is approximately 93KiB. Existing static Pages workflow includes these assets automatically when the changes are published. No deployment was performed during this task.

From `remotion/`: `pnpm install --frozen-lockfile`, `pnpm run typecheck`, `pnpm run render`. Inspect with `python ../scripts/inspect-export.py`, then run `pnpm run sync-site`. See the project README for Studio and optional local audio regeneration. Supplied audio files make ordinary rerenders independent of TTS or network services. The canonical committed MP4 is the homepage copy; duplicate masters, probe reports and individual review frames are ignored.


## Public-repository review

Reviewed the exact proposed files before the first public push with Gitleaks 8.30.1: no secret findings. Additional checks found no private home/volume paths, database files, credential files, downloaded model binaries, installed dependency directories or image EXIF in the staged set. The only financial examples are fictional. Temporary recordings, raw WAV intermediates, duplicate MP4s and probe metadata are ignored.

Distributed audio stems use lossless FLAC. Decoding each stem and the master mix produces the same PCM MD5 as its corresponding production WAV. A 60-frame Remotion render verifies the distributed audio path. The final homepage MP4 remains byte-for-byte unchanged.

The isolated video dependency audit initially found GHSA-mrhx-c7xc-m3w5 in transitive Zod. A documented override pins Zod 4.6.6; the repeated audit reports zero known vulnerabilities. Typecheck and a fresh short render pass with the patched dependency. Third-party notices distinguish the repository's MIT work, Inter's SIL OFL, Kokoro's Apache-2.0 model, the MIT synthesis engine, and Remotion's separate non-OSI license. No model or renderer binaries are distributed.

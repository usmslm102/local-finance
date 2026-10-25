# Third-party notices and asset provenance

The original LocalFinance video composition, script, illustrations, procedural music, sound effects and supporting scripts are covered by the repository's [MIT license](../../LICENSE). Third-party materials and tools retain their respective licenses; this document does not relicense them.

| Material or tool | Use | License / source |
|---|---|---|
| LocalFinance shield mark | Existing repository artwork reused in film and site | Repository MIT; `docs/mark.svg` |
| Inter variable font | Unmodified bundled font in the editable project and static site | SIL Open Font License 1.1; full copyright and terms in `remotion/public/inter-LICENSE.txt` and `docs/inter-LICENSE.txt`; [Inter](https://github.com/rsms/inter) |
| Kokoro v1.0, voice `af_heart` | Local synthesis of the original narration script | Model weights Apache-2.0; [official model card](https://huggingface.co/hexgrad/Kokoro-82M) and [voice list](https://huggingface.co/hexgrad/Kokoro-82M/blob/main/VOICES.md) |
| kokoro-onnx 0.4.9 | Optional local narration generation | MIT; full notice in `licenses/kokoro-onnx-LICENSE.txt`; [upstream](https://github.com/thewh1teagle/kokoro-onnx) |
| Remotion 4.0.538 and its renderer packages | Optional development tooling for editing and rendering | Remotion License, **not MIT or an OSI open-source license**; full terms from the installed release in `licenses/remotion-LICENSE.md`; [upstream terms](https://www.remotion.dev/license) |
| React, TypeScript and other package dependencies | Installed by pnpm for the optional renderer | Their upstream licenses apply; exact versions and integrity hashes are pinned in `remotion/pnpm-lock.yaml`. No installed package sources are vendored. |
| FFmpeg and Chrome | External local production tools | Separately installed tools under their own distribution terms. Their binaries are not included. |

Remotion's free license permits individual use, eligible small for-profit organizations and non-profit organizations to create videos, including commercially. Other organizations need its company license. Publishing this MIT composition does not grant a license to Remotion itself. The LocalFinance application and static site's video playback do not depend on Remotion at runtime.

No model weights, voice embeddings, downloaded browsers, private statements, real account records, temporary system-voice recordings or third-party stock recordings are distributed. The voice and final mix are generated audio, not recordings of the project owner or a purported endorsement. All amounts, merchants, categories and notes shown are illustrative fictional examples. The original score and SFX are synthesized from the supplied code.

The optional narration-generation model files came from the [kokoro-onnx model-files-v1.1 release](https://github.com/thewh1teagle/kokoro-onnx/releases/tag/model-files-v1.1). Their SHA-256 digests, for reproducibility:

```text
ae315a79b623f244700e4afb9246c46a26066782e049ba174bf3ba433970ee9c  kokoro-v1.0.int8.onnx
bca610b8308e8d99f32e6fe4197e7ec01679264efed0cac9140fe9c29f1fbf7d  voices-v1.0.bin
```

The published MP4 is an ordinary media export. All runtime media is self-hosted; no third-party player, telemetry or remote narration service is used by the site.

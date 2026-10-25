// Render shorter silent picture segments to avoid long-lived browser stalls,
// then mux the continuous mastered soundtrack once (no AAC seams).
import {spawnSync} from 'node:child_process';
import {mkdirSync, writeFileSync} from 'node:fs';
import {fileURLToPath} from 'node:url';
import path from 'node:path';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
const cwd=path.join(root,'remotion');
const chunks=path.join(root,'renders/chunks');
mkdirSync(chunks,{recursive:true});
function run(cmd,args) {
  const result=spawnSync(cmd,args,{cwd,stdio:'inherit',env:process.env});
  if(result.error) throw result.error;
  if(result.status!==0) process.exit(result.status||1);
}
for(let i=0; !process.argv.includes('--assemble-only') && i<3; i++) {
  run('pnpm',['exec','remotion','render','src/index.ts','LocalFinance',path.join(chunks,`${i}.mp4`),
    `--frames=${i*600}-${(i+1)*600-1}`,'--muted','--codec=h264','--pixel-format=yuv420p','--crf=19','--concurrency=2','--timeout=120000','--port=3004']);
}
writeFileSync(path.join(chunks,'concat.txt'),"file '0.mp4'\nfile '1.mp4'\nfile '2.mp4'\n");
run('ffmpeg',['-v','error','-y','-f','concat','-safe','0','-i',path.join(chunks,'concat.txt'),
  '-i',path.join(root,'remotion/public/mix.flac'),'-map','0:v:0','-map','1:a:0','-vf','scale=in_range=full:out_range=tv,format=yuv420p','-c:v','libx264','-preset','fast','-crf','18','-pix_fmt','yuv420p','-color_range','tv',
  '-c:a','aac','-b:a','192k','-t','60','-movflags','+faststart',path.join(root,'renders/localfinance-explainer.mp4')]);

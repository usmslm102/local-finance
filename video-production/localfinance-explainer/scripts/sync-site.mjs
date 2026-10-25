// Copy a verified local export into the existing static site. No deployment.
import {spawnSync} from 'node:child_process';
import {copyFileSync, mkdirSync} from 'node:fs';
import {fileURLToPath} from 'node:url';
import path from 'node:path';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
const master=path.join(root,'renders/localfinance-explainer.mp4');
const result=spawnSync('ffprobe',['-v','error','-show_streams','-show_format','-of','json',master],{encoding:'utf8'});
if(result.status!==0) throw new Error(result.stderr);
const info=JSON.parse(result.stdout);
const video=info.streams.find(x=>x.codec_type==='video');
if(video.width!==1080||video.height!==1920||video.nb_frames!=='1800'||video.r_frame_rate!=='30/1'||Math.abs(Number(info.format.duration)-60)>.05) throw new Error('Export does not match the locked production format');
const dest=path.resolve(root,'../../docs/media');
mkdirSync(dest,{recursive:true});
const poster=spawnSync('ffmpeg',['-v','error','-y','-ss','12.7','-i',master,'-frames:v','1','-q:v','2',path.join(dest,'localfinance-explainer-poster.jpg')],{stdio:'inherit'});
if(poster.status!==0) process.exit(poster.status||1);
copyFileSync(master,path.join(dest,'localfinance-explainer.mp4'));
copyFileSync(path.join(root,'renders/localfinance-explainer.vtt'),path.join(dest,'localfinance-explainer.vtt'));
console.log(`Verified video, poster and captions copied to ${dest}`);

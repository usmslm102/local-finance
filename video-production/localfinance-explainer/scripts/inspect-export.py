"""Probe/decode a rendered export and generate labeled visual review sheets."""
import json
import subprocess
import sys
from pathlib import Path
from PIL import Image, ImageDraw

ROOT=Path(__file__).resolve().parents[1]
movie=Path(sys.argv[1]) if len(sys.argv)>1 else ROOT/'renders/localfinance-explainer.mp4'
probe=json.loads(subprocess.check_output(['ffprobe','-v','error','-show_format','-show_streams','-of','json',str(movie)]))
video=next(s for s in probe['streams'] if s['codec_type']=='video')
audio=next(s for s in probe['streams'] if s['codec_type']=='audio')
assert (video['width'],video['height'])==(1080,1920)
assert video['r_frame_rate']=='30/1'
assert video['nb_frames']=='1800'
assert video['codec_name']=='h264' and video['pix_fmt']=='yuv420p'
assert abs(float(probe['format']['duration'])-60)<.05
assert audio['codec_name']=='aac' and audio['channels']==2
subprocess.run(['ffmpeg','-v','error','-i',str(movie),'-f','null','-'],check=True)
probe['format']['filename']='renders/localfinance-explainer.mp4'
(ROOT/'renders/ffprobe.json').write_text(json.dumps(probe,indent=2)+'\n')
timeline=json.loads((ROOT/'production/timeline.json').read_text())
hero=[60,285,570,900,1220,1530,1770]
boundaries=[0,2,15,30,59]
for scene in timeline['scenes']:
    a,b=scene['start_frame'],scene['end_frame']
    boundaries.extend([a,a+2,a+30,b-2])
boundaries.extend([1740,1799])
frames=sorted(set(hero+boundaries))
P=ROOT/'production/previews/final'
P.mkdir(parents=True,exist_ok=True)
for frame in frames:
    subprocess.run(['ffmpeg','-v','error','-y','-ss',str(frame/30),'-i',str(movie),'-frames:v','1','-vf','scale=270:480',str(P/f'{frame:04}.jpg')],check=True)

def sheet(frames,path):
    rows=(len(frames)+3)//4
    out=Image.new('RGB',(1080,rows*510),'#dfe6df');draw=ImageDraw.Draw(out)
    for i,frame in enumerate(frames):
        x=(i%4)*270;y=(i//4)*510
        out.paste(Image.open(P/f'{frame:04}.jpg'),(x,y))
        draw.text((x+12,y+487),f'{frame:04} / {frame/30:.3f}s',fill='#07110d')
    out.save(path)

sheet(hero,ROOT/'production/previews/final-scenes.jpg')
for i in range(0,len(boundaries),16):
    sheet(boundaries[i:i+16],ROOT/f'production/previews/final-boundaries-{i//16+1}.jpg')
print(json.dumps(dict(duration=probe['format']['duration'],width=video['width'],height=video['height'],fps=video['r_frame_rate'],frames=video['nb_frames'],audio=audio['codec_name'],channels=audio['channels'],inspected_frames=frames),indent=2))

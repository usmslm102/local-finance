"""Local narration, original score, measured captions; requires numpy and ffmpeg."""
import os
import json
import math
import re
import subprocess
import wave
from pathlib import Path
import numpy as np
import soundfile as sf
from kokoro_onnx import Kokoro
import onnxruntime as ort
ort.disable_telemetry_events()

ROOT = Path(__file__).resolve().parents[1]
PUBLIC = ROOT / 'remotion/public'
P = ROOT / 'production'
timeline = json.loads((P / 'timeline.json').read_text())
SR = 48000
length = SR * 60
voice = np.zeros(length, dtype=np.float64)
captions = []
model_dir = Path(os.environ.get('LOCALFINANCE_TTS_DIR', str(ROOT / 'models')))
kokoro = Kokoro(str(model_dir / 'kokoro-v1.0.int8.onnx'), str(model_dir / 'voices-v1.0.bin'))

class TypedSpeedSession:
    """kokoro-onnx 0.4.9 assumes int speed for newer exports; v1.1 expects float."""
    def __init__(self, session): self.session=session
    def get_inputs(self): return self.session.get_inputs()
    def run(self, names, inputs):
        speed_type=next(x.type for x in self.get_inputs() if x.name=='speed')
        inputs['speed']=np.asarray([1.04],dtype=np.float32 if speed_type=='tensor(float)' else np.int32)
        return self.session.run(names,inputs)

kokoro.sess=TypedSpeedSession(kokoro.sess)


def run(*args):
    return subprocess.check_output(args).decode().strip()

def wav(path, samples):
    with wave.open(str(path), 'wb') as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(SR)
        w.writeframes((np.clip(samples, -1, 1) * 32767).astype('<i2').tobytes())

for seg, scene in zip(timeline['narration']['segments'], timeline['scenes']):
    out = PUBLIC / (seg['id'] + '.wav')
    generated, sample_rate = kokoro.create(seg['text'], voice='af_heart', speed=1.04, lang='en-us')
    sf.write(str(PUBLIC / 'tts-temp.wav'), generated, sample_rate)
    run('ffmpeg', '-v', 'error', '-y', '-i', str(PUBLIC/'tts-temp.wav'), '-ar', str(SR), '-ac', '1', str(out))
    with wave.open(str(out), 'rb') as w:
        samples = np.frombuffer(w.readframes(w.getnframes()), dtype='<i2').astype(float) / 32768
    # Remove only leading/trailing silence; preserve spoken cadence.
    active = np.flatnonzero(abs(samples) > .003)
    if not len(active):
        raise RuntimeError('Speech synthesis produced no speech')
    samples = samples[max(0, active[0]-2400):min(len(samples),active[-1]+4800)]
    budget = (scene['end_frame'] - seg['start_frame'] - 10) / 30
    duration = len(samples) / SR
    if duration > budget:
        # Bounded tempo correction keeps the locked 60s narrative and natural pitch.
        ratio = duration / budget
        if ratio > 1.20:
            raise RuntimeError(f'{seg["id"]} needs script adjustment: {duration:.2f}s > {budget:.2f}s')
        wav(PUBLIC / 'tempo-input.wav', samples)
        run('ffmpeg', '-v', 'error', '-y', '-i', str(PUBLIC/'tempo-input.wav'), '-af', f'atempo={ratio:.6f}', str(out))
        with wave.open(str(out), 'rb') as w:
            samples=np.frombuffer(w.readframes(w.getnframes()),dtype='<i2').astype(float)/32768
    duration = len(samples)/SR
    wav(out,samples)
    start = round(seg['start_frame'] / 30 * SR)
    voice[start:start+len(samples)] += samples
    seg['end_frame'] = min(scene['end_frame']-1, seg['start_frame']+math.ceil(duration*30))
    seg['measured_duration_seconds'] = round(duration, 3)
    words = seg['text'].split()
    # Phrase captions: punctuation preferred, maximum 7 words per phrase.
    phrases=[]
    chunk=[]
    for word in words:
        chunk.append(word)
        if len(chunk)>=7 or (len(chunk)>=3 and re.search(r'[.,?!]$',word)):
            phrases.append(' '.join(chunk));chunk=[]
    if chunk: phrases.append(' '.join(chunk))
    cursor = seg['start_frame'] / 30 * 1000
    total = sum(len(x) for x in phrases)
    for phrase in phrases:
        end = cursor + duration * 1000 * len(phrase)/total
        captions.append(dict(text=phrase,startMs=round(cursor),endMs=round(end),timestampMs=None,confidence=None,pageBreakAfter=True))
        cursor=end
    print(seg['id'], duration, 'seconds')

wav(PUBLIC/'narration.wav',voice)
t=np.arange(length)/SR
music=np.zeros(length)
chords=[(146.832,184.997,220.0),(123.471,146.832,184.997),(97.999,123.471,146.832),(110.0,138.591,164.814)]
for i in range(12):
    start=i*5; stop=(i+1)*5
    sel=(t>=start)&(t<stop)
    local=t[sel]-start
    env=np.minimum(local/.6,1)*np.minimum((5-local)/.8,1)
    for freq in chords[i%4]:
        music[sel]+=.009*env*(np.sin(2*np.pi*freq*local)+.18*np.sin(2*np.pi*2*freq*local))
for i in range(96):
    start=i*.625
    n=min(int(SR*.6),length-int(start*SR))
    local=np.arange(n)/SR
    freq=chords[(int(start)//5)%4][i%3]*4
    music[int(start*SR):int(start*SR)+n]+=.009*np.exp(-local*9)*np.sin(2*np.pi*freq*local)*np.minimum(local/.008,1)
music*=np.minimum(t/1.5,1)*np.minimum((60-t)/2,1)
wav(PUBLIC/'music.wav',music)
sfx=np.zeros(length)
for scene in timeline['scenes']:
    start=scene['start_frame']/30
    n=int(SR*.3);local=np.arange(n)/SR
    ping=.025*np.exp(-local*20)*np.sin(2*np.pi*(880*local+140*local*local))*np.minimum(local/.006,1)
    sfx[int(start*SR):int(start*SR)+n]+=ping
wav(PUBLIC/'sfx.wav',sfx)
mix=voice+music+sfx
wav(PUBLIC/'mix-raw.wav',mix)
run('ffmpeg','-v','error','-y','-i',str(PUBLIC/'mix-raw.wav'),'-af','loudnorm=I=-16:TP=-1.5:LRA=11','-ar',str(SR),'-ac','2',str(PUBLIC/'mix.wav'))
(P/'timeline.json').write_text(json.dumps(timeline,indent=2)+'\n')
(P/'captions.json').write_text(json.dumps(captions,indent=2)+'\n')

def stamp(ms,sep):
    ms=round(ms);h,ms=divmod(ms,3600000);m,ms=divmod(ms,60000);s,ms=divmod(ms,1000)
    return f'{h:02}:{m:02}:{s:02}{sep}{ms:03}'

for ext,sep in [('vtt','.'),('srt',',')]:
    head='WEBVTT\n\n' if ext=='vtt' else ''
    text=head+'\n\n'.join(f'{i+1}\n{stamp(c["startMs"],sep)} --> {stamp(c["endMs"],sep)}\n{c["text"]}' for i,c in enumerate(captions))+'\n'
    (ROOT/'renders'/('localfinance-explainer.'+ext)).write_text(text)
timeline['narration']['voice_profile']='Kokoro v1.0 af_heart, local ONNX synthesis, conversational English'
(P/'timeline.json').write_text(json.dumps(timeline,indent=2)+'\n')
manifest=json.loads((P/'asset-manifest.json').read_text())
for asset in manifest['assets']:
    if asset['type']=='audio':
        asset['status']='ready'
        if asset['id']=='voice-mix':
            asset['origin']='Local Kokoro v1.0 af_heart via kokoro-onnx 0.4.9'
            asset['license']='Kokoro model Apache-2.0; kokoro-onnx MIT; original script'
        if asset['id'] in ('music','sfx'): asset['license']='Original procedural synthesis authored for this project; repository MIT'
        if asset['id']=='mix': asset['origin']='Local Kokoro narration + original score and pings'
manifest['assets']=[a for a in manifest['assets'] if a['id']!='mix']
manifest['assets'].append(dict(id='mix',path='../remotion/public/mix.flac',type='audio',origin='Local Kokoro narration + original procedural score and pings',license='Original production; Kokoro Apache-2.0 synthesized narration',status='ready'))
(P/'asset-manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
(ROOT/'remotion/src/captions.json').write_text(json.dumps(captions,indent=2)+'\n')

# Remove disposable synthesis intermediates; retain lossless editable FLAC stems for distribution.
for path in PUBLIC.glob('*.aiff'): path.unlink()
for stem in ['mix','music','narration','sfx']:
    run('ffmpeg','-v','error','-y','-i',str(PUBLIC/(stem+'.wav')),'-compression_level','8',str(PUBLIC/(stem+'.flac')))
for asset in manifest['assets']:
    if asset['type']=='audio': asset['path']=asset['path'].replace('.wav','.flac')
(P/'asset-manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
for name in ['mix-raw.wav', 'tempo-input.wav', 'tts-temp.wav']: (PUBLIC/name).unlink(missing_ok=True)

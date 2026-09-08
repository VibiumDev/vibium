// Derive playback copies and captions from a completed capture. Requires ffmpeg/ffprobe.
import fs from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {execFileSync} from 'node:child_process';
const directory=path.resolve(process.argv[2]||process.env.MEETUP_CAPTURE_DIR||fileURLToPath(new URL('../assets',import.meta.url)));
const manifestPath=path.join(directory,'demo-results.json');
const manifest=JSON.parse(fs.readFileSync(manifestPath,'utf8'));
const stamp=seconds=>{const ms=Math.round(seconds*1000);return `${String(Math.floor(ms/60000)).padStart(2,'0')}:${String(Math.floor(ms/1000)%60).padStart(2,'0')}.${String(ms%1000).padStart(3,'0')}`;};
for(const run of manifest.runs){
  const {mode}=run;
  if(!['broken','fixed'].includes(mode))throw new Error('Unknown capture mode.');
  const file=path.join(directory,mode);
  const index=JSON.parse(execFileSync('unzip',['-p',file+'.zip','video/index.json'],{encoding:'utf8'}));
  const offset=index.videos[0].offsetMs;
  const packets=JSON.parse(execFileSync('ffprobe',['-v','error','-show_packets','-show_entries','packet=pts_time','-of','json',file+'.webm'],{encoding:'utf8',maxBuffer:16*1024*1024})).packets;
  const duration=Number(packets.at(-1).pts_time);
  const checkStart=(run.parents.find(p=>p.method==='vibium:check.run').startTime-offset)/1000;
  if(!(checkStart>0&&checkStart<duration))throw new Error('Check timing is outside the video.');
  execFileSync('ffmpeg',['-y','-v','error','-i',file+'.webm','-c:v','libx264','-crf','21','-pix_fmt','yuv420p','-movflags','+faststart',file+'.mp4']);
  fs.writeFileSync(file+'.vtt',`WEBVTT\n\n00:00.000 --> ${stamp(checkStart)}\nRun: add a battery pack. The page says Added to cart.\n\n${stamp(checkStart)} --> ${stamp(duration)}\nCheck: open the cart and inspect its contents.\n`);
  run.video={file:mode+'.webm',durationSeconds:Math.round(duration*1000)/1000,checkStartsSeconds:Math.round(checkStart*1000)/1000,recordingOffsetMs:offset};
  console.log(`${mode}: MP4 and captions derived from ${duration.toFixed(1)}s recording.`);
}
fs.writeFileSync(manifestPath,JSON.stringify(manifest,null,2)+'\n');

// Entry point: cache DOM, load data behind the sync loader, build both views, wire controls.
import { S, D, isPhone, cacheDom } from './state.js';
import { build } from './render.js';
import { buildMonthly } from './monthly.js';
import { applyZoom } from './zoom.js';
import { initControls } from './controls.js';

cacheDom();
if(isPhone) document.body.classList.add('phone');

const sleep=ms=>new Promise(r=>setTimeout(r,ms));
function setLoader(msg, sub, isErr){ D.lmsg.textContent=msg||''; D.lsub.textContent=sub||''; D.loader.classList.toggle('err',!!isErr); }
function hideLoader(){ D.loader.classList.add('hidden'); }
async function getJSON(u){ const r=await fetch(u,{cache:'no-store'}); if(!r.ok) throw new Error(u+' → HTTP '+r.status); return r.json(); }

async function init(){
  try{
    let st=await getJSON('/api/status');
    while(!st.ready){
      if(st.error) setLoader('Sync error', st.error, true);
      else setLoader(st.message||'Syncing with Audiobookshelf…', st.sessionsFetched?(st.sessionsFetched+' sessions'):'');
      await sleep(1500);
      st=await getJSON('/api/status');
    }
    setLoader('Building heatmap…','');
    const data=await getJSON('/api/data');
    build(data);
    buildMonthly();          // Level-0 view is cheap; build once up front
    initControls();
    applyZoom();
    hideLoader();
    requestAnimationFrame(()=>{ D.scroller.scrollLeft=D.scroller.scrollWidth; });   // start at "today"
  }catch(e){
    setLoader('Failed to load', String(e && e.message || e), true);
  }
}
init();

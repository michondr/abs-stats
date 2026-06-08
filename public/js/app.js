// Entry point: cache DOM, load data behind the sync loader, build both views, wire controls.
import { S, D, isPhone, cacheDom } from './state.js';
import { build } from './render.js';
import { buildMonthly } from './monthly.js';
import { applyZoom, scrollTodayToCenter } from './zoom.js';
import { initControls } from './controls.js';

cacheDom();
if(isPhone) document.body.classList.add('phone');

// Data source is overridable so the same frontend serves both the live app and the
// static GitHub Pages demo. Defaults match the live Go server; the demo build injects
// window.__STATUS_URL=null (skip the sync poll) and window.__DATA_URL='data.json'.
const STATUS_URL = ('__STATUS_URL' in window) ? window.__STATUS_URL : '/api/status';
const DATA_URL   = window.__DATA_URL || '/api/data';

const sleep=ms=>new Promise(r=>setTimeout(r,ms));
function setLoader(msg, sub, isErr){ D.lmsg.textContent=msg||''; D.lsub.textContent=sub||''; D.loader.classList.toggle('err',!!isErr); }
function hideLoader(){ D.loader.classList.add('hidden'); }
async function getJSON(u){ const r=await fetch(u,{cache:'no-store'}); if(!r.ok) throw new Error(u+' → HTTP '+r.status); return r.json(); }

async function init(){
  try{
    if(STATUS_URL){
      let st=await getJSON(STATUS_URL);
      while(!st.ready){
        if(st.error) setLoader('Sync error', st.error, true);
        else setLoader(st.message||'Syncing with Audiobookshelf…', st.sessionsFetched?(st.sessionsFetched+' sessions'):'');
        await sleep(1500);
        st=await getJSON(STATUS_URL);
      }
    }
    setLoader('Building heatmap…','');
    const data=await getJSON(DATA_URL);
    build(data);
    buildMonthly();          // Level-0 view is cheap; build once up front
    initControls();
    applyZoom();
    hideLoader();
    requestAnimationFrame(scrollTodayToCenter);   // center today so zoom-in anchors on it
  }catch(e){
    setLoader('Failed to load', String(e && e.message || e), true);
  }
}
init();

// Pure formatting/date helpers (no DOM, no state).

export const MONTH=['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];

export function hms(sec){
  const h=Math.floor(sec/3600),m=Math.floor((sec%3600)/60),s=sec%60;
  let o=''; if(h)o+=h+'h '; return o+String(m).padStart(2,'0')+'m '+String(s).padStart(2,'0')+'s';
}
export function md(d){return d.toLocaleDateString('en-US',{month:'short',day:'numeric'});}
export function parseDate(s){const p=String(s||'').split('-').map(Number);return new Date(p[0],(p[1]||1)-1,p[2]||1);}
export function keyOf(d){return d.getFullYear()+'-'+String(d.getMonth()+1).padStart(2,'0')+'-'+String(d.getDate()).padStart(2,'0');}
export function esc(s){return String(s==null?'':s).replace(/[&<>"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]));}

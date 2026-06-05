// LEVEL 0 — year × month overview. Built from the same /api/data; aggregates each calendar
// month's hours listened + books finished, coloured by intensity, with a per-year hours total.
import { S, D } from './state.js';
import { MONTH } from './format.js';

export function buildMonthly(){
  const data=S.data; if(!data) return;
  const days=data.days||{};

  // ym 'YYYY-M'(0-based month) -> {hrs, fin}
  const agg={};
  let maxHrs=0;
  for(const k in days){
    const p=k.split('-'); const ym=(+p[0])+'-'+(+p[1]-1);
    const a=agg[ym]||(agg[ym]={hrs:0, fin:0});
    for(const s of days[k]){ a.hrs+=(s.secs||0)/3600; if(s.fin) a.fin++; }
  }
  for(const ym in agg) if(agg[ym].hrs>maxHrs) maxHrs=agg[ym].hrs;

  const y0=S.start.getFullYear(), y1=S.today.getFullYear();
  const nyears=y1-y0+1;
  const startMs=S.start.getTime(), todayMs=S.today.getTime();
  const lvl=h=> h<=0?'' : (' l'+Math.min(4, Math.max(1, Math.ceil(h/maxHrs*4))));

  // Transposed: years across the top (columns), months Jan→Dec down the side (rows).
  let html='<div class="mtable" data-ny="'+nyears+'" style="grid-template-columns:max-content repeat('+nyears+',minmax(0,1fr));'+
           'grid-template-rows:auto repeat(12,minmax(0,1fr))">';
  html+='<div class="corner"></div>';
  for(let y=y0;y<=y1;y++){
    let yt=0; for(let mo=0;mo<12;mo++){ const a=agg[y+'-'+mo]; if(a) yt+=a.hrs; }
    html+='<div class="myear"><div class="yy">'+y+'</div><div class="yt">'+Math.round(yt)+'h</div></div>';
  }
  for(let mo=0;mo<12;mo++){
    html+='<div class="mmonth">'+MONTH[mo]+'</div>';
    for(let y=y0;y<=y1;y++){
      const monStart=new Date(y,mo,1).getTime(), monEnd=new Date(y,mo+1,0).getTime();
      const inRange = monEnd>=startMs && monStart<=todayMs;
      if(!inRange){ html+='<div class="mcell disabled"></div>'; continue; }
      const a=agg[y+'-'+mo];
      if(!a || a.hrs<=0){ html+='<div class="mcell empty"></div>'; continue; }
      const hLabel = a.hrs<10 ? a.hrs.toFixed(1) : Math.round(a.hrs);
      html+='<div class="mcell'+lvl(a.hrs)+'" data-y="'+y+'" data-m="'+mo+'">'+
              '<div class="mbooks"><span class="ic">✓</span>'+a.fin+'</div>'+
              '<div class="mhrs">'+hLabel+'h</div>'+
            '</div>';
    }
  }
  html+='</div><div class="mtitle">Tap a month to dive into its covers · swipe up / scroll in to zoom into the heatmap</div>';
  D.monthly.innerHTML=html;
  requestAnimationFrame(sizeMonthly);
}

// Make the month cells square while the 12 rows still fill the page height: measure the row height
// (rows are 1fr filling the page) and set each year column to that width. offset* metrics ignore the
// crossfade transform, so this stays correct mid-animation. Falls back to a width cap if too many years.
export function sizeMonthly(){
  const mt=D.monthly.querySelector('.mtable'); if(!mt) return;
  const ny=+mt.dataset.ny||1;
  // reset to flexible so we can re-measure the natural row height (needed on resize)
  mt.style.gridTemplateColumns='max-content repeat('+ny+',minmax(0,1fr))';
  mt.style.gridTemplateRows='auto repeat(12,minmax(0,1fr))';
  mt.style.justifyContent=''; mt.style.alignContent='stretch';
  const cell=mt.querySelector('.mcell'), label=mt.querySelector('.mmonth');
  if(!cell) return;
  const labelW=label?label.offsetWidth:40;
  const maxByW=(mt.clientWidth - labelW - ny*8 - 4)/ny;     // don't overflow the width
  const sq=Math.max(40, Math.floor(Math.min(cell.offsetHeight, maxByW)));
  mt.style.gridTemplateColumns='max-content repeat('+ny+','+sq+'px)';
  mt.style.gridTemplateRows='auto repeat(12,'+sq+'px)';
  mt.style.justifyContent='center'; mt.style.alignContent='center';
}

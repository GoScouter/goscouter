package scan

import (
	"encoding/json"
	"strings"
)

type Summary struct {
	Target     string
	Subdomains int
	Reachable  int
}

type node struct {
	ID        string     `json:"id"`
	Root      bool       `json:"root"`
	Reachable bool       `json:"reachable"`
	Report    HostReport `json:"report"`
}

type link struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type payload struct {
	Target string `json:"target"`
	Nodes  []node `json:"nodes"`
	Links  []link `json:"links"`
}

func (g *Graph) HTML() (string, Summary) {
	p := payload{}
	sum := Summary{}

	if g.Root != nil {
		p.Target = g.Root.Report.Host
		sum.Target = g.Root.Report.Host

		p.Nodes = append(p.Nodes, node{
			ID:        g.Root.Report.Host,
			Root:      true,
			Reachable: g.Root.Report.Reachable(),
			Report:    g.Root.Report,
		})

		for _, c := range g.Root.Children {
			sum.Subdomains++
			if c.Report.Reachable() {
				sum.Reachable++
			}
			p.Nodes = append(p.Nodes, node{
				ID:        c.Report.Host,
				Reachable: c.Report.Reachable(),
				Report:    c.Report,
			})
			p.Links = append(p.Links, link{Source: g.Root.Report.Host, Target: c.Report.Host})
		}
	}

	data, err := json.Marshal(p)
	if err != nil {
		data = []byte("null")
	}

	html := strings.Replace(pageTemplate, "/*__GRAPH_DATA__*/", string(data), 1)
	return html, sum
}

const pageTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>GoScouter scan</title>
<style>
  :root { color-scheme: dark; }
  * { box-sizing: border-box; }
  html, body { margin: 0; height: 100%; background: #0d1117; color: #c9d1d9;
    font: 14px/1.5 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
  #wrap { display: flex; height: 100%; }
  #graph { flex: 1; position: relative; }
  canvas { display: block; width: 100%; height: 100%; cursor: grab; }
  canvas:active { cursor: grabbing; }
  #panel { width: 380px; max-width: 45vw; border-left: 1px solid #21262d;
    background: #010409; padding: 18px; overflow-y: auto; }
  #panel h1 { font-size: 15px; margin: 0 0 4px; color: #58a6ff; word-break: break-all; }
  #panel .hint { color: #6e7681; }
  #panel h2 { font-size: 12px; text-transform: uppercase; letter-spacing: .08em;
    color: #8b949e; margin: 18px 0 6px; border-bottom: 1px solid #21262d; padding-bottom: 4px; }
  #panel .kv { display: grid; grid-template-columns: 74px 1fr; gap: 2px 10px; }
  #panel .kv .k { color: #6e7681; }
  #panel .kv .v { color: #c9d1d9; word-break: break-all; }
  #panel .err { color: #f85149; word-break: break-all; }
  #panel .badge { display: inline-block; padding: 1px 8px; border-radius: 999px;
    font-size: 11px; margin-top: 6px; }
  #panel .up { background: #12351d; color: #3fb950; }
  #panel .down { background: #3d1417; color: #f85149; }
  #legend { position: absolute; left: 14px; bottom: 12px; color: #6e7681; font-size: 12px; }
  #legend span { display: inline-flex; align-items: center; margin-right: 14px; }
  #legend i { width: 10px; height: 10px; border-radius: 50%; margin-right: 6px; }
  header { position: absolute; top: 12px; left: 16px; }
  header b { color: #58a6ff; } header small { color: #6e7681; margin-left: 8px; }
</style>
</head>
<body>
<div id="wrap">
  <div id="graph">
    <canvas id="cv"></canvas>
    <header><b>GoScouter</b> scan <small id="tgt"></small></header>
    <div id="legend">
      <span><i style="background:#3fb950"></i>reachable</span>
      <span><i style="background:#6e7681"></i>no HTTP response</span>
    </div>
  </div>
  <aside id="panel">
    <h1 id="pTitle">Select a node</h1>
    <div class="hint">Click any host in the graph to inspect its DNS and HTTP records. Drag to reposition; scroll the canvas to explore.</div>
    <div id="pBody"></div>
  </aside>
</div>
<script>
const DATA = /*__GRAPH_DATA__*/;
document.getElementById('tgt').textContent = DATA.target || '';

const cv = document.getElementById('cv'), ctx = cv.getContext('2d');
let W = 0, H = 0, DPR = window.devicePixelRatio || 1;
function resize() {
  W = cv.clientWidth; H = cv.clientHeight;
  cv.width = W * DPR; cv.height = H * DPR;
  ctx.setTransform(DPR, 0, 0, DPR, 0, 0);
}
window.addEventListener('resize', resize); resize();

// Lay nodes on a ring around the pinned root, then relax with a small force sim.
const N = DATA.nodes.map((n, i) => {
  const a = (i / Math.max(1, DATA.nodes.length)) * Math.PI * 2;
  const r = n.root ? 0 : 180 + (i % 5) * 26;
  return { ...n, x: W/2 + Math.cos(a)*r, y: H/2 + Math.sin(a)*r, vx: 0, vy: 0 };
});
const byId = Object.fromEntries(N.map(n => [n.id, n]));
const L = DATA.links.map(l => ({ s: byId[l.source], t: byId[l.target] })).filter(l => l.s && l.t);

const root = N.find(n => n.root);
let sel = null, drag = null, hover = null;

function step() {
  const cx = W/2, cy = H/2;
  for (const n of N) {
    if (n === drag) continue;
    // centering gravity
    n.vx += (cx - n.x) * 0.0015;
    n.vy += (cy - n.y) * 0.0015;
  }
  // pairwise repulsion
  for (let i = 0; i < N.length; i++) {
    for (let j = i+1; j < N.length; j++) {
      const a = N[i], b = N[j];
      let dx = a.x - b.x, dy = a.y - b.y;
      let d2 = dx*dx + dy*dy || 0.01;
      const f = 6000 / d2;
      const d = Math.sqrt(d2);
      const fx = (dx/d)*f, fy = (dy/d)*f;
      a.vx += fx; a.vy += fy; b.vx -= fx; b.vy -= fy;
    }
  }
  // spring along links
  for (const l of L) {
    let dx = l.t.x - l.s.x, dy = l.t.y - l.s.y;
    const d = Math.sqrt(dx*dx + dy*dy) || 0.01;
    const f = (d - 170) * 0.02;
    const fx = (dx/d)*f, fy = (dy/d)*f;
    l.s.vx += fx; l.s.vy += fy; l.t.vx -= fx; l.t.vy -= fy;
  }
  for (const n of N) {
    if (n === drag) continue;
    n.vx *= 0.86; n.vy *= 0.86;
    n.x += n.vx; n.y += n.vy;
  }
  if (root && root !== drag) { root.x = cx; root.y = cy; root.vx = root.vy = 0; }
}

function radius(n) { return n.root ? 11 : 7; }

function draw() {
  ctx.clearRect(0, 0, W, H);
  ctx.lineWidth = 1;
  for (const l of L) {
    ctx.strokeStyle = (hover && (hover === l.s || hover === l.t)) ? '#3fb95066' : '#21262d';
    ctx.beginPath(); ctx.moveTo(l.s.x, l.s.y); ctx.lineTo(l.t.x, l.t.y); ctx.stroke();
  }
  for (const n of N) {
    const col = n.reachable ? '#3fb950' : '#6e7681';
    ctx.beginPath(); ctx.arc(n.x, n.y, radius(n), 0, Math.PI*2);
    ctx.fillStyle = col; ctx.fill();
    if (n === sel) { ctx.strokeStyle = '#58a6ff'; ctx.lineWidth = 2.5; ctx.stroke(); ctx.lineWidth = 1; }
    if (n.root || n === hover || n === sel) {
      ctx.fillStyle = '#c9d1d9'; ctx.font = '12px ui-monospace, monospace';
      ctx.fillText(n.id, n.x + radius(n) + 4, n.y + 4);
    }
  }
}

function frame() { step(); draw(); requestAnimationFrame(frame); }
frame();

function at(mx, my) {
  let best = null, bd = 16*16;
  for (const n of N) {
    const dx = n.x - mx, dy = n.y - my, d2 = dx*dx + dy*dy;
    if (d2 < bd) { bd = d2; best = n; }
  }
  return best;
}
function pos(e) { const r = cv.getBoundingClientRect(); return [e.clientX - r.left, e.clientY - r.top]; }

cv.addEventListener('mousedown', e => { const [x,y] = pos(e); const n = at(x,y); if (n) { drag = n; select(n); } });
window.addEventListener('mousemove', e => {
  const [x,y] = pos(e);
  if (drag) { drag.x = x; drag.y = y; drag.vx = drag.vy = 0; }
  else { hover = at(x,y); cv.style.cursor = hover ? 'pointer' : 'grab'; }
});
window.addEventListener('mouseup', () => { drag = null; });

function esc(s) { return String(s).replace(/[&<>]/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;'}[c])); }

function dnsBlock(d) {
  if (!d) return '';
  const rows = [];
  const add = (k, v) => { if (v && v.length) rows.push(['k', k], ['v', Array.isArray(v) ? v.join(', ') : v]); };
  add('A', d.A); add('AAAA', d.AAAA); if (d.CNAME) add('CNAME', d.CNAME);
  add('MX', d.MX); add('NS', d.NS); add('TXT', d.TXT);
  if (!rows.length) return '<h2>DNS</h2><div class="hint">no records</div>';
  let out = '<h2>DNS</h2><div class="kv">';
  for (let i = 0; i < rows.length; i += 2) out += '<div class="'+rows[i][0]+'">'+esc(rows[i][1])+'</div><div class="v">'+esc(rows[i+1][1])+'</div>';
  return out + '</div>';
}
function httpBlock(title, h, err) {
  let out = '<h2>' + title + '</h2>';
  if (err) return out + '<div class="err">' + esc(err) + '</div>';
  if (!h) return out + '<div class="hint">not probed</div>';
  out += '<div class="kv">';
  out += '<div class="k">Status</div><div class="v">' + esc(h.Status || '') + '</div>';
  out += '<div class="k">Proto</div><div class="v">' + esc(h.Proto || '') + '</div>';
  if (h.FinalURL && h.FinalURL !== h.RequestURL) out += '<div class="k">Redirect</div><div class="v">' + esc(h.FinalURL) + '</div>';
  const hd = h.Headers || {};
  for (const k of Object.keys(hd).sort()) out += '<div class="k">' + esc(k) + '</div><div class="v">' + esc(hd[k].join(', ')) + '</div>';
  return out + '</div>';
}

function select(n) {
  sel = n;
  const r = n.report;
  document.getElementById('pTitle').textContent = r.host;
  const badge = n.reachable ? '<span class="badge up">reachable</span>' : '<span class="badge down">no HTTP response</span>';
  let html = badge;
  html += dnsBlock(r.dns);
  if (r.dnsErr) html += '<div class="err">DNS: ' + esc(r.dnsErr) + '</div>';
  html += httpBlock('HTTP', r.http, r.httpErr);
  html += httpBlock('HTTPS', r.https, r.httpsErr);
  document.getElementById('pBody').innerHTML = html;
}
if (root) select(root);
</script>
</body>
</html>
`

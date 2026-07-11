/**
 * 登录页左侧品牌插画：统一 API 枢纽 + 国内可用模型节点
 */

const MODEL_DEFS = [
  { label: 'DeepSeek', color: '#6366f1' },
  { label: 'Kimi', color: '#8b5cf6' },
  { label: 'GLM', color: '#7c3aed' },
  { label: 'Qwen', color: '#0891b2' },
  { label: '豆包', color: '#f97316' },
  { label: '文心', color: '#2563eb' },
  { label: '混元', color: '#059669' },
  { label: '星火', color: '#e11d48' },
  { label: 'MiniMax', color: '#a855f7' },
  { label: 'Yi', color: '#0ea5e9' },
];

const FEATURES = [
  { text: '低延迟', x: 168 },
  { text: '弹性计费', x: 260 },
  { text: '全链路监控', x: 368 },
];

const HUB = { x: 260, y: 172 };
const ORBIT = { rx: 168, ry: 102 };

/** 椭圆均匀排布，起始角略偏左，底部留出枢纽文案空间 */
function layoutOnOrbit(defs, hub, rx, ry) {
  const n = defs.length;
  const start = -Math.PI / 2 - Math.PI / 7;
  return defs.map((d, i) => {
    const angle = start + (2 * Math.PI * i) / n;
    return {
      ...d,
      x: Math.round(hub.x + rx * Math.cos(angle)),
      y: Math.round(hub.y + ry * Math.sin(angle)),
    };
  });
}

const MODELS = layoutOnOrbit(MODEL_DEFS, HUB, ORBIT.rx, ORBIT.ry);

const FLOW_DOTS = [
  { cx: 228, cy: 118 },
  { cx: 292, cy: 118 },
  { cx: 188, cy: 158 },
  { cx: 332, cy: 158 },
  { cx: 260, cy: 98 },
  { cx: 210, cy: 208 },
  { cx: 310, cy: 208 },
];

export default function LoginBrandIllustration() {
  return (
    <div className="tob-login-illus" aria-hidden>
      <svg
        viewBox="0 0 520 360"
        className="tob-login-illus-svg"
        xmlns="http://www.w3.org/2000/svg"
        overflow="visible"
      >
        <defs>
          <linearGradient id="tob-illus-grad-a" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#6366f1" />
            <stop offset="55%" stopColor="#8b5cf6" />
            <stop offset="100%" stopColor="#06b6d4" />
          </linearGradient>
          <linearGradient id="tob-illus-line" x1="0%" y1="0%" x2="100%" y2="0%">
            <stop offset="0%" stopColor="#6366f1" stopOpacity="0.12" />
            <stop offset="50%" stopColor="#8b5cf6" stopOpacity="0.5" />
            <stop offset="100%" stopColor="#06b6d4" stopOpacity="0.32" />
          </linearGradient>
          <radialGradient id="tob-illus-glow" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor="#6366f1" stopOpacity="0.35" />
            <stop offset="70%" stopColor="#8b5cf6" stopOpacity="0.08" />
            <stop offset="100%" stopColor="#06b6d4" stopOpacity="0" />
          </radialGradient>
          <filter id="tob-illus-blur" x="-20%" y="-20%" width="140%" height="140%">
            <feGaussianBlur stdDeviation="18" />
          </filter>
        </defs>

        <ellipse cx="260" cy="188" rx="210" ry="125" fill="url(#tob-illus-glow)" filter="url(#tob-illus-blur)" />
        <circle cx="120" cy="78" r="50" fill="#6366f1" fillOpacity="0.06" />
        <circle cx="400" cy="255" r="54" fill="#06b6d4" fillOpacity="0.08" />

        <ellipse
          className="tob-login-illus-orbit"
          cx={HUB.x}
          cy={HUB.y}
          rx={ORBIT.rx}
          ry={ORBIT.ry}
          fill="none"
          stroke="url(#tob-illus-line)"
          strokeWidth="1.5"
          strokeDasharray="6 10"
          opacity="0.65"
        />

        <g stroke="url(#tob-illus-line)" strokeWidth="1.5" fill="none" opacity="0.8">
          {MODELS.map((m) => (
            <path key={m.label} d={`M${HUB.x} ${HUB.y} L${m.x} ${m.y}`} />
          ))}
        </g>

        <g className="tob-login-illus-dots" fill="#6366f1">
          {FLOW_DOTS.map((d, i) => (
            <circle
              key={i}
              cx={d.cx}
              cy={d.cy}
              r={i % 2 === 0 ? 3 : 2.5}
              opacity={0.35 + (i % 3) * 0.12}
            />
          ))}
        </g>

        <g className="tob-login-illus-nodes">
          {MODELS.map((m) => (
            <ModelNode key={m.label} x={m.x} y={m.y} label={m.label} color={m.color} />
          ))}
        </g>

        <g className="tob-login-illus-hub">
          <circle cx={HUB.x} cy={HUB.y} r="52" fill="url(#tob-illus-glow)" />
          <circle cx={HUB.x} cy={HUB.y} r="44" fill="#fff" fillOpacity="0.95" />
          <circle
            cx={HUB.x}
            cy={HUB.y}
            r="44"
            fill="none"
            stroke="url(#tob-illus-grad-a)"
            strokeWidth="2"
          />
          <rect
            x={HUB.x - 16}
            y={HUB.y - 16}
            width="32"
            height="32"
            rx="9"
            fill="url(#tob-illus-grad-a)"
          />
          <g transform={`translate(${HUB.x - 8}, ${HUB.y - 8})`} fill="#fff">
            <path
              d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"
              transform="scale(0.75)"
            />
          </g>
          <text
            x={HUB.x}
            y={HUB.y + 58}
            textAnchor="middle"
            fill="#64748b"
            fontSize="11"
            fontWeight="600"
            fontFamily="system-ui, -apple-system, 'PingFang SC', sans-serif"
          >
            统一 API
          </text>
        </g>

        <g fontFamily="system-ui, -apple-system, 'PingFang SC', sans-serif">
          {FEATURES.map((f) => (
            <Chip key={f.text} x={f.x} y={318} text={f.text} />
          ))}
        </g>
      </svg>
    </div>
  );
}

function measureTextWidth(text, latinPx = 7, cjkPx = 12) {
  return [...text].reduce((sum, ch) => sum + (/[\u4e00-\u9fff]/.test(ch) ? cjkPx : latinPx), 0);
}

function ModelNode({ x, y, label, color }) {
  const textW = measureTextWidth(label);
  const w = Math.ceil(textW + 40);
  const h = 30;
  return (
    <g transform={`translate(${x - w / 2}, ${y - h / 2})`}>
      <rect width={w} height={h} rx={h / 2} fill="#fff" fillOpacity="0.95" />
      <rect
        width={w}
        height={h}
        rx={h / 2}
        fill="none"
        stroke={color}
        strokeWidth="1.2"
        strokeOpacity="0.35"
      />
      <circle cx="15" cy={h / 2} r="5" fill={color} fillOpacity="0.85" />
      <text
        x="28"
        y={h / 2 + 4}
        fill="#334155"
        fontSize="11"
        fontWeight="600"
        fontFamily="system-ui, -apple-system, 'PingFang SC', sans-serif"
      >
        {label}
      </text>
    </g>
  );
}

function Chip({ x, y, text }) {
  const textW = measureTextWidth(text, 10, 13);
  const w = Math.ceil(textW + 28);
  const h = 26;
  return (
    <g transform={`translate(${x - w / 2}, ${y})`}>
      <rect width={w} height={h} rx={h / 2} fill="#f8fafc" stroke="#e2e8f0" strokeWidth="1" />
      <text
        x={w / 2}
        y={h / 2 + 4}
        textAnchor="middle"
        fill="#64748b"
        fontSize="10"
        fontWeight="600"
        fontFamily="system-ui, -apple-system, 'PingFang SC', sans-serif"
      >
        {text}
      </text>
    </g>
  );
}

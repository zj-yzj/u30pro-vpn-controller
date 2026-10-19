import assert from 'node:assert/strict';

// Physical panel acceptance criteria, independent of Android dp / screenshot shape.
const center = 120, safeRadius = 116;
const regions = [
  { name: 'all interactive controls', x: 38, y: 38, width: 164, height: 164 },
  { name: 'centered title', x: 70, y: 18, width: 100, height: 18 },
  { name: 'centered footer', x: 68, y: 206, width: 104, height: 16 },
];
const report = [];
for (const rectangle of regions) {
  let maximum = 0;
  for (const x of [rectangle.x, rectangle.x + rectangle.width]) {
    for (const y of [rectangle.y, rectangle.y + rectangle.height]) {
      maximum = Math.max(maximum, Math.hypot(x - center, y - center));
    }
  }
  assert(maximum <= safeRadius, `${rectangle.name} extends beyond the 4px inset circle`);
  report.push({ ...rectangle, farthestCornerRadius: +maximum.toFixed(4), limit: safeRadius });
}

// A draggable square's center is projected into the circle reduced by its
// half-diagonal. Sample every direction, including rounding back to pixels.
const size = 38, half = size / 2;
const projectedRadius = center - Math.hypot(half, half) - 5;
let bubbleMaximum = 0;
for (let degrees = 0; degrees < 360; degrees += 0.25) {
  const angle = degrees * Math.PI / 180;
  const left = Math.round(center + Math.cos(angle) * projectedRadius - half);
  const top = Math.round(center + Math.sin(angle) * projectedRadius - half);
  for (const x of [left, left + size]) for (const y of [top, top + size]) {
    bubbleMaximum = Math.max(bubbleMaximum, Math.hypot(x - center, y - center));
  }
}
assert(bubbleMaximum <= safeRadius, 'rounded shortcut corners must stay inside inset circle');
console.log(JSON.stringify({ ok: true, framebuffer: [240, 240], visibleRadius: 120,
  acceptanceRadius: safeRadius, regions: report,
  shortcut: { size, sampledDirections: 1440, farthestCornerRadius: +bubbleMaximum.toFixed(4) },
  limitation: 'Geometry validates bounds only; inspect real screenshots through the round mask and confirm on the physical device.'
}, null, 2));

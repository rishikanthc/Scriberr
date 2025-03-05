const colors = [
  '#FF6B6B', // Coral Red
  '#4ECDC4', // Turquoise
  '#45B7D1', // Sky Blue
  '#96CEB4', // Sage Green
  '#D4A5A5', // Dusty Rose
  '#9B72AA', // Purple
  '#FFB347', // Orange
  '#87A7B3'  // Steel Blue
];

const speakerColorMap = new Map<string, string>();
let colorIndex = 0;

export function getSpeakerColor(speaker: string): string {
  if (!speaker) return '#6B7280';
  
  if (!speakerColorMap.has(speaker)) {
    speakerColorMap.set(speaker, colors[colorIndex % colors.length]);
    colorIndex++;
  }
  
  return speakerColorMap.get(speaker) || '#6B7280';
}


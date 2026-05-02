import { useEffect, useRef } from "react";

type RecordingPulseFieldProps = {
  stream: MediaStream | null;
  active: boolean;
  paused: boolean;
};

type AudioContextWindow = Window & { webkitAudioContext?: typeof AudioContext };
type RgbColor = { r: number; g: number; b: number };
type VisualColors = { ink: RgbColor; accent: RgbColor };

export function RecordingPulseField({ stream, active, paused }: RecordingPulseFieldProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const context = canvas.getContext("2d", { alpha: true });
    if (!context) return;

    let animationFrame = 0;
    let audioContext: AudioContext | null = null;
    let analyser: AnalyserNode | null = null;
    let source: MediaStreamAudioSourceNode | null = null;
    let timeData: Uint8Array | null = null;
    let frequencyData: Uint8Array | null = null;
    let smoothedEnergy = 0;
    let smoothedSpeech = 0;
    let smoothedLow = 0;
    let smoothedHigh = 0;
    let lastDrawnAt = 0;
    let colors = resolveVisualColors(canvas);

    const resize = () => {
      const rect = canvas.getBoundingClientRect();
      const pixelRatio = Math.min(window.devicePixelRatio || 1, 2);
      const width = Math.max(1, Math.floor(rect.width * pixelRatio));
      const height = Math.max(1, Math.floor(rect.height * pixelRatio));
      if (canvas.width !== width || canvas.height !== height) {
        canvas.width = width;
        canvas.height = height;
        colors = resolveVisualColors(canvas);
      }
      context.setTransform(pixelRatio, 0, 0, pixelRatio, 0, 0);
    };

    const connectAnalyser = () => {
      if (!stream || !active || paused) return;
      const AudioContextClass = window.AudioContext || (window as AudioContextWindow).webkitAudioContext;
      if (!AudioContextClass) return;

      audioContext = new AudioContextClass();
      void audioContext.resume();
      analyser = audioContext.createAnalyser();
      analyser.fftSize = 1024;
      analyser.smoothingTimeConstant = 0.82;
      source = audioContext.createMediaStreamSource(stream);
      source.connect(analyser);
      timeData = new Uint8Array(analyser.fftSize);
      frequencyData = new Uint8Array(analyser.frequencyBinCount);
    };

    const draw = (timestamp: number) => {
      resize();
      if (timestamp - lastDrawnAt < 33) {
        animationFrame = window.requestAnimationFrame(draw);
        return;
      }
      lastDrawnAt = timestamp;

      const width = canvas.clientWidth;
      const height = canvas.clientHeight;
      context.clearRect(0, 0, width, height);

      let energy = 0;
      let speech = 0;
      let low = 0;
      let high = 0;

      if (analyser && timeData && frequencyData && active && !paused) {
        analyser.getByteTimeDomainData(timeData);
        analyser.getByteFrequencyData(frequencyData);

        for (const sample of timeData) {
          const normalized = (sample - 128) / 128;
          energy += normalized * normalized;
        }
        energy = Math.sqrt(energy / timeData.length);

        const lowBandEnd = Math.min(frequencyData.length, 28);
        const speechBandEnd = Math.min(frequencyData.length, 120);
        const highBandEnd = Math.min(frequencyData.length, 190);
        for (let index = 2; index < lowBandEnd; index += 1) {
          low += frequencyData[index] / 255;
        }
        low /= Math.max(1, lowBandEnd - 2);
        for (let index = 8; index < speechBandEnd; index += 1) {
          const weight = index < lowBandEnd ? 0.45 : 1;
          speech += (frequencyData[index] / 255) * weight;
        }
        speech /= Math.max(1, speechBandEnd - 8);
        for (let index = speechBandEnd; index < highBandEnd; index += 1) {
          high += frequencyData[index] / 255;
        }
        high /= Math.max(1, highBandEnd - speechBandEnd);
      }

      smoothedEnergy += (energy - smoothedEnergy) * 0.16;
      smoothedSpeech += (speech - smoothedSpeech) * 0.12;
      smoothedLow += (low - smoothedLow) * 0.12;
      smoothedHigh += (high - smoothedHigh) * 0.1;

      drawDistortedDotLattice(context, {
        width,
        height,
        timestamp,
        active,
        paused,
        energy: smoothedEnergy,
        speech: smoothedSpeech,
        low: smoothedLow,
        high: smoothedHigh,
        colors,
      });

      animationFrame = window.requestAnimationFrame(draw);
    };

    resize();
    connectAnalyser();
    animationFrame = window.requestAnimationFrame(draw);
    window.addEventListener("resize", resize);

    return () => {
      window.cancelAnimationFrame(animationFrame);
      window.removeEventListener("resize", resize);
      source?.disconnect();
      analyser?.disconnect();
      void audioContext?.close();
    };
  }, [active, paused, stream]);

  return (
    <div className="scr-recording-pulse-field" aria-hidden="true">
      <canvas ref={canvasRef} />
    </div>
  );
}

function drawDistortedDotLattice(
  context: CanvasRenderingContext2D,
  options: {
    width: number;
    height: number;
    timestamp: number;
    active: boolean;
    paused: boolean;
    energy: number;
    speech: number;
    low: number;
    high: number;
    colors: VisualColors;
  }
) {
  const { width, height, timestamp, active, paused, energy, speech, low, high, colors } = options;
  const sound = active && !paused ? Math.min(1, energy * 5.8 + speech * 2.7) : 0;
  const motion = sound > 0.025 ? smoothstep(0.025, 0.62, sound) : 0;
  const spacing = 12;
  const columns = Math.ceil(width / spacing) + 6;
  const rows = Math.ceil(height / spacing) + 6;
  const time = timestamp * 0.001;
  const centerX = width * (0.5 + Math.sin(time * 0.31) * motion * 0.07);
  const centerY = height * (0.48 + Math.cos(time * 0.27) * motion * 0.08);
  const maxRadius = Math.max(width, height) * 0.58;
  const twist = motion * (1.7 + low * 2.4);
  const ripple = motion * (5 + speech * 16);
  const baseAlpha = active && !paused ? 0.12 : 0.08;
  const baseRadius = active && !paused ? 0.82 : 0.62;

  for (let row = -3; row < rows; row += 1) {
    const y = row * spacing;
    for (let column = -3; column < columns; column += 1) {
      const x = column * spacing;
      const dx = x - centerX;
      const dy = y - centerY;
      const radiusFromCenter = Math.sqrt(dx * dx + dy * dy);
      const normalizedRadius = Math.min(1, radiusFromCenter / maxRadius);
      const angle = Math.atan2(dy, dx);
      const falloff = Math.pow(1 - normalizedRadius, 1.85);
      const spin = twist * falloff * Math.sin(normalizedRadius * 7.2 - time * (1.1 + speech * 1.5));
      const warpedAngle = angle + spin;
      const radialWave = Math.sin(normalizedRadius * 28 - time * (2.1 + low * 2.8) + angle * 2.2);
      const tangentialWave = Math.cos((x / width) * 9.5 - (y / height) * 7.3 + time * (1.3 + high * 2.2));
      const push = radialWave * ripple * falloff + tangentialWave * motion * 5;
      const warpedRadius = radiusFromCenter + push;
      const warpedX = centerX + Math.cos(warpedAngle) * warpedRadius;
      const warpedY = centerY + Math.sin(warpedAngle) * warpedRadius;
      const vortex = smoothstep(0.12, 0.88, falloff * motion);
      const waveFocus = Math.max(0, radialWave * 0.5 + 0.5) * vortex;
      const lineInterference = Math.max(0, Math.sin((x + y) * 0.025 + time * (1.6 + speech)) * 0.5 + 0.5);
      const intensity = Math.min(1, waveFocus * 0.72 + lineInterference * motion * 0.24 + high * 0.34 * falloff);
      const dotRadius = baseRadius + intensity * (3.1 + motion * 2.2);
      const alpha = Math.min(0.88, baseAlpha + intensity * (0.18 + motion * 0.58));
      const color = intensity > 0.68 && motion > 0.34 ? colors.accent : colors.ink;

      context.beginPath();
      context.fillStyle = `rgba(${color.r}, ${color.g}, ${color.b}, ${alpha})`;
      context.arc(warpedX, warpedY, dotRadius, 0, Math.PI * 2);
      context.fill();
    }
  }
}

function smoothstep(edge0: number, edge1: number, value: number) {
  const t = Math.max(0, Math.min(1, (value - edge0) / (edge1 - edge0)));
  return t * t * (3 - 2 * t);
}

function resolveVisualColors(element: HTMLElement): VisualColors {
  const styles = getComputedStyle(element);
  return {
    ink: parseRgb(styles.getPropertyValue("--scr-recorder-visual-ink").trim(), { r: 32, g: 32, b: 34 }),
    accent: parseRgb(styles.getPropertyValue("--scr-recorder-visual-accent").trim(), { r: 255, g: 105, b: 38 }),
  };
}

function parseRgb(value: string, fallback: RgbColor): RgbColor {
  const match = value.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
  if (!match) return fallback;
  return {
    r: Number(match[1] || fallback.r),
    g: Number(match[2] || fallback.g),
    b: Number(match[3] || fallback.b),
  };
}

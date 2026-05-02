import { useEffect, useRef } from "react";

type RecordingPulseFieldProps = {
  stream: MediaStream | null;
  active: boolean;
  paused: boolean;
};

type AudioContextWindow = Window & { webkitAudioContext?: typeof AudioContext };
type RgbColor = { r: number; g: number; b: number };

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
    let inkColor = resolveInkColor(canvas);

    const resize = () => {
      const rect = canvas.getBoundingClientRect();
      const pixelRatio = Math.min(window.devicePixelRatio || 1, 2);
      const width = Math.max(1, Math.floor(rect.width * pixelRatio));
      const height = Math.max(1, Math.floor(rect.height * pixelRatio));
      if (canvas.width !== width || canvas.height !== height) {
        canvas.width = width;
        canvas.height = height;
        inkColor = resolveInkColor(canvas);
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

      drawHalftoneField(context, {
        width,
        height,
        timestamp,
        active,
        paused,
        energy: smoothedEnergy,
        speech: smoothedSpeech,
        low: smoothedLow,
        high: smoothedHigh,
        color: inkColor,
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

function drawHalftoneField(
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
    color: RgbColor;
  }
) {
  const { width, height, timestamp, active, paused, energy, speech, low, high, color } = options;
  const sound = active && !paused ? Math.min(1, energy * 5.2 + speech * 2.4) : 0;
  const visible = sound > 0.035 ? Math.min(1, sound) : 0;
  const spacing = 11;
  const columns = Math.ceil(width / spacing) + 4;
  const rows = Math.ceil(height / spacing) + 4;
  const time = timestamp * 0.001;
  const amplitude = height * (0.055 + visible * 0.24 + low * 0.12);
  const thickness = height * (0.035 + visible * 0.12);
  const contrast = Math.min(1, 0.18 + visible * 0.8 + high * 0.28);

  for (let row = -2; row < rows; row += 1) {
    const y = row * spacing + ((row % 2) * spacing) / 2;
    for (let column = -2; column < columns; column += 1) {
      const x = column * spacing;
      const nx = width > 0 ? x / width : 0;
      const ny = height > 0 ? y / height : 0;
      const sweep = Math.sin(nx * 13.4 + time * (0.45 + speech * 1.3));
      const ripple = Math.sin(nx * 27.5 - time * (0.72 + low * 1.7));
      const contour = Math.sin(nx * 4.6 + time * 0.2) * amplitude * 0.34;
      const upper = height * 0.18 + Math.sin(nx * 8.4 + time * 0.62) * amplitude + ripple * amplitude * 0.28;
      const middle = height * 0.48 + Math.sin(nx * 9.8 - time * 0.5 + 1.7) * amplitude * 0.88 + sweep * amplitude * 0.2 + contour;
      const lower = height * 0.8 + Math.sin(nx * 7.2 + time * 0.38 + 3.1) * amplitude * 0.72 - ripple * amplitude * 0.22;
      const quietGhost = active && !paused ? 0 : 0.018 * Math.sin(nx * 20 + ny * 16);
      const band =
        gaussianDistance(y, upper, thickness * 0.95) * (0.72 + high * 0.25) +
        gaussianDistance(y, middle, thickness * 1.16) * (0.92 + speech * 0.3) +
        gaussianDistance(y, lower, thickness * 0.9) * (0.62 + low * 0.32) +
        quietGhost;
      const cluster = 0.52 + 0.48 * Math.sin(nx * 19.5 + ny * 11.5 + time * (0.28 + visible));
      const edgeFade = Math.sin(Math.PI * Math.max(0, Math.min(1, nx))) * 0.9 + 0.1;
      const density = Math.max(0, Math.min(1, band * cluster * edgeFade * contrast));
      if (density < 0.035) continue;

      const radius = 0.55 + density * (2.9 + visible * 1.6);
      const alpha = Math.min(0.82, density * (0.18 + visible * 0.7));
      context.beginPath();
      context.fillStyle = `rgba(${color.r}, ${color.g}, ${color.b}, ${alpha})`;
      context.arc(x, y, radius, 0, Math.PI * 2);
      context.fill();
    }
  }
}

function gaussianDistance(value: number, center: number, width: number) {
  const distance = (value - center) / Math.max(1, width);
  return Math.exp(-distance * distance);
}

function resolveInkColor(element: HTMLElement): RgbColor {
  const color = getComputedStyle(element).getPropertyValue("--scr-recorder-visual-ink").trim();
  const match = color.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
  if (!match) return { r: 32, g: 32, b: 34 };
  return {
    r: Number(match[1] || 32),
    g: Number(match[2] || 32),
    b: Number(match[3] || 34),
  };
}

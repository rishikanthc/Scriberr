import { useEffect, useRef } from "react";

type RecordingPulseFieldProps = {
  stream: MediaStream | null;
  active: boolean;
  paused: boolean;
};

type PulseDot = {
  x: number;
  y: number;
  radius: number;
  phase: number;
  band: number;
};

const dotCount = 84;
type AudioContextWindow = Window & { webkitAudioContext?: typeof AudioContext };

export function RecordingPulseField({ stream, active, paused }: RecordingPulseFieldProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const context = canvas.getContext("2d", { alpha: true });
    if (!context) return;

    const dots = createDots();
    let animationFrame = 0;
    let audioContext: AudioContext | null = null;
    let analyser: AnalyserNode | null = null;
    let source: MediaStreamAudioSourceNode | null = null;
    let timeData: Uint8Array | null = null;
    let frequencyData: Uint8Array | null = null;
    let smoothedEnergy = 0;
    let smoothedSpeech = 0;
    let lastDrawnAt = 0;

    const resize = () => {
      const rect = canvas.getBoundingClientRect();
      const pixelRatio = Math.min(window.devicePixelRatio || 1, 2);
      const width = Math.max(1, Math.floor(rect.width * pixelRatio));
      const height = Math.max(1, Math.floor(rect.height * pixelRatio));
      if (canvas.width !== width || canvas.height !== height) {
        canvas.width = width;
        canvas.height = height;
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
      let brightness = 0;

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
        for (let index = 8; index < speechBandEnd; index += 1) {
          const weight = index < lowBandEnd ? 0.45 : 1;
          speech += (frequencyData[index] / 255) * weight;
        }
        speech /= Math.max(1, speechBandEnd - 8);
        brightness = Math.min(1, energy * 4.8 + speech * 1.8);
      }

      smoothedEnergy += (energy - smoothedEnergy) * 0.16;
      smoothedSpeech += (speech - smoothedSpeech) * 0.12;

      const pulse = active && !paused ? Math.max(smoothedEnergy, smoothedSpeech * 0.75) : 0;
      const plainness = pulse < 0.018 ? 0 : Math.min(1, pulse * 7);
      const centerX = width * (0.5 + Math.sin(timestamp * 0.00022) * 0.08);
      const centerY = height * (0.5 + Math.cos(timestamp * 0.00018) * 0.12);
      const spread = 0.18 + Math.min(0.62, pulse * 1.9);

      if (plainness > 0.01) {
        const glow = context.createRadialGradient(centerX, centerY, 0, centerX, centerY, Math.max(width, height) * spread);
        glow.addColorStop(0, `rgba(255, 106, 0, ${0.045 + brightness * 0.08})`);
        glow.addColorStop(0.45, `rgba(255, 160, 88, ${0.018 + brightness * 0.045})`);
        glow.addColorStop(1, "rgba(255, 255, 255, 0)");
        context.fillStyle = glow;
        context.fillRect(0, 0, width, height);
      }

      for (const dot of dots) {
        const distanceX = dot.x - 0.5;
        const distanceY = dot.y - 0.5;
        const distance = Math.sqrt(distanceX * distanceX + distanceY * distanceY);
        const wave = Math.sin(timestamp * (0.0014 + dot.band * 0.00026) + dot.phase + pulse * 7);
        const localPulse = Math.max(0, 1 - Math.abs(distance - pulse * 0.52) / 0.34);
        const density = Math.min(1, plainness * (0.35 + localPulse * 0.9 + dot.band * 0.15));
        if (density < 0.02) continue;

        const drift = wave * pulse * 18;
        const x = dot.x * width + distanceX * pulse * 34 + drift * (0.45 + dot.band * 0.2);
        const y = dot.y * height + distanceY * pulse * 22 + Math.cos(dot.phase + timestamp * 0.001) * pulse * 10;
        const radius = dot.radius + density * (3.2 + dot.band * 2.2);
        const alpha = Math.min(0.72, density * (0.14 + brightness * 0.36));

        context.beginPath();
        context.fillStyle = `rgba(255, ${138 + Math.round(dot.band * 68)}, ${72 + Math.round(dot.band * 70)}, ${alpha})`;
        context.arc(x, y, radius, 0, Math.PI * 2);
        context.fill();
      }

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

function createDots(): PulseDot[] {
  return Array.from({ length: dotCount }, (_, index) => {
    const ring = Math.sqrt((index + 0.5) / dotCount);
    const angle = index * 2.399963229728653;
    return {
      x: 0.5 + Math.cos(angle) * ring * 0.46,
      y: 0.5 + Math.sin(angle) * ring * 0.34,
      radius: 0.85 + (index % 5) * 0.16,
      phase: angle * 1.7,
      band: (index % 9) / 8,
    };
  });
}

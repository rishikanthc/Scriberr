import { useEffect, useRef, useState } from "react";

interface AudioVisualizerProps {
    audioRef: React.RefObject<HTMLAudioElement | null>;
    isPlaying: boolean;
    isHovering?: boolean;
    hoverPercent?: number;
}

// Global cache to prevent "InvalidStateError" when React re-renders
// This ensures we don't try to create a new SourceNode for an audio element that already has one.
const audioSourceMap = new WeakMap<HTMLAudioElement, MediaElementAudioSourceNode>();

export function AudioVisualizer({
    audioRef,
    isPlaying,
    isHovering = false,
    hoverPercent = 0,
}: AudioVisualizerProps) {
    const canvasRef = useRef<HTMLCanvasElement>(null);
    const containerRef = useRef<HTMLDivElement>(null);
    const contextRef = useRef<AudioContext | null>(null);
    const analyzerRef = useRef<AnalyserNode | null>(null);
    const rafRef = useRef<number | null>(null);

    const peakPositionsRef = useRef<number[]>([]);
    const peakDropsRef = useRef<number[]>([]);
    const [dimensions, setDimensions] = useState({ width: 0, height: 0 });

    // Persist frequency data across renders/pauses
    const dataArrayRef = useRef<Uint8Array | null>(null);

    // Initialize with dummy data "wave" pattern for initial state
    useEffect(() => {
        if (!dataArrayRef.current) {
            const size = 128; // analyser.fftSize / 2
            const data = new Uint8Array(size);
            for (let i = 0; i < size; i++) {
                // Gentle bell curve wave
                const x = (i / size) * Math.PI;
                const val = Math.sin(x) * 100; // Amplitude ~100/255
                data[i] = val;
            }
            dataArrayRef.current = data;
        }
    }, []);

    // 1. Handle Window/Container Resizing
    useEffect(() => {
        const updateSize = () => {
            if (containerRef.current) {
                const { clientWidth, clientHeight } = containerRef.current;
                setDimensions({
                    width: clientWidth * window.devicePixelRatio,
                    height: clientHeight * window.devicePixelRatio,
                });
            }
        };

        updateSize();
        const observer = new ResizeObserver(updateSize);
        if (containerRef.current) observer.observe(containerRef.current);
        return () => observer.disconnect();
    }, []);

    // 2. Initialize Audio Context & Analyzer
    useEffect(() => {
        if (!audioRef.current) return;

        const initAudio = () => {
            if (!contextRef.current) {
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                const AudioContextClass = window.AudioContext || (window as any).webkitAudioContext;
                contextRef.current = new AudioContextClass();
            }
            const ctx = contextRef.current;

            if (!analyzerRef.current) {
                const analyzer = ctx.createAnalyser();
                analyzer.fftSize = 256;
                analyzer.smoothingTimeConstant = 0.8;
                analyzerRef.current = analyzer;
            }

            const audioEl = audioRef.current!;

            // Check cache to reuse existing MediaElementSource
            if (audioSourceMap.has(audioEl)) {
                try {
                    const source = audioSourceMap.get(audioEl)!;
                    source.connect(analyzerRef.current!);
                    analyzerRef.current!.connect(ctx.destination);
                } catch { /* ignore already connected errors */ }
            } else {
                try {
                    const source = ctx.createMediaElementSource(audioEl);
                    source.connect(analyzerRef.current!);
                    analyzerRef.current!.connect(ctx.destination);
                    audioSourceMap.set(audioEl, source);
                } catch (e) {
                    console.error("Audio Graph Error:", e);
                }
            }
        };

        initAudio();

        if (isPlaying && contextRef.current?.state === "suspended") {
            contextRef.current.resume();
        }
    }, [audioRef, isPlaying]);

    // 3. The Drawing Loop (The "Electric Ember" Design)
    useEffect(() => {
        if (!canvasRef.current || !analyzerRef.current || dimensions.width === 0)
            return;

        const canvas = canvasRef.current;
        const ctx = canvas.getContext("2d", { alpha: true });
        if (!ctx) return;

        const SCALE = window.devicePixelRatio;
        const TILE_SIZE = 5 * SCALE;
        const COL_GAP = 2 * SCALE;
        const ROW_GAP = 2 * SCALE;
        const TOTAL_ROW_HEIGHT = TILE_SIZE + ROW_GAP;
        const COL_WIDTH = TILE_SIZE + COL_GAP;

        // --- THEME GRADIENT (Electric Ember) ---
        const gradient = ctx.createLinearGradient(0, 0, 0, dimensions.height);
        gradient.addColorStop(0, "#FFAB40"); // Top: Amber
        gradient.addColorStop(0.5, "#FF6D1F"); // Mid: Orange
        gradient.addColorStop(1, "#FF3D00");   // Bottom: Deep Red

        const draw = () => {
            // Ensure we have data array buffer
            if (!dataArrayRef.current && analyzerRef.current) {
                dataArrayRef.current = new Uint8Array(analyzerRef.current.frequencyBinCount);
            }
            if (!dataArrayRef.current) return;

            const dataArray = dataArrayRef.current;

            if (isPlaying && analyzerRef.current) {
                analyzerRef.current.getByteFrequencyData(dataArray);
            }
            // If !isPlaying, we just reuse the existing dataArray (persisting the last frame)

            ctx.clearRect(0, 0, dimensions.width, dimensions.height);
            const barCount = Math.floor(dimensions.width / COL_WIDTH);

            if (peakPositionsRef.current.length !== barCount) {
                peakPositionsRef.current = new Array(barCount).fill(0);
                peakDropsRef.current = new Array(barCount).fill(0);
            }

            const maxTilesColumn = Math.floor(dimensions.height / TOTAL_ROW_HEIGHT);

            for (let i = 0; i < barCount; i++) {
                const binIndex = Math.floor(i * (dataArray.length / barCount) * 0.7);
                let value = dataArray[binIndex] || 0;
                value = Math.min(255, value * 1.2);

                const x = i * COL_WIDTH;
                const activeTiles = Math.floor((value / 255) * maxTilesColumn);

                // Peak Logic
                if (activeTiles > peakPositionsRef.current[i]) {
                    peakPositionsRef.current[i] = activeTiles;
                    peakDropsRef.current[i] = 0;
                } else {
                    peakDropsRef.current[i]++;
                    if (peakDropsRef.current[i] > 5) { // Hold peak for 5 frames
                        peakPositionsRef.current[i] = Math.max(0, peakPositionsRef.current[i] - 1);
                        peakDropsRef.current[i] = 0;
                    }
                }
                const peakTile = peakPositionsRef.current[i];

                for (let j = 0; j < maxTilesColumn; j++) {
                    const y = dimensions.height - j * TOTAL_ROW_HEIGHT - TILE_SIZE;

                    if (j < activeTiles) {
                        // Main Bar
                        ctx.fillStyle = gradient;
                        ctx.globalAlpha = 0.8;
                        ctx.beginPath();
                        if (ctx.roundRect) {
                            ctx.roundRect(x, y, TILE_SIZE, TILE_SIZE, 1 * SCALE);
                        } else {
                            ctx.rect(x, y, TILE_SIZE, TILE_SIZE);
                        }
                        ctx.fill();
                        ctx.globalAlpha = 1.0;
                    } else if (j === peakTile && peakTile > 0 && isPlaying) {
                        // Floating Peak
                        ctx.fillStyle = "#FFAB40";
                        ctx.globalAlpha = 0.5;
                        ctx.beginPath();
                        if (ctx.roundRect) {
                            ctx.roundRect(x, y, TILE_SIZE, TILE_SIZE, 1 * SCALE);
                        } else {
                            ctx.rect(x, y, TILE_SIZE, TILE_SIZE);
                        }
                        ctx.fill();
                        ctx.globalAlpha = 1.0;
                    }
                }
            }

            // Always animate to keep the visualizer active even when paused (to show static state)
            // or we could stop it if purely static, but "electric ember" might want subtle idle animation later.
            // For now, keep it running to render at least the static frame.
            rafRef.current = requestAnimationFrame(draw);
        };

        draw();

        return () => {
            if (rafRef.current) cancelAnimationFrame(rafRef.current);
        };
    }, [isPlaying, isHovering, hoverPercent, dimensions]);

    return (
        <div ref={containerRef} className="w-full h-full">
            <canvas
                ref={canvasRef}
                width={dimensions.width}
                height={dimensions.height}
                style={{ width: "100%", height: "100%" }}
                className="block"
            />
        </div>
    );
}
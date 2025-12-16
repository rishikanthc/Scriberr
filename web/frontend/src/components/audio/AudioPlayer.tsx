import { useEffect, useRef, useState, forwardRef, useImperativeHandle } from 'react';
import WaveSurfer from 'wavesurfer.js';
import { Play, Pause } from 'lucide-react';

import { useTheme } from '@/contexts/ThemeContext';
import { useAuth } from '@/features/auth/hooks/useAuth';
import { Button } from '@/components/ui/button';

export interface AudioPlayerRef {
    playPause: () => void;
    seekTo: (ratio: number) => void;
    getCurrentTime: () => number;
    getDuration: () => number;
    isPlaying: () => boolean;
}

interface AudioPlayerProps {
    audioId: string;
    collapsed?: boolean;
    onToggleCollapse?: () => void;
    onTimeUpdate?: (time: number) => void;
    onPlayStateChange?: (isPlaying: boolean) => void;
    onDurationChange?: (duration: number) => void;
    className?: string;
}

export const AudioPlayer = forwardRef<AudioPlayerRef, AudioPlayerProps>(({
    audioId,
    collapsed = false,
    onTimeUpdate,
    onPlayStateChange,
    onDurationChange,
    className = ''
}, ref) => {
    const { theme } = useTheme();
    const { getAuthHeaders } = useAuth();
    const containerRef = useRef<HTMLDivElement>(null);
    const wavesurferRef = useRef<WaveSurfer | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [loadingProgress, setLoadingProgress] = useState(0);
    const [isPlaying, setIsPlaying] = useState(false);
    const [duration, setDuration] = useState(0);
    const [currentTime, setCurrentTime] = useState(0);

    useImperativeHandle(ref, () => ({
        playPause: () => wavesurferRef.current?.playPause(),
        seekTo: (ratio: number) => wavesurferRef.current?.seekTo(ratio),
        getCurrentTime: () => wavesurferRef.current?.getCurrentTime() || 0,
        getDuration: () => wavesurferRef.current?.getDuration() || 0,
        isPlaying: () => wavesurferRef.current?.isPlaying() || false,
    }));

    useEffect(() => {
        if (!containerRef.current) return;

        const initWaveSurfer = async () => {
            try {
                setIsLoading(true);
                setError(null);
                setLoadingProgress(0);

                const audioUrl = `/api/v1/transcription/${audioId}/audio`;

                const isDark = theme === 'dark';

                // Design System Colors
                // Wave: Neutral Gray (Light: #D4D4D4, Dark: #404040)
                // Progress: Brand Solid #FF6D20
                // Cursor: Brand Solid #FF6D20 with opacity

                const waveColor = isDark ? '#404040' : '#E5E5E5';
                const progressColor = '#FF6D20'; // Brand Solid
                const cursorColor = '#FF6D20';   // Brand Solid

                const ws = WaveSurfer.create({
                    container: containerRef.current!,
                    waveColor,
                    progressColor,
                    cursorColor,
                    barWidth: 3,
                    barGap: 3,
                    barRadius: 3,
                    height: collapsed ? 0 : 48, // Compact height
                    normalize: true,
                    backend: 'WebAudio',
                    dragToSeek: true,
                    fetchParams: {
                        headers: { ...getAuthHeaders() }
                    }
                });

                wavesurferRef.current = ws;

                ws.on('loading', (percent) => {
                    setLoadingProgress(percent);
                });

                ws.on('ready', () => {
                    setIsLoading(false);
                    const dur = ws.getDuration();
                    setDuration(dur);
                    onDurationChange?.(dur);
                });

                ws.on('error', (err) => {
                    console.error("WaveSurfer error:", err);
                    setError("Failed to load audio. Please try again.");
                    setIsLoading(false);
                });

                ws.on('play', () => {
                    setIsPlaying(true);
                    onPlayStateChange?.(true);
                });

                ws.on('pause', () => {
                    setIsPlaying(false);
                    onPlayStateChange?.(false);
                });

                ws.on('timeupdate', (time) => {
                    setCurrentTime(time);
                    onTimeUpdate?.(time);
                });

                ws.on('finish', () => {
                    setIsPlaying(false);
                    onPlayStateChange?.(false);
                });

                await ws.load(audioUrl);

            } catch (error) {
                console.error('Error initializing audio player:', error);
                setError("An unexpected error occurred.");
                setIsLoading(false);
            }
        };

        initWaveSurfer();

        return () => {
            if (wavesurferRef.current) {
                wavesurferRef.current.destroy();
                wavesurferRef.current = null;
            }
        };
    }, [audioId, theme, getAuthHeaders]); // eslint-disable-line react-hooks/exhaustive-deps

    // Update height when collapsed state changes
    useEffect(() => {
        if (wavesurferRef.current) {
            wavesurferRef.current.setOptions({
                height: collapsed ? 0 : 48
            });
        }
    }, [collapsed]);

    const togglePlayPause = () => {
        wavesurferRef.current?.playPause();
    };

    const formatTime = (seconds: number) => {
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    };

    const retryLoad = () => {
        window.location.reload();
    };

    if (error) {
        return (
            <div className={`transition-all duration-300 ${className} flex items-center justify-center p-8 bg-[var(--error)]/5 rounded-[var(--radius-card)] border border-[var(--error)]/20`}>
                <div className="text-center">
                    <p className="text-[var(--error)] mb-2">{error}</p>
                    <Button
                        variant="destructive"
                        onClick={retryLoad}
                    >
                        Retry
                    </Button>
                </div>
            </div>
        );
    }

    return (
        <div className={`transition-all duration-300 ${className}`}>

            <div className="flex items-center gap-6">
                {/* Play/Pause Button - Premium Gradient & Shadow */}
                <button
                    onClick={togglePlayPause}
                    disabled={isLoading}
                    className={`
                        group relative w-12 h-12 flex-shrink-0 flex items-center justify-center 
                        rounded-full text-white shadow-lg shadow-orange-500/30 
                        transition-all duration-300 hover:scale-105 hover:shadow-orange-500/40
                        active:scale-95 border-none outline-none
                        ${isLoading ? 'opacity-50 cursor-not-allowed' : ''}
                    `}
                    style={{ background: 'var(--brand-gradient)' }}
                >
                    {/* Inner glow effect */}
                    <div className="absolute inset-0 rounded-full bg-white/10 opacity-0 group-hover:opacity-100 transition-opacity" />

                    {isLoading ? (
                        <div className="h-6 w-6 border-2 border-white/80 border-t-transparent rounded-full animate-spin" />
                    ) : isPlaying ? (
                        <Pause className="h-6 w-6 fill-current relative z-10" />
                    ) : (
                        <Play className="h-6 w-6 fill-current ml-1 relative z-10" />
                    )}
                </button>

                {/* Waveform & Info */}
                <div className="flex-1 min-w-0 flex flex-col justify-center gap-1.5">
                    {/* Time & Title Row */}
                    <div className="flex items-center justify-between text-xs font-bold tracking-wide text-[var(--text-tertiary)] px-1 uppercase">
                        {isLoading ? (
                            <span className="animate-pulse">Loading audio... {loadingProgress}%</span>
                        ) : (
                            <>
                                <span>{formatTime(currentTime)}</span>
                                <span>-{formatTime(Math.max(0, duration - currentTime))}</span>
                            </>
                        )}
                    </div>

                    {/* Waveform Container */}
                    <div className="relative w-full group">
                        {isLoading && (
                            <div className="absolute inset-0 flex items-center justify-center z-10 bg-[var(--bg-main)]/50 backdrop-blur-sm rounded-lg">
                            </div>
                        )}
                        {/* Waveform */}
                        <div
                            ref={containerRef}
                            className={`w-full transition-all duration-300 ${collapsed ? 'h-0 opacity-0' : 'h-12 opacity-100'}`}
                        />
                    </div>
                </div>
            </div>
        </div>
    );
});

AudioPlayer.displayName = 'AudioPlayer';

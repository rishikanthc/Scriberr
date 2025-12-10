import { useEffect, useRef, useState, forwardRef, useImperativeHandle } from 'react';
import WaveSurfer from 'wavesurfer.js';
import { Play, Pause } from 'lucide-react';

import { useTheme } from '@/contexts/ThemeContext';
import { useAuth } from '@/features/auth/hooks/useAuth';

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
                // Aesthetic Gray / True Black Palette
                const waveColor = isDark ? '#525252' : '#D1D5DB'; // black-600 / gray-300
                const progressColor = isDark ? '#3b82f6' : '#2563eb'; // Blue-500 (dark) / Blue-600 (light)
                const cursorColor = isDark ? '#fe9a00' : '#fe9a00'; // Brand Orange accent for cursor

                const ws = WaveSurfer.create({
                    container: containerRef.current!,
                    waveColor,
                    progressColor,
                    cursorColor,
                    barWidth: 2,
                    barGap: 2,
                    barRadius: 2,
                    height: collapsed ? 0 : 64,
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
    }, [audioId, theme, getAuthHeaders]);

    // Update height when collapsed state changes
    useEffect(() => {
        if (wavesurferRef.current) {
            wavesurferRef.current.setOptions({
                height: collapsed ? 0 : 64
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
        // Force re-render/re-init by toggling a key or similar, 
        // but here we can just call the effect logic again if we extract it, 
        // or simpler: just unmount/remount the component from parent.
        // For now, let's just reload the page or ask user to refresh.
        // Better: we can just clear error and let the effect run again? 
        // Actually, since the effect depends on audioId, we can't easily re-trigger it without changing dependencies.
        // A simple way is to use a retry counter.
        window.location.reload(); // Simple fallback for now
    };

    if (error) {
        return (
            <div className={`transition-all duration-300 ${className} flex items-center justify-center p-8 bg-red-50 dark:bg-red-900/20 rounded-xl border border-red-200 dark:border-red-800`}>
                <div className="text-center">
                    <p className="text-red-600 dark:text-red-400 mb-2">{error}</p>
                    <button
                        onClick={retryLoad}
                        className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-md text-sm font-medium transition-colors"
                    >
                        Retry
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div className={`transition-all duration-300 ${className}`}>

            <div className="flex items-center gap-4">
                {/* Play/Pause Button */}
                <button
                    onClick={togglePlayPause}
                    disabled={isLoading}
                    className={`w-12 h-12 sm:w-14 sm:h-14 flex-shrink-0 flex items-center justify-center rounded-full bg-brand-500 hover:bg-brand-600 dark:bg-brand-500 dark:hover:bg-brand-600 text-white shadow-md hover:scale-105 hover:shadow-lg transition-all cursor-pointer border border-brand-400/20 ${isLoading ? 'opacity-50 cursor-not-allowed' : ''}`}
                >
                    {isLoading ? (
                        <div className="h-5 w-5 sm:h-6 sm:w-6 border-2 border-white border-t-transparent rounded-full animate-spin" />
                    ) : isPlaying ? (
                        <Pause className="h-5 w-5 sm:h-6 sm:w-6 fill-current" />
                    ) : (
                        <Play className="h-5 w-5 sm:h-6 sm:w-6 fill-current ml-1" />
                    )}
                </button>

                {/* Waveform & Info */}
                <div className="flex-1 min-w-0 flex flex-col justify-center gap-2">
                    {/* Time & Title Row */}
                    <div className="flex items-center justify-between text-xs sm:text-sm font-medium text-muted-foreground px-1">
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
                    <div className="relative w-full">
                        {isLoading && (
                            <div className="absolute inset-0 flex items-center justify-center z-10 bg-white/50 dark:bg-carbon-900/50 backdrop-blur-sm rounded-lg">
                                {/* Optional: Add a more prominent loader here if needed */}
                            </div>
                        )}
                        <div
                            ref={containerRef}
                            className={`w-full transition-all duration-300 ${collapsed ? 'h-0 opacity-0' : 'h-16 opacity-100'}`}
                        />
                    </div>

                    {/* Progress Bar (visible when collapsed) */}
                    {collapsed && (
                        <div className="h-1 w-full bg-carbon-200 dark:bg-carbon-800 rounded-full overflow-hidden">
                            <div
                                className="h-full bg-primary transition-all duration-100"
                                style={{ width: `${(currentTime / duration) * 100}%` }}
                            />
                        </div>
                    )}
                </div>
            </div>

            {/* Secondary Controls Row (Volume, Skip) - Only visible when expanded */}

        </div>
    );
});

AudioPlayer.displayName = 'AudioPlayer';

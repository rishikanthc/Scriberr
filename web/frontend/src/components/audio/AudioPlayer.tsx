import { useEffect, useRef, useState, forwardRef, useImperativeHandle } from 'react';
import WaveSurfer from 'wavesurfer.js';
import { Play, Pause } from 'lucide-react';

import { useTheme } from '@/contexts/ThemeContext';
import { useAuth } from '@/contexts/AuthContext';

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
    const [isPlaying, setIsPlaying] = useState(false);
    const [duration, setDuration] = useState(0);
    const [currentTime, setCurrentTime] = useState(0);
    const [isReady, setIsReady] = useState(false);

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
                const audioUrl = `/api/v1/transcription/${audioId}/audio`;
                const response = await fetch(audioUrl, { headers: { ...getAuthHeaders() } });

                if (!response.ok) throw new Error('Failed to load audio');

                const blob = await response.blob();
                const url = URL.createObjectURL(blob);

                const isDark = theme === 'dark';
                // Aesthetic Gray / True Black Palette
                const waveColor = isDark ? '#525252' : '#D1D5DB'; // black-600 / gray-300
                const progressColor = isDark ? '#E5E5E5' : '#111827'; // black-200 / gray-900 (High contrast)
                const cursorColor = isDark ? '#3b82f6' : '#2563eb'; // Blue accent for cursor

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
                });

                wavesurferRef.current = ws;

                ws.on('ready', () => {
                    setIsReady(true);
                    const dur = ws.getDuration();
                    setDuration(dur);
                    onDurationChange?.(dur);
                });

                ws.on('play', () => {
                    setIsPlaying(true);
                    onPlayStateChange?.(true);
                });

                ws.on('pause', () => {
                    setIsPlaying(false);
                    onPlayStateChange?.(false);
                });

                ws.on('audioprocess', (time) => {
                    setCurrentTime(time);
                    onTimeUpdate?.(time);
                });

                ws.on('interaction', () => {
                    const time = ws.getCurrentTime();
                    setCurrentTime(time);
                    onTimeUpdate?.(time);
                });

                ws.on('finish', () => {
                    setIsPlaying(false);
                    onPlayStateChange?.(false);
                });

                await ws.load(url);

            } catch (error) {
                console.error('Error initializing audio player:', error);
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

    return (
        <div className={`glass rounded-xl p-4 transition-all duration-300 ${className}`}>

            <div className="flex items-center gap-4">
                {/* Play/Pause Button */}
                <button
                    onClick={togglePlayPause}
                    className={`w-12 h-12 sm:w-14 sm:h-14 flex-shrink-0 flex items-center justify-center rounded-full bg-blue-600 hover:bg-blue-700 dark:bg-blue-600 dark:hover:bg-blue-500 text-white shadow-md hover:scale-105 hover:shadow-lg transition-all cursor-pointer border border-blue-500/20 ${!isReady ? 'opacity-50' : ''}`}
                >
                    {isPlaying ? (
                        <Pause className="h-5 w-5 sm:h-6 sm:w-6 fill-current" />
                    ) : (
                        <Play className="h-5 w-5 sm:h-6 sm:w-6 fill-current ml-1" />
                    )}
                </button>

                {/* Waveform & Info */}
                <div className="flex-1 min-w-0 flex flex-col justify-center gap-2">
                    {/* Time & Title Row */}
                    <div className="flex items-center justify-between text-xs sm:text-sm font-medium text-muted-foreground px-1">
                        <span>{formatTime(currentTime)}</span>
                    </div>

                    {/* Waveform Container */}
                    <div
                        ref={containerRef}
                        className={`w-full transition-all duration-300 ${collapsed ? 'h-0 opacity-0' : 'h-16 opacity-100'}`}
                    />

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

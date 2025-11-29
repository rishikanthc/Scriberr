import { useEffect, useRef, useState, forwardRef, useImperativeHandle } from 'react';
import WaveSurfer from 'wavesurfer.js';
import { Play, Pause, Volume2, VolumeX, SkipBack, SkipForward, Maximize2, Minimize2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Slider } from '@/components/ui/slider';
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
    onToggleCollapse,
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
    const [volume, setVolume] = useState(1);
    const [isMuted, setIsMuted] = useState(false);
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
                // Warm minimalist colors
                const waveColor = isDark ? '#57534e' : '#d6d3d1'; // stone-600 / stone-300
                const progressColor = isDark ? '#d97706' : '#f59e0b'; // amber-600 / amber-500
                const cursorColor = isDark ? '#f59e0b' : '#d97706';

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
                    minPxPerSec: 50,
                });

                wavesurferRef.current = ws;

                await ws.load(url);

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

    const skipForward = () => {
        wavesurferRef.current?.skip(5);
    };

    const skipBackward = () => {
        wavesurferRef.current?.skip(-5);
    };

    const handleVolumeChange = (value: number[]) => {
        const newVol = value[0];
        setVolume(newVol);
        if (newVol > 0) setIsMuted(false);
        wavesurferRef.current?.setVolume(newVol);
    };

    const toggleMute = () => {
        if (isMuted) {
            wavesurferRef.current?.setVolume(volume);
            setIsMuted(false);
        } else {
            wavesurferRef.current?.setVolume(0);
            setIsMuted(true);
        }
    };

    const formatTime = (seconds: number) => {
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    };

    return (
        <div className={`glass rounded-xl p-4 transition-all duration-300 ${className}`}>

            {/* Main Controls Row */}
            <div className="flex items-center gap-4">
                {/* Play/Pause Button - Prominent */}
                <button
                    onClick={togglePlayPause}
                    disabled={!isReady}
                    className="w-12 h-12 sm:w-14 sm:h-14 flex items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg hover:scale-105 hover:shadow-xl transition-all disabled:opacity-50 disabled:hover:scale-100 cursor-pointer"
                >
                    {isPlaying ? (
                        <Pause className="h-5 w-5 sm:h-6 sm:w-6 fill-current" />
                    ) : (
                        <Play className="h-5 w-5 sm:h-6 sm:w-6 fill-current ml-1" />
                    )}
                </button>

                {/* Waveform & Time Info */}
                <div className="flex-1 min-w-0 flex flex-col justify-center gap-2">
                    {/* Time & Title Row */}
                    <div className="flex items-center justify-between text-xs sm:text-sm font-medium text-muted-foreground px-1">
                        <span>{formatTime(currentTime)}</span>
                        <div className="flex items-center gap-2">
                            {onToggleCollapse && (
                                <button
                                    onClick={onToggleCollapse}
                                    className="p-1 hover:bg-stone-100 dark:hover:bg-stone-800 rounded-md transition-colors cursor-pointer"
                                    title={collapsed ? "Expand" : "Collapse"}
                                >
                                    {collapsed ? <Maximize2 className="h-3 w-3" /> : <Minimize2 className="h-3 w-3" />}
                                </button>
                            )}
                            <span>{formatTime(duration)}</span>
                        </div>
                    </div>

                    {/* Waveform Container */}
                    <div
                        ref={containerRef}
                        className={`w-full transition-all duration-300 ${collapsed ? 'h-0 opacity-0' : 'h-16 opacity-100'}`}
                    />

                    {/* Progress Bar (visible when collapsed) */}
                    {collapsed && (
                        <div className="h-1 w-full bg-stone-200 dark:bg-stone-700 rounded-full overflow-hidden">
                            <div
                                className="h-full bg-primary transition-all duration-100"
                                style={{ width: `${(currentTime / duration) * 100}%` }}
                            />
                        </div>
                    )}
                </div>
            </div>

            {/* Secondary Controls Row (Volume, Skip) - Only visible when expanded */}
            {!collapsed && (
                <div className="flex items-center justify-between mt-4 px-2 animate-fade-in">
                    <div className="flex items-center gap-2">
                        <Button variant="ghost" size="icon" onClick={skipBackward} className="h-8 w-8 rounded-full hover:bg-stone-100 dark:hover:bg-stone-800 cursor-pointer">
                            <SkipBack className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="icon" onClick={skipForward} className="h-8 w-8 rounded-full hover:bg-stone-100 dark:hover:bg-stone-800 cursor-pointer">
                            <SkipForward className="h-4 w-4" />
                        </Button>
                    </div>

                    <div className="flex items-center gap-2 group">
                        <button onClick={toggleMute} className="p-1.5 rounded-full hover:bg-stone-100 dark:hover:bg-stone-800 transition-colors cursor-pointer">
                            {isMuted || volume === 0 ? (
                                <VolumeX className="h-4 w-4 text-muted-foreground" />
                            ) : (
                                <Volume2 className="h-4 w-4 text-muted-foreground" />
                            )}
                        </button>
                        <div className="w-0 overflow-hidden group-hover:w-24 transition-all duration-300 ease-out">
                            <Slider
                                value={[isMuted ? 0 : volume]}
                                max={1}
                                step={0.01}
                                onValueChange={handleVolumeChange}
                                className="w-20 cursor-pointer"
                            />
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
});

AudioPlayer.displayName = 'AudioPlayer';

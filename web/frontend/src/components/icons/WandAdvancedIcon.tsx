import { Wand2 } from "lucide-react";
import { cn } from "@/lib/utils";

interface WandAdvancedIconProps {
    className?: string;
    strokeWidth?: number;
}

/**
 * A composite icon showing Wand2 with a small "+" badge
 * to indicate advanced transcription parameters
 */
export function WandAdvancedIcon({ className, strokeWidth = 2 }: WandAdvancedIconProps) {
    return (
        <div className={cn("relative inline-flex", className)}>
            <Wand2 className="h-full w-full" strokeWidth={strokeWidth} />
            {/* Small + badge in top-right corner */}
            <span
                className="absolute -top-0.5 -right-0.5 flex items-center justify-center 
					min-w-[10px] h-[10px] text-[7px] font-bold leading-none
					bg-current rounded-full"
                style={{ color: 'inherit' }}
            >
                <span className="text-white dark:text-carbon-900">+</span>
            </span>
        </div>
    );
}

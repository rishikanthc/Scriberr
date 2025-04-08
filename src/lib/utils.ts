import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

/**
 * Format seconds to MM:SS format for audio player display
 * 
 * @param seconds The time in seconds to format
 * @returns Formatted time string in MM:SS format
 */
export function formatTime(seconds: number): string {
	if (isNaN(seconds) || !isFinite(seconds)) {
		return '0:00';
	}
	
	const mins = Math.floor(seconds / 60);
	const secs = Math.floor(seconds % 60);
	return `${mins}:${secs.toString().padStart(2, '0')}`;
}

/**
 * Process LLM responses that contain thinking sections.
 * 
 * @param text The LLM response text that may contain <think>...</think> sections
 * @param mode 'remove' to completely remove thinking sections, 'process' to format them for display
 * @returns Processed text with thinking sections handled according to the specified mode
 */
export function processThinkingSections(text: string, mode: 'remove' | 'process' = 'process'): { 
	processedText: string; 
	hasThinkingSections: boolean;
	thinkingSections: string[];
} {
	if (!text) {
		return { processedText: '', hasThinkingSections: false, thinkingSections: [] };
	}

	// Regular expression to match <think>...</think> sections, including line breaks
	const thinkingPattern = /<think>([\s\S]*?)<\/think>/g;
	
	// Extract all thinking sections
	const thinkingSections: string[] = [];
	let match;
	while ((match = thinkingPattern.exec(text)) !== null) {
		thinkingSections.push(match[1].trim());
	}

	const hasThinkingSections = thinkingSections.length > 0;

	// Process the text based on the mode
	let processedText = text;
	
	if (mode === 'remove') {
		// Remove all thinking sections
		processedText = text.replace(thinkingPattern, '');
		// Clean up extra blank lines
		processedText = processedText.replace(/\n{3,}/g, '\n\n').trim();
	} else if (hasThinkingSections) {
		// Replace each thinking section with a placeholder for later display
		let index = 0;
		processedText = text.replace(thinkingPattern, () => {
			return `\n[THINKING_SECTION_${index++}]\n`;
		});
	}
	
	// Make sure we at least return the original text if there are no thinking sections
	if (processedText === '') {
		processedText = text;
	}

	return { 
		processedText, 
		hasThinkingSections, 
		thinkingSections 
	};
}
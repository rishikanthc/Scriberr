interface FileWithType {
	file: File;
	isVideo: boolean;
}

interface FileGroup {
	type: 'single' | 'video' | 'multitrack';
	files: File[];
	aupFile?: File;
	title?: string;
}

interface MultiTrackFiles {
	audioFiles: File[];
	aupFile: File;
	title: string;
}

/**
 * Detects if a file is an audio file
 */
export const isAudioFile = (file: File): boolean => {
	return file.type.startsWith('audio/') || 
		   ['.mp3', '.wav', '.flac', '.m4a', '.aac', '.ogg'].some(ext => 
			   file.name.toLowerCase().endsWith(ext)
		   );
};

/**
 * Detects if a file is a video file
 */
export const isVideoFile = (file: File): boolean => {
	return file.type.startsWith('video/') || 
		   ['.mp4', '.avi', '.mov', '.mkv', '.wmv', '.flv', '.webm'].some(ext => 
			   file.name.toLowerCase().endsWith(ext)
		   );
};

/**
 * Detects if a file is an .aup Audacity project file
 */
export const isAupFile = (file: File): boolean => {
	return file.name.toLowerCase().endsWith('.aup');
};

/**
 * Extracts a clean title from a filename by removing the extension
 */
export const extractTitle = (filename: string): string => {
	return filename.replace(/\.[^/.]+$/, '');
};

/**
 * Validates that a multi-track file set has the required components
 */
export const validateMultiTrackFiles = (files: File[]): { isValid: boolean; error?: string } => {
	const audioFiles = files.filter(isAudioFile);
	const aupFiles = files.filter(isAupFile);
	
	if (aupFiles.length === 0) {
		return { isValid: false, error: 'Multi-track uploads require an .aup Audacity project file' };
	}
	
	if (aupFiles.length > 1) {
		return { isValid: false, error: 'Only one .aup file is allowed per multi-track upload' };
	}
	
	if (audioFiles.length === 0) {
		return { isValid: false, error: 'Multi-track uploads require at least one audio file' };
	}
	
	return { isValid: true };
};

/**
 * Groups dropped files into appropriate categories for processing
 */
export const groupFiles = (files: File[]): FileGroup => {
	const audioFiles = files.filter(isAudioFile);
	const videoFiles = files.filter(isVideoFile);
	const aupFiles = files.filter(isAupFile);
	
	// Multi-track mode if .aup file is present
	if (aupFiles.length > 0) {
		const title = extractTitle(aupFiles[0].name);
		return {
			type: 'multitrack',
			files: audioFiles,
			aupFile: aupFiles[0],
			title
		};
	}
	
	// Video files take precedence over audio files
	if (videoFiles.length > 0) {
		return {
			type: 'video',
			files: videoFiles
		};
	}
	
	// Regular audio files
	if (audioFiles.length > 0) {
		return {
			type: 'single',
			files: audioFiles
		};
	}
	
	// No supported files found
	return {
		type: 'single',
		files: []
	};
};

/**
 * Converts regular files to FileWithType format for video files
 */
export const convertToFileWithType = (files: File[], isVideo: boolean = false): FileWithType[] => {
	return files.map(file => ({
		file,
		isVideo
	}));
};

/**
 * Prepares multi-track files for upload
 */
export const prepareMultiTrackFiles = (fileGroup: FileGroup): MultiTrackFiles | null => {
	if (fileGroup.type !== 'multitrack' || !fileGroup.aupFile) {
		return null;
	}
	
	return {
		audioFiles: fileGroup.files,
		aupFile: fileGroup.aupFile,
		title: fileGroup.title || extractTitle(fileGroup.aupFile.name)
	};
};

/**
 * Gets a human-readable description of the files being processed
 */
export const getFileDescription = (fileGroup: FileGroup): string => {
	const count = fileGroup.files.length;
	
	switch (fileGroup.type) {
		case 'multitrack':
			return `Multi-track project: ${fileGroup.title} (${count} audio tracks)`;
		case 'video':
			return count === 1 ? '1 video file' : `${count} video files`;
		case 'single':
			return count === 1 ? '1 audio file' : `${count} audio files`;
		default:
			return 'Unknown file type';
	}
};

/**
 * Gets the appropriate icon name for the file group type
 */
export const getFileGroupIcon = (fileGroup: FileGroup): string => {
	switch (fileGroup.type) {
		case 'multitrack':
			return 'Users'; // Multiple people icon for multi-track
		case 'video':
			return 'Video';
		case 'single':
			return 'FileAudio';
		default:
			return 'File';
	}
};

/**
 * Checks if the file group contains any supported files
 */
export const hasValidFiles = (fileGroup: FileGroup): boolean => {
	if (fileGroup.type === 'multitrack') {
		const validation = validateMultiTrackFiles([...fileGroup.files, fileGroup.aupFile!]);
		return validation.isValid;
	}
	
	return fileGroup.files.length > 0;
};
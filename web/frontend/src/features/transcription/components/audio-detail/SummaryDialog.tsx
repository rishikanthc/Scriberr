import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import {
    Command,
    CommandEmpty,
    CommandGroup,
    CommandInput,
    CommandItem,
    CommandList
} from "@/components/ui/command";
import {
    Popover,
    PopoverContent,
    PopoverTrigger
} from "@/components/ui/popover";
import { Label } from "@/components/ui/label";
import { useToast } from "@/components/ui/toast";
import { useState, useEffect } from "react";
import ReactMarkdown from 'react-markdown';
import remarkMath from 'remark-math';
import rehypeRaw from 'rehype-raw';
import rehypeKatex from 'rehype-katex';
import rehypeHighlight from 'rehype-highlight';
import { useSummaryTemplates, useSummarizer, useExistingSummary } from "@/features/transcription/hooks/useTranscriptionSummary";

import { useTranscript, useAudioDetail } from "@/features/transcription/hooks/useAudioDetail";

interface SummaryDialogProps {
    audioId: string;
    isOpen: boolean;
    onClose: (open: boolean) => void;
    llmReady: boolean | null;
}

export function SummaryDialog({ audioId, isOpen, onClose, llmReady }: SummaryDialogProps) {
    const { toast } = useToast();
    const { data: templates = [], isLoading: templatesLoading } = useSummaryTemplates();
    const { data: existingSummary } = useExistingSummary(audioId);
    const { data: transcript } = useTranscript(audioId, true);
    const { data: audioFile } = useAudioDetail(audioId);

    // Hooks
    const { generateSummary, isStreaming, streamContent, error } = useSummarizer(audioId);

    // State
    const [selectedTemplateId, setSelectedTemplateId] = useState<string>("");
    const [tplPopoverOpen, setTplPopoverOpen] = useState(false);
    const [showOutput, setShowOutput] = useState(false);

    const selectedTemplate = templates.find(t => t.id === selectedTemplateId);

    // Auto-show existing summary if available and not streaming
    useEffect(() => {
        if (isOpen && existingSummary && !isStreaming && !streamContent) {
            setShowOutput(true);
        }
    }, [isOpen, existingSummary, isStreaming, streamContent]);

    const handleStartSummary = () => {
        if (!selectedTemplate || !transcript) return;

        setShowOutput(true);
        const transcriptText = transcript.text || '';
        generateSummary(selectedTemplate.id, selectedTemplate.model, selectedTemplate.prompt, transcriptText);
    };

    const handleCopy = async () => {
        const content = streamContent || existingSummary?.content || "";
        if (content) {
            await navigator.clipboard.writeText(content);
            toast({ title: 'Copied to clipboard' });
        }
    };

    const handleDownload = () => {
        const content = streamContent || existingSummary?.content || "";
        if (!content) return;

        const title = audioFile?.title || "summary";
        const filename = `${title.replace(/[^a-z0-9]/gi, '_').toLowerCase()}-summary.md`;

        const blob = new Blob([content], { type: 'text/markdown' });
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = filename;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(url);
    };

    if (showOutput) {
        // Output View
        return (
            <Dialog open={isOpen} onOpenChange={onClose}>
                <DialogContent className="sm:max-w-3xl bg-white dark:bg-carbon-800 border-carbon-200 dark:border-carbon-700 max-h-[85vh] overflow-y-auto">
                    <DialogHeader>
                        <DialogTitle className="text-carbon-900 dark:text-carbon-100">Summary</DialogTitle>
                        <DialogDescription className="flex items-center gap-2 text-carbon-600 dark:text-carbon-400">
                            {isStreaming ? (
                                <>
                                    <span>Generating summary...</span>
                                    <span className="inline-block h-3.5 w-3.5 border-2 border-carbon-600 border-t-transparent rounded-full animate-spin" aria-label="Loading" />
                                </>
                            ) : (
                                <span>Summary {error ? 'failed' : 'ready'}</span>
                            )}
                        </DialogDescription>
                    </DialogHeader>

                    <div className="flex items-center justify-end gap-2 mb-2">
                        <button
                            className="px-2.5 py-1.5 rounded-md bg-carbon-900 text-white text-sm hover:bg-carbon-950 transition-colors"
                            onClick={() => {
                                setShowOutput(false);
                                setSelectedTemplateId('');
                            }}
                            disabled={isStreaming}
                        >
                            Regenerate
                        </button>
                        <button
                            className="px-2.5 py-1.5 rounded-md bg-carbon-200 dark:bg-carbon-700 text-sm"
                            onClick={handleCopy}
                            disabled={!streamContent && !existingSummary?.content}
                        >
                            Copy Text
                        </button>
                        <button
                            className="px-2.5 py-1.5 rounded-md bg-carbon-200 dark:bg-carbon-700 text-sm"
                            onClick={handleDownload}
                            disabled={!streamContent && !existingSummary?.content}
                        >
                            Download .md
                        </button>
                    </div>

                    <div className="prose prose-gray dark:prose-invert max-w-none min-h-[200px]">
                        {error ? (
                            <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
                        ) : (
                            <ReactMarkdown
                                remarkPlugins={[remarkMath]}
                                rehypePlugins={[rehypeRaw as any, rehypeKatex as any, rehypeHighlight as any]}
                            >
                                {streamContent || existingSummary?.content || ""}
                            </ReactMarkdown>
                        )}
                        {!error && !streamContent && !existingSummary?.content && isStreaming && (
                            <p className="text-sm text-carbon-500">Generating summary...</p>
                        )}
                    </div>
                </DialogContent>
            </Dialog>
        );
    }

    // Template Selector View
    return (
        <Dialog open={isOpen} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-lg bg-white dark:bg-carbon-800 border-carbon-200 dark:border-carbon-700">
                <DialogHeader>
                    <DialogTitle className="text-carbon-900 dark:text-carbon-100">Summarize Transcript</DialogTitle>
                    <DialogDescription className="text-carbon-600 dark:text-carbon-400">Choose a summarization template</DialogDescription>
                </DialogHeader>

                {llmReady === false && (
                    <div className="p-3 bg-amber-50 dark:bg-amber-900/20 text-amber-800 dark:text-amber-200 rounded-md text-sm mb-2">
                        LLM is not configured or active. Please check settings.
                    </div>
                )}

                <div className="py-2 space-y-3">
                    <div className="space-y-1">
                        <Label className="text-sm text-carbon-700 dark:text-carbon-300">Template</Label>
                        <Popover open={tplPopoverOpen} onOpenChange={setTplPopoverOpen}>
                            <PopoverTrigger asChild>
                                <button
                                    className="w-full inline-flex justify-between items-center rounded-md border border-carbon-300 dark:border-carbon-600 bg-white dark:bg-carbon-800 px-3 py-2 text-sm text-carbon-900 dark:text-carbon-100 hover:bg-carbon-50 dark:hover:bg-carbon-700 cursor-pointer"
                                    aria-label="Choose template"
                                    disabled={!llmReady}
                                >
                                    <span className="truncate text-left">{selectedTemplate ? selectedTemplate.name : (templatesLoading ? 'Loading...' : 'Select a template')}</span>
                                    <span className="text-xs text-carbon-500 ml-2 truncate">{selectedTemplate?.model ? `(${selectedTemplate.model})` : ''}</span>
                                </button>
                            </PopoverTrigger>
                            <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-0 bg-white dark:bg-carbon-800 border border-carbon-200 dark:border-carbon-700">
                                <Command>
                                    <CommandInput placeholder="Search templates..." />
                                    <CommandList className="max-h-64 overflow-auto">
                                        <CommandEmpty>{templatesLoading ? 'Loading...' : 'No templates found'}</CommandEmpty>
                                        <CommandGroup heading="Templates">
                                            {templates.map(t => (
                                                <CommandItem
                                                    key={t.id}
                                                    value={t.name}
                                                    onSelect={() => { setSelectedTemplateId(t.id); setTplPopoverOpen(false); }}
                                                >
                                                    <div className="flex flex-col">
                                                        <span className="text-sm">{t.name}</span>
                                                        <span className="text-xs text-carbon-500">Model: {t.model || '—'}</span>
                                                    </div>
                                                </CommandItem>
                                            ))}
                                        </CommandGroup>
                                    </CommandList>
                                </Command>
                            </PopoverContent>
                        </Popover>

                        {!templatesLoading && templates.length === 0 && (
                            <p className="text-xs text-carbon-500">No templates. Create one in Settings → Summary.</p>
                        )}
                        {selectedTemplate && !selectedTemplate.model && (
                            <p className="text-xs text-red-600">Selected template has no model configured. Edit it in Settings.</p>
                        )}
                    </div>

                    <div className="mt-1 flex items-center justify-end gap-2">
                        <button className="px-3 py-1.5 rounded-md bg-carbon-200 dark:bg-carbon-700" onClick={() => onClose(false)}>Cancel</button>
                        <button
                            className="px-3 py-1.5 rounded-md bg-carbon-900 text-white disabled:opacity-50 hover:bg-carbon-950 transition-colors"
                            disabled={!selectedTemplateId || !selectedTemplate?.model || !llmReady}
                            onClick={handleStartSummary}
                        >
                            Summarize
                        </button>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
}

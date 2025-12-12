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

import { Sparkles, Download, Copy, RefreshCw, ChevronDown } from "lucide-react";

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
                <DialogContent className="sm:max-w-3xl bg-[var(--bg-card)] border-[var(--border-subtle)] shadow-[var(--shadow-float)] max-h-[85vh] overflow-y-auto">
                    <DialogHeader className="border-b border-[var(--border-subtle)] pb-4">
                        <DialogTitle className="text-[var(--text-primary)] flex items-center gap-2">
                            <Sparkles className="h-5 w-5 text-[var(--brand-solid)]" />
                            Summary
                        </DialogTitle>
                        <DialogDescription className="flex items-center gap-2 text-[var(--text-secondary)]">
                            {isStreaming ? (
                                <>
                                    <span>Generating summary...</span>
                                    <span className="inline-block h-3.5 w-3.5 border-2 border-[var(--brand-solid)] border-t-transparent rounded-full animate-spin" aria-label="Loading" />
                                </>
                            ) : (
                                <span>Summary {error ? 'failed' : 'ready'}</span>
                            )}
                        </DialogDescription>
                    </DialogHeader>

                    <div className="flex items-center justify-end gap-2 mb-2 pt-2">
                        <button
                            className="inline-flex items-center gap-2 px-3 py-1.5 rounded-[var(--radius-btn)] bg-[var(--bg-main)] border border-[var(--border-subtle)] text-[var(--text-primary)] text-sm hover:bg-[var(--bg-subtle)] transition-colors transition-shadow disabled:opacity-50"
                            onClick={() => {
                                setShowOutput(false);
                                setSelectedTemplateId('');
                            }}
                            disabled={isStreaming}
                        >
                            <RefreshCw className="h-3.5 w-3.5" />
                            Regenerate
                        </button>
                        <button
                            className="inline-flex items-center gap-2 px-3 py-1.5 rounded-[var(--radius-btn)] bg-[var(--bg-main)] border border-[var(--border-subtle)] text-[var(--text-primary)] text-sm hover:bg-[var(--bg-subtle)] transition-colors transition-shadow disabled:opacity-50"
                            onClick={handleCopy}
                            disabled={!streamContent && !existingSummary?.content}
                        >
                            <Copy className="h-3.5 w-3.5" />
                            Copy
                        </button>
                        <button
                            className="inline-flex items-center gap-2 px-3 py-1.5 rounded-[var(--radius-btn)] bg-[var(--bg-main)] border border-[var(--border-subtle)] text-[var(--text-primary)] text-sm hover:bg-[var(--bg-subtle)] transition-colors transition-shadow disabled:opacity-50"
                            onClick={handleDownload}
                            disabled={!streamContent && !existingSummary?.content}
                        >
                            <Download className="h-3.5 w-3.5" />
                            Download
                        </button>
                    </div>

                    <div className="prose prose-stone dark:prose-invert max-w-none min-h-[200px] p-4 bg-[var(--bg-main)] rounded-[var(--radius-card)] border border-[var(--border-subtle)]">
                        {error ? (
                            <p className="text-sm text-[var(--error)]">{error}</p>
                        ) : (
                            <ReactMarkdown
                                remarkPlugins={[remarkMath]}
                                rehypePlugins={[rehypeRaw as any, rehypeKatex as any, rehypeHighlight as any]}
                                components={{
                                    // Override typography colors if prose doesn't handle vars well
                                    p: ({ node, ...props }) => <p className="text-[var(--text-secondary)] leading-7" {...props} />,
                                    h1: ({ node, ...props }) => <h1 className="text-[var(--text-primary)] font-bold text-2xl mt-6 mb-4" {...props} />,
                                    h2: ({ node, ...props }) => <h2 className="text-[var(--text-primary)] font-bold text-xl mt-6 mb-3" {...props} />,
                                    h3: ({ node, ...props }) => <h3 className="text-[var(--text-primary)] font-bold text-lg mt-5 mb-2" {...props} />,
                                    li: ({ node, ...props }) => <li className="text-[var(--text-secondary)]" {...props} />,
                                    strong: ({ node, ...props }) => <strong className="text-[var(--text-primary)] font-bold" {...props} />,
                                }}
                            >
                                {streamContent || existingSummary?.content || ""}
                            </ReactMarkdown>
                        )}
                        {!error && !streamContent && !existingSummary?.content && isStreaming && (
                            <p className="text-sm text-[var(--text-tertiary)] italic">Generating summary...</p>
                        )}
                    </div>
                </DialogContent>
            </Dialog>
        );
    }

    // Template Selector View
    return (
        <Dialog open={isOpen} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-lg bg-[var(--bg-card)] border-[var(--border-subtle)] shadow-[var(--shadow-float)]">
                <DialogHeader>
                    <DialogTitle className="text-[var(--text-primary)]">Summarize Transcript</DialogTitle>
                    <DialogDescription className="text-[var(--text-secondary)]">Choose a summarization template to generate insights.</DialogDescription>
                </DialogHeader>

                {llmReady === false && (
                    <div className="p-3 bg-[var(--warning-translucent)] text-[var(--warning-solid)] border border-[var(--warning-solid)]/20 rounded-[var(--radius-card)] text-sm mb-2">
                        LLM is not configured or active. Please check settings.
                    </div>
                )}

                <div className="py-4 space-y-4">
                    <div className="space-y-1.5 local-form-group">
                        <Label className="text-sm font-medium text-[var(--text-secondary)]">Template</Label>
                        <Popover open={tplPopoverOpen} onOpenChange={setTplPopoverOpen}>
                            <PopoverTrigger asChild>
                                <button
                                    className="w-full inline-flex justify-between items-center rounded-[var(--radius-card)] border border-[var(--border-subtle)] bg-[var(--bg-main)] px-3 py-2.5 text-sm text-[var(--text-primary)] hover:border-[var(--brand-solid)]/50 focus:ring-2 focus:ring-[var(--brand-solid)]/20 transition-all outline-none disabled:opacity-50"
                                    aria-label="Choose template"
                                    disabled={!llmReady}
                                >
                                    <span className="truncate text-left">{selectedTemplate ? selectedTemplate.name : (templatesLoading ? 'Loading...' : 'Select a template')}</span>
                                    <span className="flex items-center text-xs text-[var(--text-tertiary)] ml-2 truncate">
                                        {selectedTemplate?.model ? `(${selectedTemplate.model})` : ''}
                                        <ChevronDown className="ml-2 h-4 w-4 opacity-50" />
                                    </span>
                                </button>
                            </PopoverTrigger>
                            <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-1 bg-[var(--bg-card)] border border-[var(--border-subtle)] shadow-xl rounded-[var(--radius-card)]">
                                <Command className="bg-transparent">
                                    <CommandInput placeholder="Search templates..." className="border-none focus:ring-0" />
                                    <CommandList className="max-h-64 overflow-auto p-1">
                                        <CommandEmpty className="py-2 text-center text-xs text-[var(--text-tertiary)]">{templatesLoading ? 'Loading...' : 'No templates found'}</CommandEmpty>
                                        <CommandGroup heading="Templates" className="text-[var(--text-tertiary)]">
                                            {templates.map(t => (
                                                <CommandItem
                                                    key={t.id}
                                                    value={t.name}
                                                    onSelect={() => { setSelectedTemplateId(t.id); setTplPopoverOpen(false); }}
                                                    className="rounded-sm aria-selected:bg-[var(--brand-solid)] aria-selected:text-white cursor-pointer"
                                                >
                                                    <div className="flex flex-col w-full">
                                                        <span className="text-sm font-medium">{t.name}</span>
                                                        <span className="text-xs opacity-70">Model: {t.model || '—'}</span>
                                                    </div>
                                                </CommandItem>
                                            ))}
                                        </CommandGroup>
                                    </CommandList>
                                </Command>
                            </PopoverContent>
                        </Popover>

                        {!templatesLoading && templates.length === 0 && (
                            <p className="text-xs text-[var(--text-tertiary)] pl-1">No templates found. Go to Settings → Summary to create one.</p>
                        )}
                        {selectedTemplate && !selectedTemplate.model && (
                            <p className="text-xs text-[var(--error)] pl-1">Selected template has no model configured.</p>
                        )}
                    </div>

                    <div className="mt-4 flex items-center justify-end gap-3 pt-2">
                        <button
                            className="px-4 py-2 rounded-[var(--radius-btn)] text-[var(--text-secondary)] hover:bg-[var(--bg-main)] transition-colors text-sm font-medium"
                            onClick={() => onClose(false)}
                        >
                            Cancel
                        </button>
                        <button
                            className="px-4 py-2 rounded-[var(--radius-btn)] shadow-lg shadow-orange-500/20 text-white text-sm font-medium transition-all hover:scale-[1.02] active:scale-[0.98] disabled:opacity-50 disabled:pointer-events-none"
                            style={{ background: 'var(--brand-gradient)' }}
                            disabled={!selectedTemplateId || !selectedTemplate?.model || !llmReady}
                            onClick={handleStartSummary}
                        >
                            Generate Summary
                        </button>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
}

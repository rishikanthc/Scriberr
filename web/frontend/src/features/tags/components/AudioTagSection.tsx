import { useMemo, useState } from "react";
import { Check, Plus, X } from "lucide-react";
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { IconButton } from "@/shared/ui/Button";
import type { AudioTag } from "@/features/tags/api/tagsApi";
import { useRemoveTranscriptionTag, useReplaceTranscriptionTags, useTags, useTranscriptionTags } from "@/features/tags/hooks/useTags";

type AudioTagSectionProps = {
  transcriptionId?: string;
  enabled: boolean;
};

export function AudioTagSection({ transcriptionId, enabled }: AudioTagSectionProps) {
  const [pickerOpen, setPickerOpen] = useState(false);
  const allTagsQuery = useTags();
  const assignedTagsQuery = useTranscriptionTags(transcriptionId, enabled && Boolean(transcriptionId));
  const replaceTagsMutation = useReplaceTranscriptionTags(transcriptionId || "");
  const removeTagMutation = useRemoveTranscriptionTag(transcriptionId || "");
  const allTags = allTagsQuery.data?.items || [];
  const assignedTags = assignedTagsQuery.data?.items || [];
  const assignedIds = useMemo(() => new Set(assignedTags.map((tag) => tag.id)), [assignedTags]);

  const toggleTag = async (tag: AudioTag) => {
    if (!transcriptionId || replaceTagsMutation.isPending) return;
    const nextIds = assignedIds.has(tag.id)
      ? assignedTags.filter((item) => item.id !== tag.id).map((item) => item.id)
      : [...assignedTags.map((item) => item.id), tag.id];
    await replaceTagsMutation.mutateAsync(nextIds);
  };

  const removeTag = (tagId: string) => {
    if (!transcriptionId || removeTagMutation.isPending) return;
    removeTagMutation.mutate(tagId);
  };

  if (!enabled || !transcriptionId) return null;

  return (
    <section className="scr-audio-tags-section" aria-label="Tags">
      <div className="scr-audio-tags-heading">
        <h2>Tags</h2>
        <Popover open={pickerOpen} onOpenChange={setPickerOpen}>
          <PopoverTrigger asChild>
            <IconButton label="Add tags" className="scr-audio-tags-add" disabled={allTagsQuery.isLoading || replaceTagsMutation.isPending}>
              <Plus size={15} aria-hidden="true" />
            </IconButton>
          </PopoverTrigger>
          <PopoverContent className="scr-tag-picker" align="start" side="bottom">
            <Command>
              <CommandInput placeholder="Search tags" />
              <CommandList>
                <CommandEmpty>{allTagsQuery.isLoading ? "Loading tags." : "No tags found."}</CommandEmpty>
                <CommandGroup heading="Available tags">
                  {allTags.map((tag) => {
                    const selected = assignedIds.has(tag.id);
                    return (
                      <CommandItem
                        key={tag.id}
                        value={`${tag.name} ${tag.description || ""} ${tag.when_to_use || ""}`}
                        onSelect={() => void toggleTag(tag)}
                      >
                        <span className="scr-tag-picker-option">
                          <span className="scr-tag-picker-check" data-selected={selected}>
                            {selected ? <Check size={13} aria-hidden="true" /> : null}
                          </span>
                          <span>{tag.name}</span>
                        </span>
                      </CommandItem>
                    );
                  })}
                </CommandGroup>
              </CommandList>
            </Command>
          </PopoverContent>
        </Popover>
      </div>

      {assignedTagsQuery.isLoading ? (
        <p className="scr-audio-tags-empty">Loading tags.</p>
      ) : assignedTags.length > 0 ? (
        <div className="scr-audio-tags-list">
          {assignedTags.map((tag) => (
            <span className="scr-audio-tag-chip" key={tag.id}>
              {tag.name}
              <button type="button" aria-label={`Remove ${tag.name}`} onClick={() => removeTag(tag.id)}>
                <X size={12} aria-hidden="true" />
              </button>
            </span>
          ))}
        </div>
      ) : (
        <p className="scr-audio-tags-empty">No tags assigned.</p>
      )}
    </section>
  );
}

export type AudioTag = {
  id: string;
  name: string;
  color: string | null;
  description: string | null;
  when_to_use: string | null;
  created_at: string;
  updated_at: string;
};

export type TagsResponse = {
  items: AudioTag[];
  next_cursor: string | null;
};

export type SaveTagPayload = {
  id?: string;
  name: string;
  color?: string | null;
  description?: string | null;
  when_to_use?: string | null;
};

export async function listTags(headers: Record<string, string>): Promise<TagsResponse> {
  const items: AudioTag[] = [];
  let nextCursor: string | null = null;

  do {
    const params = new URLSearchParams({ limit: "100" });
    if (nextCursor) params.set("cursor", nextCursor);

    const response = await fetch(`/api/v1/tags?${params.toString()}`, { headers });
    if (!response.ok) throw new Error(await readError(response));

    const page = await response.json() as TagsResponse;
    items.push(...page.items);
    nextCursor = page.next_cursor;
  } while (nextCursor);

  return { items, next_cursor: null };
}

export async function createTag(payload: SaveTagPayload, headers: Record<string, string>): Promise<AudioTag> {
  const response = await fetch("/api/v1/tags", {
    method: "POST",
    headers: jsonHeaders(headers),
    body: JSON.stringify(tagPayload(payload)),
  });
  if (!response.ok) throw new Error(await readError(response));
  return response.json() as Promise<AudioTag>;
}

export async function updateTag(payload: SaveTagPayload & { id: string }, headers: Record<string, string>): Promise<AudioTag> {
  const response = await fetch(`/api/v1/tags/${payload.id}`, {
    method: "PATCH",
    headers: jsonHeaders(headers),
    body: JSON.stringify(tagPayload(payload)),
  });
  if (!response.ok) throw new Error(await readError(response));
  return response.json() as Promise<AudioTag>;
}

export async function deleteTag(tagId: string, headers: Record<string, string>): Promise<void> {
  const response = await fetch(`/api/v1/tags/${tagId}`, {
    method: "DELETE",
    headers,
  });
  if (!response.ok) throw new Error(await readError(response));
}

export async function listTranscriptionTags(transcriptionId: string, headers: Record<string, string>): Promise<TagsResponse> {
  const response = await fetch(`/api/v1/transcriptions/${transcriptionId}/tags`, { headers });
  if (!response.ok) throw new Error(await readError(response));
  return response.json() as Promise<TagsResponse>;
}

export async function replaceTranscriptionTags(transcriptionId: string, tagIds: string[], headers: Record<string, string>): Promise<TagsResponse> {
  const response = await fetch(`/api/v1/transcriptions/${transcriptionId}/tags`, {
    method: "PUT",
    headers: jsonHeaders(headers),
    body: JSON.stringify({ tag_ids: tagIds }),
  });
  if (!response.ok) throw new Error(await readError(response));
  return response.json() as Promise<TagsResponse>;
}

export async function addTranscriptionTag(transcriptionId: string, tagId: string, headers: Record<string, string>): Promise<TagsResponse> {
  const response = await fetch(`/api/v1/transcriptions/${transcriptionId}/tags/${tagId}`, {
    method: "POST",
    headers,
  });
  if (!response.ok) throw new Error(await readError(response));
  return response.json() as Promise<TagsResponse>;
}

export async function removeTranscriptionTag(transcriptionId: string, tagId: string, headers: Record<string, string>): Promise<void> {
  const response = await fetch(`/api/v1/transcriptions/${transcriptionId}/tags/${tagId}`, {
    method: "DELETE",
    headers,
  });
  if (!response.ok) throw new Error(await readError(response));
}

export function normalizeTagName(name: string) {
  return name.trim().toLowerCase().replace(/\s+/g, " ");
}

export function hasDuplicateTagName(tags: AudioTag[], name: string, excludingTagId?: string) {
  const normalized = normalizeTagName(name);
  if (!normalized) return false;
  return tags.some((tag) => tag.id !== excludingTagId && normalizeTagName(tag.name) === normalized);
}

function tagPayload(payload: SaveTagPayload) {
  return {
    name: payload.name,
    color: payload.color || null,
    description: payload.description || null,
    when_to_use: payload.when_to_use || null,
  };
}

function jsonHeaders(headers: Record<string, string>) {
  return {
    ...headers,
    "Content-Type": "application/json",
  };
}

async function readError(response: Response) {
  try {
    const data = await response.json();
    return data?.error?.message || data?.message || response.statusText;
  } catch {
    return response.statusText;
  }
}

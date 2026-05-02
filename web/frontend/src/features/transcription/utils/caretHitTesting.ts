export type TranscriptCaretPoint = {
  clientX: number;
  clientY: number;
  target?: EventTarget | null;
};

export type TranscriptCaretHit = {
  textElement: HTMLElement;
  textNode: Text;
  nodeOffset: number;
  textOffset: number;
  segmentIndex: number;
};

type CaretPositionLike = {
  offsetNode: Node;
  offset: number;
};

type CaretDocument = Document & {
  caretPositionFromPoint?: (x: number, y: number) => CaretPositionLike | null;
  caretRangeFromPoint?: (x: number, y: number) => Range | null;
};

const interactiveSelector = [
  "a[href]",
  "button",
  "input",
  "textarea",
  "select",
  "summary",
  "[role='button']",
  "[role='menuitem']",
  "[contenteditable='true']",
].join(",");

export function resolveTranscriptCaretHit(root: HTMLElement, point: TranscriptCaretPoint): TranscriptCaretHit | null {
  if (isInteractiveNonTranscriptTarget(point.target)) return null;

  const caret = caretFromPoint(root.ownerDocument, point.clientX, point.clientY);
  if (!caret || !root.contains(caret.node)) return null;

  const resolved = resolveTextNodeAtOffset(caret.node, caret.offset);
  if (!resolved) return null;

  const textElement = transcriptTextElementForNode(resolved.textNode);
  if (!textElement || !root.contains(textElement)) return null;

  const segmentIndex = Number(textElement.dataset.transcriptSegmentIndex);
  if (!Number.isInteger(segmentIndex)) return null;

  return {
    textElement,
    textNode: resolved.textNode,
    nodeOffset: resolved.offset,
    textOffset: textNodeOffsetInside(textElement, resolved.textNode) + resolved.offset,
    segmentIndex,
  };
}

export function transcriptTextElementForNode(node: Node) {
  const parent = node.nodeType === Node.TEXT_NODE ? node.parentElement : node instanceof HTMLElement ? node : null;
  return parent?.closest<HTMLElement>("[data-transcript-text]") || null;
}

export function textNodeOffsetInside(root: HTMLElement, target: Text) {
  const walker = root.ownerDocument.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  let offset = 0;
  let node = walker.nextNode();

  while (node) {
    if (node === target) return offset;
    offset += (node as Text).length;
    node = walker.nextNode();
  }

  return 0;
}

function caretFromPoint(document: Document, clientX: number, clientY: number) {
  const caretDocument = document as CaretDocument;
  const position = caretDocument.caretPositionFromPoint?.(clientX, clientY);
  if (position) {
    return { node: position.offsetNode, offset: position.offset };
  }

  const range = caretDocument.caretRangeFromPoint?.(clientX, clientY);
  if (!range) return null;
  return { node: range.startContainer, offset: range.startOffset };
}

function resolveTextNodeAtOffset(node: Node, offset: number): { textNode: Text; offset: number } | null {
  if (node.nodeType === Node.TEXT_NODE) {
    const textNode = node as Text;
    return {
      textNode,
      offset: Math.max(0, Math.min(offset, textNode.length)),
    };
  }

  if (!(node instanceof HTMLElement)) return null;

  if (node.childNodes.length === 0) return null;

  if (offset >= node.childNodes.length) {
    const textNode = lastTextNode(node.childNodes[node.childNodes.length - 1]);
    return textNode ? { textNode, offset: textNode.length } : null;
  }

  const textNode = firstTextNode(node.childNodes[Math.max(0, offset)]);
  return textNode ? { textNode, offset: 0 } : null;
}

function firstTextNode(node: Node): Text | null {
  if (node.nodeType === Node.TEXT_NODE) return node as Text;

  const walker = documentForNode(node).createTreeWalker(node, NodeFilter.SHOW_TEXT, {
    acceptNode: (candidate) => candidate.textContent ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_REJECT,
  });
  return walker.nextNode() as Text | null;
}

function lastTextNode(node: Node): Text | null {
  if (node.nodeType === Node.TEXT_NODE) return node as Text;

  const walker = documentForNode(node).createTreeWalker(node, NodeFilter.SHOW_TEXT, {
    acceptNode: (candidate) => candidate.textContent ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_REJECT,
  });
  let last: Text | null = null;
  let next = walker.nextNode();
  while (next) {
    last = next as Text;
    next = walker.nextNode();
  }
  return last;
}

function documentForNode(node: Node) {
  return node.ownerDocument || document;
}

function isInteractiveNonTranscriptTarget(target: EventTarget | null | undefined) {
  if (!(target instanceof Element)) return false;
  if (target.closest("[data-transcript-text]")) return false;
  return Boolean(target.closest(interactiveSelector));
}

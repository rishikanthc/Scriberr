import { useEffect, useRef } from "react";
import { TextforgeEditor } from "textforge";
import "textforge/textforge.css";
import "./ReadOnlyMarkdown.css";

type TextforgeInstance = InstanceType<typeof TextforgeEditor>;

export function ReadOnlyMarkdown({ content }: { content: string }) {
  const hostRef = useRef<HTMLDivElement | null>(null);
  const editorRef = useRef<TextforgeInstance | null>(null);

  useEffect(() => {
    if (!hostRef.current) return;
    const editor = new TextforgeEditor({
      element: hostRef.current,
      content,
      contentType: "markdown",
      editable: false,
      className: "scr-textforge-readonly",
      features: {
        images: false,
        iframeEmbeds: false,
        mentions: false,
      },
    });
    editorRef.current = editor;
    return () => {
      editor.destroy();
      editorRef.current = null;
    };
  }, []);

  useEffect(() => {
    editorRef.current?.setContent(content, "markdown");
  }, [content]);

  return <div className="scr-readonly-markdown" ref={hostRef} />;
}

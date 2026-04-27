type EmptyStateProps = {
  title: string;
  description?: string;
};

export function EmptyState({ title, description }: EmptyStateProps) {
  return (
    <div className="scr-empty">
      <div>
        <p className="scr-page-title">{title}</p>
        {description ? <p className="scr-page-meta">{description}</p> : null}
      </div>
    </div>
  );
}

type Props = {
  repoUrl: string;
  count?: number | string;
  className?: string;
};

export default function GitHubStar({ repoUrl, count, className }: Props) {
  return (
    <a
      href={repoUrl}
      target="_blank"
      rel="noreferrer"
      className={`inline-flex items-center gap-2 rounded-md border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-900 hover:bg-gray-50 active:bg-gray-100 ${className ?? ''}`}
      aria-label="Star on GitHub"
    >
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" className="size-4 text-amber-500">
        <path d="M11.48 3.499a.562.562 0 0 1 1.04 0l2.125 5.111a.563.563 0 0 0 .475.345l5.518.401c.499.036.701.663.321.988l-4.204 3.57a.563.563 0 0 0-.182.557l1.285 5.385a.562.562 0 0 1-.84.61l-4.725-2.885a.563.563 0 0 0-.586 0L6.983 20.466a.562.562 0 0 1-.84-.61l1.285-5.386a.562.562 0 0 0-.182-.557l-4.204-3.57a.562.562 0 0 1 .321-.988l5.518-.4a.563.563 0 0 0 .475-.346l2.125-5.111Z" />
      </svg>
      <span>Star</span>
      {count !== undefined && (
        <span className="ml-1 rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-700">{count}</span>
      )}
    </a>
  );
}


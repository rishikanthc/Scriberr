import { useEffect, useState } from 'react';

type RepoInfo = { stargazers_count: number; forks_count: number; html_url: string; full_name: string };

export default function GithubBadge() {
  const [info, setInfo] = useState<RepoInfo | null>(null);
  const href = 'https://github.com/rishikanthc/scriberr';

  useEffect(() => {
    let alive = true;
    fetch('https://api.github.com/repos/rishikanthc/scriberr', {
      headers: { 'Accept': 'application/vnd.github+json' },
    })
      .then((r) => (r.ok ? r.json() : null))
      .then((j) => {
        if (!alive || !j) return;
        setInfo({
          stargazers_count: j.stargazers_count ?? 0,
          forks_count: j.forks_count ?? 0,
          html_url: j.html_url ?? href,
          full_name: j.full_name ?? 'rishikanthc/scriberr',
        });
      })
      .catch(() => {});
    return () => {
      alive = false;
    };
  }, []);

  return (
    <a href={href} target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-3 rounded-md border border-gray-200 bg-white px-3 py-1.5 hover:bg-gray-50 transition" title="View on GitHub" aria-label="View Scriberr on GitHub">
      <GithubMark className="size-5 text-gray-900" />
      <div className="flex flex-col leading-tight min-w-0">
        <span className="text-sm text-gray-900 font-medium truncate">rishikanthc/scriberr</span>
        <div className="mt-0.5 inline-flex items-center gap-3 text-xs text-gray-700">
          <span className="inline-flex items-center gap-1">
            <StarIcon className="size-4" />
            {formatCount(info?.stargazers_count)}
          </span>
          <span className="inline-flex items-center gap-1">
            <ForkIcon className="size-4" />
            {formatCount(info?.forks_count)}
          </span>
        </div>
      </div>
    </a>
  );
}

function formatCount(n?: number) {
  if (typeof n !== 'number') return '—';
  if (n < 1000) return String(n);
  if (n < 10000) return (n / 1000).toFixed(1).replace(/\.0$/, '') + 'k';
  if (n < 1000000) return Math.round(n / 1000) + 'k';
  return (n / 1000000).toFixed(1).replace(/\.0$/, '') + 'm';
}

function GithubMark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 16 16" aria-hidden="true" className={className} fill="currentColor">
      <path d="M8 0C3.58 0 0 3.64 0 8.13c0 3.59 2.29 6.63 5.47 7.7.4.07.55-.18.55-.39 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.5-2.69-.96-.09-.25-.48-.96-.82-1.15-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.22 1.87.88 2.33.67.07-.53.28-.88.5-1.08-1.78-.2-3.64-.91-3.64-4.06 0-.9.31-1.63.82-2.21-.08-.2-.36-1.02.08-2.13 0 0 .67-.22 2.2.84a7.43 7.43 0 0 1 2-.27c.68 0 1.36.09 2 .27 1.53-1.06 2.2-.84 2.2-.84.44 1.11.16 1.93.08 2.13.51.58.82 1.31.82 2.21 0 3.16-1.87 3.86-3.65 4.06.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.19 0 .21.15.46.55.39A8.04 8.04 0 0 0 16 8.13C16 3.64 12.42 0 8 0Z" />
    </svg>
  );
}

function StarIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className={className}>
      <path d="M12 17.3l-5.4 3 1-5.9-4.3-4.2 6-0.9L12 4l2.7 5.3 6 0.9-4.3 4.2 1 5.9z" />
    </svg>
  );
}

function ForkIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className={className}>
      <path d="M7 4a3 3 0 1 0 0 6 3 3 0 0 0 0-6zm10 0a3 3 0 1 0 0 6 3 3 0 0 0 0-6zM7 10v2a5 5 0 0 0 5 5 5 5 0 0 0 5-5v-2" />
      <path d="M12 17v3" />
    </svg>
  );
}

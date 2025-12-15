import { useEffect, useState } from 'react';
import { Github, Star, GitFork } from 'lucide-react';

interface RepoStats {
    stargazers_count: number;
    forks_count: number;
}

export function GithubBadge() {
    const [stats, setStats] = useState<RepoStats | null>(null);

    useEffect(() => {
        fetch('https://api.github.com/repos/rishikanthc/scriberr')
            .then(res => res.json())
            .then(data => {
                setStats({
                    stargazers_count: data.stargazers_count,
                    forks_count: data.forks_count
                });
            })
            .catch(err => console.error('Failed to fetch repo stats:', err));
    }, []);

    const formatCount = (count: number) => {
        if (count >= 1000) {
            return (count / 1000).toFixed(1) + 'k';
        }
        return count;
    };

    return (
        <a
            href="https://github.com/rishikanthc/scriberr"
            target="_blank"
            rel="noopener noreferrer"
            className="group flex items-center gap-0 rounded-md overflow-hidden border border-gray-200 shadow-sm hover:shadow transition-all duration-200"
        >
            <div className="flex items-center gap-2 px-3 py-1.5 bg-gray-50 group-hover:bg-gray-100 transition-colors border-r border-gray-200">
                <Github className="w-4 h-4 text-gray-700" />
                <span className="text-xs font-semibold text-gray-700 font-heading">Star</span>
            </div>
            {stats && (
                <div className="flex items-center bg-white">
                    <div className="flex items-center gap-1 px-2 py-1.5 border-r border-gray-100 last:border-0 hover:bg-gray-50 transition-colors">
                        <Star className="w-3.5 h-3.5 text-amber-400 fill-amber-400" />
                        <span className="text-xs font-medium text-gray-600 font-mono">{formatCount(stats.stargazers_count)}</span>
                    </div>
                    <div className="flex items-center gap-1 px-2 py-1.5 hover:bg-gray-50 transition-colors">
                        <GitFork className="w-3.5 h-3.5 text-gray-400" />
                        <span className="text-xs font-medium text-gray-600 font-mono">{formatCount(stats.forks_count)}</span>
                    </div>
                </div>
            )}
        </a>
    );
}

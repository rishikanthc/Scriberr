import { useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';

interface TOCItem {
    id: string;
    text: string;
    level: number;
}

export function TableOfContents() {
    const [headings, setHeadings] = useState<TOCItem[]>([]);
    const [activeId, setActiveId] = useState<string>('');
    const location = useLocation();

    useEffect(() => {
        // Find all h2 and h3 elements within the docs content
        const elements = Array.from(document.querySelectorAll('.docs-content h2, .docs-content h3'));

        const items: TOCItem[] = elements.map((element) => {
            // Ensure element has an ID
            if (!element.id) {
                element.id = element.textContent?.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)+/g, '') || '';
            }

            return {
                id: element.id,
                text: element.textContent || '',
                level: parseInt(element.tagName.substring(1)),
            };
        });

        // Update headings state in a microtask to avoid cascading renders
        queueMicrotask(() => {
            setHeadings(items);
        });

        const observer = new IntersectionObserver(
            (entries) => {
                entries.forEach((entry) => {
                    if (entry.isIntersecting) {
                        setActiveId(entry.target.id);
                    }
                });
            },
            { rootMargin: '-100px 0px -66%' }
        );

        elements.forEach((elem) => observer.observe(elem));

        return () => observer.disconnect();
    }, [location.pathname]);

    if (headings.length === 0) return null;

    return (
        <nav className="hidden xl:block sticky top-24 max-h-[calc(100vh-8rem)] overflow-y-auto w-64 pl-8">
            <h4 className="text-sm font-semibold text-gray-900 mb-2 uppercase tracking-wider">On this page</h4>
            <ul className="space-y-1.5">
                {headings.map((heading) => (
                    <li key={heading.id} style={{ paddingLeft: heading.level === 3 ? '1rem' : '0' }}>
                        <a
                            href={`#${heading.id}`}
                            className={`text-sm font-[family-name:var(--font-heading)] transition-colors duration-200 block ${activeId === heading.id
                                ? 'text-[#FF6D20] font-medium'
                                : 'text-gray-500 hover:text-gray-900'
                                }`}
                            onClick={(e) => {
                                e.preventDefault();
                                document.getElementById(heading.id)?.scrollIntoView({ behavior: 'smooth' });
                                setActiveId(heading.id);
                                // Update URL hash without jumping
                                window.history.pushState(null, '', `#${heading.id}`);
                            }}
                        >
                            {heading.text}
                        </a>
                    </li>
                ))}
            </ul>
        </nav>
    );
}

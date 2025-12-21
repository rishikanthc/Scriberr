import { useEffect, useMemo, useState } from 'react';
import { ScriberrLogo } from './ScriberrLogo';

/* eslint-disable @typescript-eslint/no-explicit-any */

type SwaggerDoc = {
    openapi?: string;
    swagger?: string;
    host?: string;
    basePath?: string;
    schemes?: string[];
    info?: { title?: string; version?: string; description?: string };
    tags?: { name: string; description?: string }[];
    paths?: Record<string, Record<string, any>>;
    components?: {
        schemas?: Record<string, any>;
        requestBodies?: Record<string, any>;
        securitySchemes?: Record<string, any>;
    };
    definitions?: Record<string, any>;
    securityDefinitions?: Record<string, any>;
};

type Endpoint = {
    method: string;
    path: string;
    summary?: string;
    description?: string;
    tag?: string;
    meta?: any;
};

export default function ApiReference() {
    const [doc, setDoc] = useState<SwaggerDoc | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [filter, setFilter] = useState<string>('');
    const [extra, setExtra] = useState<{ endpoints: Endpoint[] } | null>(null);
    const [mobileTagsOpen, setMobileTagsOpen] = useState(false);

    useEffect(() => {
        fetch('/api/swagger.json')
            .then((r) => (r.ok ? r.json() : Promise.reject(r.statusText)))
            .then((j) => setDoc(j))
            .catch((e) => setError(String(e)));
    }, []);

    useEffect(() => {
        fetch('/api/undocumented.json')
            .then((r) => (r.ok ? r.json() : null))
            .then((j) => j && setExtra(j))
            .catch(() => { });
    }, []);

    const grouped = useMemo(() => {
        if (!doc?.paths) return {} as Record<string, Endpoint[]>;
        const endpoints: Endpoint[] = [];
        for (const [path, ops] of Object.entries(doc.paths)) {
            for (const [method, meta] of Object.entries(ops)) {
                const m = method.toUpperCase();
                if (!['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].includes(m)) continue;
                const tag: string | undefined = Array.isArray(meta.tags) ? meta.tags[0] : undefined;
                endpoints.push({ method: m, path, summary: meta.summary || meta.operationId, description: meta.description, tag, meta });
            }
        }
        const byTag: Record<string, Endpoint[]> = {};
        for (const ep of endpoints) {
            const key = ep.tag || 'General';
            if (!byTag[key]) byTag[key] = [];
            byTag[key].push(ep);
        }
        if (extra?.endpoints?.length) {
            const keyFn = (e: Endpoint) => `${e.method}:${e.path}`;
            const seen = new Set(endpoints.map(keyFn));
            for (const e of extra.endpoints) {
                const k = keyFn(e);
                if (seen.has(k)) continue;
                const tag = e.tag || 'Undocumented';
                if (!byTag[tag]) byTag[tag] = [];
                byTag[tag].push({ ...e, meta: {} });
            }
        }
        for (const k of Object.keys(byTag)) byTag[k].sort((a, b) => a.path.localeCompare(b.path));
        return byTag;
    }, [doc, extra]);

    const tagOrder = useMemo(() => {
        const available = Object.keys(grouped);
        if (!doc?.tags) return available;
        const declared = doc.tags.map((t) => t.name);
        return [...declared.filter((t) => available.includes(t)), ...available.filter((t) => !declared.includes(t))];
    }, [doc, grouped]);

    return (
        <div className="min-h-screen bg-white">
            <header className="sticky top-0 z-10 bg-white/80 backdrop-blur-md border-b border-gray-100">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-3 flex items-center justify-between gap-3">
                    <a href="/" className="flex items-center gap-3 select-none min-w-0">
                        <ScriberrLogo />
                        <span className="text-gray-300 text-lg font-light">/</span>
                        <span className="text-sm font-semibold text-[#FF6D20] tracking-wide uppercase pt-1">API Reference</span>
                    </a>
                    <div className="flex items-center gap-2 w-full sm:w-auto">
                        <input
                            placeholder="Search endpoints"
                            value={filter}
                            onChange={(e) => setFilter(e.target.value)}
                            className="w-full sm:w-64 rounded-lg border border-gray-200 bg-gray-50/50 px-3 py-1.5 text-sm outline-none focus:ring-2 focus:ring-[#FF6D20]/20 focus:border-[#FF6D20] transition-colors"
                        />
                        <button
                            className="sm:hidden inline-flex items-center justify-center rounded-lg border border-gray-200 bg-white px-2.5 py-1.5 text-gray-700 hover:bg-gray-50"
                            onClick={() => setMobileTagsOpen((v) => !v)}
                            aria-label="Toggle tags"
                        >
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className="size-5">
                                <path d="M4 7h16M4 12h16M4 17h16" />
                            </svg>
                        </button>
                    </div>
                </div>
            </header>

            <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
                {mobileTagsOpen && (
                    <div className="sm:hidden mb-6 border border-gray-200 rounded-lg p-3 bg-gray-50/50">
                        <div className="text-[11px] font-semibold uppercase tracking-wider text-gray-500 mb-2">Tags</div>
                        <ul className="grid grid-cols-2 gap-2 text-sm">
                            {tagOrder.length > 0 ? (
                                tagOrder.map((tag) => (
                                    <li key={`m-${tag}`}>
                                        <a href={`#tag-${encodeURIComponent(tag)}`} className="text-gray-700 hover:text-[#FF6D20] transition-colors block py-1" onClick={() => setMobileTagsOpen(false)}>{tag}</a>
                                    </li>
                                ))
                            ) : (
                                <li className="text-gray-400">Loading tags…</li>
                            )}
                        </ul>
                    </div>
                )}

                {error && (
                    <div className="mb-8 p-4 rounded-lg bg-red-50 text-red-600 border border-red-100 flex items-center gap-2">
                        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10" /><line x1="12" x2="12" y1="8" y2="12" /><line x1="12" x2="12.01" y1="16" y2="16" /></svg>
                        <p>Failed to load API definition: {error}</p>
                    </div>
                )}

                <div className="mb-12">
                    <h1 className="text-3xl md:text-4xl font-bold tracking-tight text-gray-900 font-heading mb-2">{doc?.info?.title || 'Scriberr API'}</h1>
                    <div className="flex items-center gap-2 text-sm text-gray-600">
                        <span className="px-2 py-0.5 rounded-full bg-gray-100 text-gray-700 font-medium">v{doc?.info?.version || '1.0.0'}</span>
                        {doc && <BaseURL doc={doc} />}
                    </div>
                </div>

                <AuthIntro />

                <div className="grid grid-cols-1 md:grid-cols-[240px_minmax(0,1fr)] gap-8 mt-12">
                    <aside className="hidden md:block">
                        <div className="sticky top-24 pr-4">
                            <div className="text-[11px] font-semibold uppercase tracking-wider text-gray-500 mb-3">Tags</div>
                            <ul className="space-y-1 text-sm">
                                {tagOrder.length > 0 ? (
                                    tagOrder.map((tag) => (
                                        <li key={tag}>
                                            <a
                                                href={`#tag-${encodeURIComponent(tag)}`}
                                                className="block py-1.5 px-2 -mx-2 rounded-md text-gray-600 hover:text-gray-900 hover:bg-gray-50 transition-colors"
                                            >
                                                {tag}
                                            </a>
                                        </li>
                                    ))
                                ) : (
                                    <li className="text-gray-400">Loading tags…</li>
                                )}
                            </ul>
                        </div>
                    </aside>

                    <div>
                        <section className="space-y-12">
                            {doc ? (
                                tagOrder.map((tag) => {
                                    const eps = (grouped[tag] || []).filter((e) => {
                                        const f = filter.trim().toLowerCase();
                                        if (!f) return true;
                                        return (
                                            e.path.toLowerCase().includes(f) ||
                                            e.method.toLowerCase().includes(f) ||
                                            (e.summary || '').toLowerCase().includes(f) ||
                                            (e.description || '').toLowerCase().includes(f)
                                        );
                                    });
                                    if (!eps.length) return null;
                                    return (
                                        <div key={tag} id={`tag-${tag}`} className="scroll-mt-24">
                                            <h2 className="text-xl font-bold mb-6 text-gray-900 flex items-center gap-2 after:h-px after:flex-1 after:bg-gray-100">
                                                {tag}
                                            </h2>
                                            <div className="space-y-4">
                                                {eps.map((e) => (
                                                    <EndpointCard key={`${e.method}-${e.path}`} ep={e} doc={doc} />
                                                ))}
                                            </div>
                                        </div>
                                    );
                                })
                            ) : (
                                <div className="space-y-4">
                                    {[1, 2, 3].map(i => (
                                        <div key={i} className="animate-pulse bg-gray-50 h-32 rounded-lg border border-gray-100"></div>
                                    ))}
                                </div>
                            )}
                        </section>
                    </div>
                </div>
            </main>
        </div>
    );
}

function BaseURL({ doc }: { doc: SwaggerDoc }) {
    const loc = typeof window !== 'undefined' ? window.location : ({ protocol: 'http:', host: 'localhost:8080' } as any);
    const basePath = (doc as any).basePath || '';
    const base = doc.openapi ? (Array.isArray((doc as any).servers) && (doc as any).servers[0]?.url) || `${loc.protocol}//${loc.host}` : `${loc.protocol}//${doc.host || loc.host}${basePath}`;
    return (
        <>
            <span className="text-gray-300">•</span>
            <span>Base URL: <code className="font-mono text-gray-900 bg-gray-100 px-1 py-0.5 rounded text-xs">{base}</code></span>
        </>
    );
}

function AuthIntro() {
    const origin = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8080';
    const apiBase = `${origin}/api/v1`;

    const examples = {
        getJwt: {
            label: 'Get Token',
            curl: [
                'curl -X POST',
                `'${apiBase}/auth/login'`,
                "-H 'Content-Type: application/json'",
                "-d '{\"username\":\"alice\",\"password\":\"your-password\"}'",
            ].join(' ')
        },
        useJwt: {
            label: 'Use Token',
            curl: [
                'curl -X GET',
                `'${apiBase}/transcription/list'`,
                "-H 'Authorization: Bearer YOUR_JWT'",
            ].join(' ')
        },
        useApiKey: {
            label: 'Use API Key',
            curl: [
                'curl -X GET',
                `'${apiBase}/transcription/list'`,
                "-H 'X-API-Key: YOUR_API_KEY'",
            ].join(' ')
        }
    };

    const [activeExample, setActiveExample] = useState<keyof typeof examples>('getJwt');

    return (
        <section className="mt-8 mb-16">
            <div className="bg-gradient-to-br from-gray-50 to-white rounded-2xl border border-gray-100 overflow-hidden shadow-sm">
                <div className="grid lg:grid-cols-5">
                    <div className="lg:col-span-2 p-6 md:p-8 border-b lg:border-b-0 lg:border-r border-gray-100 flex flex-col justify-center">
                        <h3 className="text-xl font-bold text-gray-900 mb-4 flex items-center gap-2">
                            <div className="p-2 rounded-lg bg-orange-100 text-[#FF6D20]">
                                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect width="18" height="11" x="3" y="11" rx="2" ry="2" /><path d="M7 11V7a5 5 0 0 1 10 0v4" /></svg>
                            </div>
                            Authentication
                        </h3>
                        <p className="text-gray-600 mb-6 leading-relaxed">
                            The API supports two authentication methods: <strong>Bearer JWT</strong> and <strong>API Key</strong>.
                            <br /><br />
                            Most endpoints are flexible, but sensitive user management actions require a JWT session.
                        </p>
                        <div className="space-y-3">
                            <div className="flex items-start gap-3 p-3 rounded-lg bg-white border border-gray-100 shadow-sm">
                                <span className="text-xs font-bold px-2 py-0.5 rounded bg-blue-50 text-blue-700 border border-blue-100 mt-0.5">JWT</span>
                                <div className="text-xs text-gray-500">
                                    Required for user settings, LLM config, and account management.
                                </div>
                            </div>
                            <div className="flex items-start gap-3 p-3 rounded-lg bg-white border border-gray-100 shadow-sm">
                                <span className="text-xs font-bold px-2 py-0.5 rounded bg-emerald-50 text-emerald-700 border border-emerald-100 mt-0.5">API Key</span>
                                <div className="text-xs text-gray-500">
                                    Suitable for server-side integrations: transcription, chat, and data retrieval.
                                </div>
                            </div>
                        </div>
                    </div>

                    <div className="lg:col-span-3 bg-[#0d1117] p-6 md:p-8 flex flex-col">
                        <div className="flex items-center gap-2 mb-4 overflow-x-auto">
                            {(Object.keys(examples) as Array<keyof typeof examples>).map((key) => (
                                <button
                                    key={key}
                                    onClick={() => setActiveExample(key)}
                                    className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all duration-200 ${activeExample === key
                                        ? 'bg-white/10 text-white shadow-sm ring-1 ring-white/5'
                                        : 'text-gray-400 hover:text-gray-200 hover:bg-white/5'
                                        }`}
                                >
                                    {examples[key].label}
                                </button>
                            ))}
                        </div>
                        <div className="flex-1">
                            <CodeTabs
                                curl={examples[activeExample].curl}
                                js={curlToFetch(examples[activeExample].curl)}
                                wrap
                                bordered={false}
                            />
                        </div>
                    </div>
                </div>
            </div>
        </section>
    );
}

function EndpointCard({ ep, doc }: { ep: Endpoint; doc: SwaggerDoc }) {
    const id = useMemo(() => makeId(`${ep.method}-${ep.path}`), [ep.method, ep.path]);
    const [expanded, setExpanded] = useState(false);
    const [showAllParams, setShowAllParams] = useState(false);
    const [showAllResponses, setShowAllResponses] = useState(false);
    const parameters: any[] = Array.isArray(ep.meta?.parameters) ? ep.meta.parameters : [];
    const requestBody = ep.meta?.requestBody;
    const responses = ep.meta?.responses || {};

    const endpointSecurity: string[] = useMemo(() => {
        const sec = Array.isArray(ep.meta?.security) ? ep.meta.security : [];
        const names: string[] = [];
        for (const s of sec) {
            if (s && typeof s === 'object') {
                names.push(...Object.keys(s));
            }
        }
        return [...new Set(names)];
    }, [ep.meta]);

    const securityDefs: Record<string, any> = doc.securityDefinitions || doc.components?.securitySchemes || {};
    const bodyResolved = resolveRequestBody(requestBody, doc) || legacyBodyFromSwagger2(ep.meta);
    const exampleContentType = bodyResolved && Object.keys(bodyResolved.content || {})[0];
    const exampleSchema = exampleContentType ? bodyResolved!.content[exampleContentType]?.schema : undefined;
    const examplePayload = exampleSchema ? makeExample(exampleSchema, doc.components, doc.definitions) : undefined;
    const preferredSec = endpointSecurity[0];
    const authHeaders = makeAuthHeaders(preferredSec, securityDefs);
    const curl = makeCurl(ep.method, ep.path, exampleContentType, examplePayload, authHeaders, parameters);

    return (
        <article id={id} className={`group rounded-xl border transition-all duration-200 ${expanded ? 'bg-white border-gray-200 shadow-sm ring-1 ring-gray-900/5' : 'bg-white border-transparent hover:border-gray-200 hover:shadow-sm hover:bg-gray-50/50'}`}>
            <button onClick={() => setExpanded((v) => !v)} className="w-full text-left p-4">
                <div className="flex items-start gap-4">
                    <MethodBadge method={ep.method} />
                    <div className="flex-1 min-w-0 pt-1">
                        <div className="flex items-center gap-3">
                            <code className="text-sm font-semibold text-gray-900 truncate font-mono">{ep.path}</code>
                            {ep.summary && <span className="text-gray-400">/</span>}
                            {ep.summary && <div className="text-sm text-gray-600 truncate">{ep.summary}</div>}
                            <a href={`#${id}`} title="Permalink" className="ml-auto text-gray-300 hover:text-[#FF6D20] opacity-0 group-hover:opacity-100 transition-opacity p-1">#</a>
                        </div>
                        {!expanded && ep.description && (
                            <div className="text-xs text-gray-500 mt-1 truncate max-w-2xl">{ep.description}</div>
                        )}
                    </div>
                </div>
            </button>

            {expanded && (
                <div className="px-4 pb-5 pt-0 border-t border-gray-100 mt-2">
                    <div className="pt-4 space-y-6">
                        {ep.description && (
                            <p className="text-sm text-gray-600 leading-relaxed">{ep.description}</p>
                        )}

                        {!!endpointSecurity.length && (
                            <div className="flex flex-wrap items-center gap-2 text-xs">
                                <span className="text-gray-500 font-medium">Authorization:</span>
                                {endpointSecurity.map((name) => (
                                    <AuthPill key={name} name={name} def={securityDefs?.[name]} />
                                ))}
                            </div>
                        )}

                        {!!parameters.length && (
                            <div>
                                <h4 className="text-xs font-semibold uppercase tracking-wider text-gray-500 mb-3">Parameters</h4>
                                <div className="bg-gray-50 rounded-lg border border-gray-100 p-1">
                                    <ParamGroups params={showAllParams ? parameters : parameters.slice(0, 8)} />
                                </div>
                                {parameters.length > 8 && (
                                    <button onClick={() => setShowAllParams((v) => !v)} className="mt-2 text-xs font-medium text-[#FF6D20] hover:text-orange-700">
                                        {showAllParams ? 'Show less' : `Show all ${parameters.length} parameters`}
                                    </button>
                                )}
                            </div>
                        )}

                        {exampleSchema && (
                            <div>
                                <h4 className="text-xs font-semibold uppercase tracking-wider text-gray-500 mb-3">Request Body <span className="font-normal text-gray-400">({exampleContentType})</span></h4>
                                <CodeBlock text={formatMaybeJSON(examplePayload)} />
                            </div>
                        )}

                        {responses && Object.keys(responses).length > 0 && (
                            <div>
                                <h4 className="text-xs font-semibold uppercase tracking-wider text-gray-500 mb-3">Response</h4>
                                {(() => {
                                    const entries = Object.entries(responses);
                                    const idx = entries.findIndex(([c]) => c === '200');
                                    const primary = entries[idx >= 0 ? idx : 0] as any;
                                    const others = entries.filter((_, i) => i !== (idx >= 0 ? idx : 0));
                                    const [code, r] = primary;
                                    const content = (r?.content || {}) as any;
                                    const firstCT = Object.keys(content)[0];
                                    const schema = firstCT ? content[firstCT]?.schema : r?.schema;
                                    const ex = schema ? makeExample(schema, doc.components, doc.definitions) : (r?.description || '');
                                    return (
                                        <div className="space-y-3">
                                            <div className="rounded-lg border border-gray-200 overflow-hidden">
                                                <div className="bg-gray-50 px-3 py-2 border-b border-gray-200 flex items-center justify-between">
                                                    <div className="flex items-center gap-2">
                                                        <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${code.startsWith('2') ? 'bg-green-100 text-green-700' : 'bg-gray-200 text-gray-700'}`}>
                                                            HTTP {code}
                                                        </span>
                                                        {r?.description && <span className="text-xs text-gray-500">{r.description}</span>}
                                                    </div>
                                                </div>
                                                <CodeBlock text={formatMaybeJSON(ex)} bordered={false} />
                                            </div>

                                            {others.length > 0 && (
                                                <div className="pt-1">
                                                    <button onClick={() => setShowAllResponses((v) => !v)} className="text-xs font-medium text-gray-500 hover:text-gray-800 flex items-center gap-1">
                                                        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={`transition-transform ${showAllResponses ? 'rotate-180' : ''}`}><polyline points="6 9 12 15 18 9" /></svg>
                                                        {showAllResponses ? 'Hide other responses' : `Show ${others.length} other responses`}
                                                    </button>
                                                </div>
                                            )}

                                            {showAllResponses && (
                                                <div className="space-y-3 pl-3 border-l-2 border-gray-100">
                                                    {others.map(([oc, or]: any) => {
                                                        const ocContent = (or?.content || {}) as any;
                                                        const ocCT = Object.keys(ocContent)[0];
                                                        const ocSchema = ocCT ? ocContent[ocCT]?.schema : or?.schema;
                                                        const ocEx = ocSchema ? makeExample(ocSchema, doc.components, doc.definitions) : (or?.description || '');
                                                        return (
                                                            <div key={oc} className="rounded-lg border border-gray-200 overflow-hidden">
                                                                <div className="bg-gray-50 px-3 py-2 border-b border-gray-200 flex items-center gap-2">
                                                                    <span className="text-[10px] font-bold px-2 py-0.5 rounded-full bg-gray-200 text-gray-700">HTTP {oc}</span>
                                                                    {or?.description && <span className="text-xs text-gray-500">{or.description}</span>}
                                                                </div>
                                                                <CodeBlock text={formatMaybeJSON(ocEx)} bordered={false} />
                                                            </div>
                                                        );
                                                    })}
                                                </div>
                                            )}
                                        </div>
                                    );
                                })()}
                            </div>
                        )}

                        <div>
                            <h4 className="text-xs font-semibold uppercase tracking-wider text-gray-500 mb-3">Example Usage</h4>
                            <CodeTabs curl={curl} js={curlToFetch(curl)} />
                        </div>
                    </div>
                </div>
            )}
        </article>
    );
}

function CodeBlock({ text, wrap, bordered = true }: { text: any; language?: string; wrap?: boolean, bordered?: boolean }) {
    const value = typeof text === 'string' ? text : JSON.stringify(text, null, 2);
    const [copied, setCopied] = useState(false);
    return (
        <div className={`relative group ${bordered ? 'rounded-lg border border-gray-200 bg-[#0d1117]' : 'bg-[#0d1117]'}`}>
            <pre className={`p-4 text-xs font-mono text-gray-300 overflow-x-auto ${wrap ? 'whitespace-pre-wrap' : ''}`}><code>{value}</code></pre>
            <button
                onClick={() => {
                    navigator.clipboard.writeText(value).then(() => {
                        setCopied(true);
                        setTimeout(() => setCopied(false), 1200);
                    });
                }}
                className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity inline-flex items-center gap-1.5 rounded-md border border-white/10 bg-white/5 px-2 py-1 text-[10px] text-gray-300 hover:bg-white/10 hover:text-white"
            >
                {copied ? (
                    <>
                        <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12" /></svg>
                        Copied
                    </>
                ) : (
                    'Copy'
                )}
            </button>
        </div>
    );
}

function CodeTabs({ curl, js, wrap, bordered = true }: { curl: string; js: string; wrap?: boolean; bordered?: boolean }) {
    const [tab, setTab] = useState<'curl' | 'js'>('curl');
    return (
        <div className={`overflow-hidden ${bordered ? 'rounded-lg border border-gray-200 bg-[#0d1117]' : ''}`}>
            <div className={`flex px-2 pt-2 gap-1 ${bordered ? 'border-b border-white/10 bg-white/5' : ''}`}>
                <button
                    onClick={() => setTab('curl')}
                    className={`px-3 py-1.5 text-xs font-medium rounded-t-md transition-colors ${tab === 'curl' ? 'bg-[#0d1117] text-white border-x border-t border-white/10 relative -bottom-px' : 'text-gray-400 hover:text-gray-200 hover:bg-white/5'}`}
                >
                    cURL
                </button>
                <button
                    onClick={() => setTab('js')}
                    className={`px-3 py-1.5 text-xs font-medium rounded-t-md transition-colors ${tab === 'js' ? 'bg-[#0d1117] text-white border-x border-t border-white/10 relative -bottom-px' : 'text-gray-400 hover:text-gray-200 hover:bg-white/5'}`}
                >
                    JavaScript
                </button>
            </div>
            <CodeBlock text={tab === 'curl' ? curl : js} wrap={wrap} bordered={false} />
        </div>
    );
}

function MethodBadge({ method }: { method: string }) {
    const styles: Record<string, string> = {
        GET: 'bg-emerald-50 text-emerald-700 border-emerald-200 hover:bg-emerald-100',
        POST: 'bg-blue-50 text-blue-700 border-blue-200 hover:bg-blue-100',
        PUT: 'bg-amber-50 text-amber-700 border-amber-200 hover:bg-amber-100',
        PATCH: 'bg-cyan-50 text-cyan-700 border-cyan-200 hover:bg-cyan-100',
        DELETE: 'bg-rose-50 text-rose-700 border-rose-200 hover:bg-rose-100',
    };
    const cls = styles[method] || 'bg-gray-100 text-gray-700 border-gray-200';
    return (
        <span className={`inline-flex items-center justify-center rounded-md border w-[60px] py-1 text-[11px] font-bold tracking-wide transition-colors ${cls}`}>
            {method}
        </span>
    );
}

function ParamGroups({ params }: { params: any[] }) {
    const groups: Record<string, any[]> = { path: [], query: [], header: [], cookie: [], body: [] };
    for (const p of params) {
        const k = (p.in || 'other').toLowerCase();
        if (!groups[k]) groups[k] = [];
        groups[k].push(p);
    }
    const order = ['path', 'query', 'header', 'cookie'];
    return (
        <div className="divide-y divide-gray-100">
            {order.map((k) => groups[k] && groups[k].length ? (
                <div key={k} className="p-3">
                    <div className="text-[10px] font-bold uppercase text-gray-400 mb-2">{k} parameters</div>
                    <ul className="space-y-2">
                        {groups[k].map((p) => {
                            const schema = p.schema || {};
                            const type = schema.type || (schema.$ref ? refName(schema.$ref) : p.type || '');
                            const bits: string[] = [];
                            if (schema.default !== undefined || p.default !== undefined) bits.push(`default: ${schema.default ?? p.default}`);
                            if (schema.enum) bits.push(`enum: ${(schema.enum || []).join(', ')}`);
                            const desc = [p.description, ...bits].filter(Boolean).join(' — ');
                            return (
                                <li key={`${p.name}-${p.in}`} className="text-xs">
                                    <div className="flex items-baseline gap-2 mb-0.5">
                                        <code className="font-mono font-semibold text-gray-900 bg-gray-100 px-1 py-0.5 rounded">{p.name}</code>
                                        <span className="text-gray-500 font-mono text-[11px]">{type}</span>
                                        {p.required && <span className="text-red-500 font-medium text-[10px] uppercase tracking-wider">Required</span>}
                                    </div>
                                    {desc && <div className="text-gray-600 pl-1">{desc}</div>}
                                </li>
                            );
                        })}
                    </ul>
                </div>
            ) : null)}
        </div>
    );
}

function makeId(input: string) {
    return input.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '');
}

function refName(ref: string) {
    const m = ref.match(/#\/(components\/(schemas|requestBodies)|definitions)\/(.+)$/);
    return m ? m[m.length - 1] : ref;
}

function resolveRef(obj: any, components?: SwaggerDoc['components'], definitions?: Record<string, any>): any {
    if (!obj || !obj.$ref) return obj;
    const name = refName(obj.$ref);
    return components?.schemas?.[name] || components?.requestBodies?.[name] || definitions?.[name] || obj;
}

function resolveRequestBody(rb: any, doc: SwaggerDoc): any {
    if (!rb) return undefined;
    if (rb.$ref) return resolveRef(rb, doc.components, doc.definitions);
    return rb;
}

function legacyBodyFromSwagger2(meta: any) {
    if (!meta?.parameters) return undefined;
    const body = meta.parameters.find((p: any) => p.in === 'body');
    if (!body) return undefined;
    return {
        content: { 'application/json': { schema: body.schema || {} } },
    };
}

function makeExample(
    schema: any,
    components?: SwaggerDoc['components'],
    definitions?: Record<string, any>,
    depth = 0,
    seen: Set<string> = new Set()
): any {
    if (!schema) return '';
    if (depth > 8) return '…';
    if (schema.example !== undefined) return schema.example;

    if (schema.$ref) {
        const ref = schema.$ref as string;
        const name = refName(ref);
        if (seen.has(name)) return `{circular:${name}}`;
        seen.add(name);
        return makeExample(resolveRef(schema, components, definitions), components, definitions, depth + 1, seen);
    }

    const type = schema.type;
    if (schema.examples && Array.isArray(schema.examples) && schema.examples.length) return schema.examples[0];

    if (type === 'object' || (schema.properties && typeof schema.properties === 'object')) {
        const obj: any = {};
        const props = schema.properties || {};
        let count = 0;
        for (const [k, v] of Object.entries<any>(props)) {
            obj[k] = makeExample(v, components, definitions, depth + 1, seen);
            if (++count > 15) break;
        }
        return obj;
    }

    if (type === 'array') {
        return [makeExample(schema.items || {}, components, definitions, depth + 1, seen)];
    }

    if (schema.enum && Array.isArray(schema.enum)) return schema.enum[0];

    switch (type) {
        case 'integer':
        case 'number':
            return 123;
        case 'boolean':
            return true;
        case 'string':
        default:
            return schema?.format === 'date-time' ? new Date().toISOString() : 'string';
    }
}

function formatMaybeJSON(value: any) {
    if (typeof value === 'string') return value;
    try { return JSON.stringify(value, null, 2); } catch { return String(value); }
}

// Very basic curl generator
function makeCurl(method: string, path: string, ct?: string, body?: any, authHeaders?: string[], params?: any[]) {
    const origin = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8080';
    const qParams = (params || []).filter((p) => p.in === 'query');
    const qs = method === 'GET' && qParams.length
        ? '?' + qParams.map((p) => `${encodeURIComponent(p.name)}=${encodeURIComponent(exampleForParam(p))}`).join('&')
        : '';
    const url = `${origin}${path}${qs}`;
    const parts: string[] = ['curl', '-X', method, `'${url}'`];
    if (authHeaders && authHeaders.length) {
        for (const h of authHeaders) parts.push('-H', `'${h}'`);
    }
    if (ct) parts.push('-H', `'Content-Type: ${ct}'`);
    if (body !== undefined) parts.push('-d', `'${JSON.stringify(body)}'`);
    return parts.join(' ');
}

function curlToFetch(curl: string): string {
    try {
        const methodMatch = curl.match(/-X\s+(GET|POST|PUT|PATCH|DELETE)/i);
        const urlMatch = curl.match(/'https?:[^']+'/);
        const headerRe = /-H\s+'([^']+)'/g;
        const dataMatch = curl.match(/-d\s+'([^']+)'/);

        const method = methodMatch ? methodMatch[1].toUpperCase() : 'GET';
        const url = urlMatch ? urlMatch[0].slice(1, -1) : '';
        const headers: Record<string, string> = {};
        let m: RegExpExecArray | null;
        while ((m = headerRe.exec(curl)) !== null) {
            const [key, ...rest] = m[1].split(':');
            const value = rest.join(':').trim();
            headers[key.trim()] = value;
        }
        const body = dataMatch ? dataMatch[1] : undefined;

        const lines = [
            `const res = await fetch('${url}', {`,
            `  method: '${method}',`,
            Object.keys(headers).length ? `  headers: ${JSON.stringify(headers, null, 2).replace(/\n/g, '\n  ').replace(/^/gm, '  ')},` : '',
            body ? `  body: ${isLikelyJSON(body) ? `JSON.stringify(${safeJSON(body)})` : `'${body}'`},` : '',
            `});`,
            `const data = await res.json();`,
            `console.log(data);`
        ].filter(Boolean);
        return lines.join('\n');
    } catch {
        return `// Could not convert to fetch.\n// Original:\n${curl}`;
    }
}

function isLikelyJSON(s: string) {
    try { JSON.parse(s); return true; } catch { return false; }
}
function safeJSON(s: string) {
    try { return JSON.parse(s); } catch { return s; }
}

function makeAuthHeaders(secName?: string, defs: Record<string, any> = {}): string[] {
    if (!secName || !defs[secName]) return [];
    const def = defs[secName];
    if (def.type === 'apiKey') {
        const name = def.name || (def.description?.includes('Authorization') ? 'Authorization' : 'X-API-Key');
        if (name.toLowerCase() === 'authorization') return ['Authorization: Bearer YOUR_JWT'];
        return [`${name}: YOUR_API_KEY`];
    }
    if (def.type === 'http' && def.scheme === 'bearer') return ['Authorization: Bearer YOUR_JWT'];
    return [];
}

function AuthPill({ name, def }: { name: string; def: any }) {
    const label = (() => {
        if (!def) return name;
        if (def.type === 'apiKey') return def.name === 'Authorization' ? 'Bearer token' : `${def.name} header`;
        if (def.type === 'http' && def.scheme === 'bearer') return 'Bearer token';
        return name;
    })();
    return (
        <span className="inline-flex items-center gap-1.5 rounded-full bg-blue-50 text-blue-700 px-2 py-0.5 border border-blue-100">
            <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><rect width="18" height="11" x="3" y="11" rx="2" ry="2" /><path d="M7 11V7a5 5 0 0 1 10 0v4" /></svg>
            {label}
        </span>
    );
}

function exampleForParam(p: any) {
    const s = p.schema || {};
    if (p.example !== undefined) return p.example;
    if (s.example !== undefined) return s.example;
    if (s.default !== undefined) return s.default;
    if (s.enum && s.enum.length) return s.enum[0];
    const t = s.type || p.type;
    switch (t) {
        case 'integer':
        case 'number':
            return 1;
        case 'boolean':
            return true;
        case 'string':
        default:
            return 'value';
    }
}

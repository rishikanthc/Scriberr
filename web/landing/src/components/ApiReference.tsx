import { useEffect, useMemo, useState } from 'react';
import GithubBadge from './GithubBadge';

type SwaggerDoc = {
  openapi?: string;
  swagger?: string;
  info?: { title?: string; version?: string; description?: string };
  tags?: { name: string; description?: string }[];
  paths?: Record<string, Record<string, any>>;
  components?: {
    schemas?: Record<string, any>;
    requestBodies?: Record<string, any>;
    securitySchemes?: Record<string, any>;
  };
  securityDefinitions?: Record<string, any>; // Swagger 2.0
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
      .catch(() => {});
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
    // Merge extra undocumented endpoints not present in spec
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
      <header className="api-topbar">
        <div className="container-narrow py-3 flex items-center justify-between gap-3">
          <a href="/" className="flex items-center gap-2 select-none min-w-0">
            <span className="logo-font-poiret text-lg text-gray-900">Scriberr</span>
            <span className="text-gray-300">/</span>
            <span className="text-sm text-gray-600">API Reference</span>
          </a>
          <div className="flex items-center gap-2 w-full sm:w-auto">
            <input
              placeholder="Search endpoints"
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              className="w-full sm:w-56 rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm outline-none focus:ring-2 focus:ring-gray-300"
            />
            <button
              className="sm:hidden inline-flex items-center justify-center rounded-md border border-gray-200 bg-white px-2.5 py-1.5 text-gray-700 hover:bg-gray-50"
              onClick={() => setMobileTagsOpen((v) => !v)}
              aria-label="Toggle tags"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className="size-5">
                <path d="M4 7h16M4 12h16M4 17h16" />
              </svg>
            </button>
            <div className="hidden md:block"><GithubBadge /></div>
          </div>
        </div>
      </header>

      <main className="container-narrow py-10">
        {mobileTagsOpen && (
          <div className="sm:hidden mb-4 border border-gray-200 rounded-lg p-3">
            <div className="text-[11px] font-medium text-gray-500 mb-2">Tags</div>
            <ul className="grid grid-cols-2 gap-2 text-sm">
              {tagOrder.length > 0 ? (
                tagOrder.map((tag) => (
                  <li key={`m-${tag}`}>
                    <a href={`#tag-${encodeURIComponent(tag)}`} className="text-gray-700 hover:text-gray-900" onClick={() => setMobileTagsOpen(false)}>{tag}</a>
                  </li>
                ))
              ) : (
                <li className="text-gray-400">Loading tags…</li>
              )}
            </ul>
          </div>
        )}
        {error && <p className="text-red-600">Failed to load: {error}</p>}

        <div className="mb-10">
          <h1 className="text-[28px] font-semibold tracking-tight text-gray-900">{doc?.info?.title || 'Scriberr API'}</h1>
          <div className="mt-1 text-sm text-gray-600">Version {doc?.info?.version || '1.0.0'}</div>
          {doc && <BaseURL doc={doc} />}
        </div>

        <AuthIntro />

        <div className="grid grid-cols-1 md:grid-cols-[240px_minmax(0,1fr)] gap-8 mt-10">
          <aside className="api-sidebar">
            <div className="sticky top-24 pr-6">
              <div className="text-[11px] font-medium text-gray-500 mb-2">Tags</div>
              <ul className="space-y-2 text-sm min-h-[200px]">
                {tagOrder.length > 0 ? (
                  tagOrder.map((tag) => (
                    <li key={tag}>
                      <a href={`#tag-${encodeURIComponent(tag)}`} className="text-gray-600 hover:text-gray-900">{tag}</a>
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
                    <div key={tag} id={`tag-${tag}`}>
                      <h2 className="text-base font-semibold mb-3 text-gray-900">{tag}</h2>
                      <div className="space-y-4">
                        {eps.map((e) => (
                          <EndpointCard key={`${e.method}-${e.path}`} ep={e} doc={doc} />
                        ))}
                      </div>
                    </div>
                  );
                })
              ) : (
                <div className="text-gray-500">Loading API…</div>
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
    <div className="mt-2 text-sm text-gray-600">
      Base URL: <code className="text-gray-900">{base}</code>
    </div>
  );
}

function AuthIntro() {
  const origin = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8080';
  const apiBase = `${origin}/api/v1`;
  const jwtLogin = [
    'curl -X POST',
    `'${apiBase}/auth/login'`,
    "-H 'Content-Type: application/json'",
    "-d '{\"username\":\"alice\",\"password\":\"your-password\"}'",
  ].join(' ');
  const jwtExample = [
    'curl -X GET',
    `'${apiBase}/transcription/list'`,
    "-H 'Authorization: Bearer YOUR_JWT'",
  ].join(' ');
  const apiKeyExample = [
    'curl -X GET',
    `'${apiBase}/transcription/list'`,
    "-H 'X-API-Key: YOUR_API_KEY'",
  ].join(' ');

  return (
    <section className="container-narrow">
      <div className="grid md:grid-cols-2 gap-6">
        <article className="api-card">
          <h3 className="text-sm font-semibold text-gray-900">Authentication</h3>
          <p className="text-sm text-gray-600 mt-2">
            Protected endpoints accept either a Bearer JWT in the <code>Authorization</code> header or an API key via the <code>X-API-Key</code> header. Some endpoints require JWT specifically (user account and LLM config).
          </p>
          <ul className="list-disc pl-5 text-sm text-gray-600 mt-2">
            <li>JWT-only: <code>/auth/change-password</code>, <code>/auth/change-username</code>, <code>/api-keys</code>, <code>/llm/config</code></li>
            <li>API key or JWT: transcription, chat, notes, summaries, summarize, admin</li>
          </ul>
        </article>

        <article className="api-card">
          <h3 className="text-sm font-semibold text-gray-900">Get a JWT</h3>
          <p className="text-sm text-gray-600 mt-2">Authenticate and use the token with the Authorization header.</p>
          <div className="mt-3"><CodeTabs curl={jwtLogin} js={curlToFetch(jwtLogin)} wrap /></div>
        </article>

        <article className="api-card md:col-span-1">
          <h3 className="text-sm font-semibold text-gray-900">Use JWT</h3>
          <div className="mt-3"><CodeTabs curl={jwtExample} js={curlToFetch(jwtExample)} wrap /></div>
        </article>

        <article className="api-card md:col-span-1">
          <h3 className="text-sm font-semibold text-gray-900">Use API Key</h3>
          <div className="mt-3"><CodeTabs curl={apiKeyExample} js={curlToFetch(apiKeyExample)} wrap /></div>
        </article>
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

  // Security extraction (supports Swagger2 and OAS3)
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
  const examplePayload = exampleSchema ? makeExample(exampleSchema, doc.components) : undefined;

  // Choose first acceptable security for curl example
  const preferredSec = endpointSecurity[0];
  const authHeaders = makeAuthHeaders(preferredSec, securityDefs);
  const curl = makeCurl(ep.method, ep.path, exampleContentType, examplePayload, authHeaders, parameters);

  return (
    <article id={id} className="api-card">
      <button onClick={() => setExpanded((v) => !v)} className="w-full text-left">
        <div className="flex items-start gap-3">
          <MethodBadge method={ep.method} />
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <code className="text-sm text-gray-900 truncate">{ep.path}</code>
              <a href={`#${id}`} title="Permalink" className="ml-auto text-gray-400 hover:text-gray-700">#</a>
            </div>
            {ep.summary && <div className="text-[13px] text-gray-600 mt-0.5 truncate">{ep.summary}</div>}
          </div>
        </div>
      </button>
      {expanded && (
        <div className="pt-4">
          {ep.description && (
            <p className="text-sm text-gray-600">{ep.description}</p>
          )}

          {!!endpointSecurity.length && (
            <div className="mt-3 text-xs text-gray-600 flex flex-wrap items-center gap-2">
              <span className="text-gray-500">Auth:</span>
              {endpointSecurity.map((name) => (
                <AuthPill key={name} name={name} def={securityDefs?.[name]} />
              ))}
            </div>
          )}

          {!!parameters.length && (
            <div className="mt-4">
              <h4 className="api-section-title mb-2">Parameters</h4>
              <ParamGroups params={showAllParams ? parameters : parameters.slice(0, 8)} />
              {parameters.length > 8 && (
                <button onClick={() => setShowAllParams((v) => !v)} className="mt-2 text-xs text-gray-600 hover:text-gray-900">
                  {showAllParams ? 'Show less' : `Show all ${parameters.length} parameters`}
                </button>
              )}
            </div>
          )}

          {exampleSchema && (
            <div className="mt-3">
              <h4 className="api-section-title mb-2">Request Body <span className="text-gray-500">({exampleContentType})</span></h4>
              <CodeBlock text={formatMaybeJSON(examplePayload)} />
            </div>
          )}

          {responses && Object.keys(responses).length > 0 && (
            <div className="mt-3">
              <h4 className="api-section-title mb-2">Response</h4>
              {(() => {
                const entries = Object.entries(responses);
                const idx = entries.findIndex(([c]) => c === '200');
                const primary = entries[idx >= 0 ? idx : 0] as any;
                const others = entries.filter((_, i) => i !== (idx >= 0 ? idx : 0));
                const [code, r] = primary;
                const content = (r?.content || {}) as any;
                const firstCT = Object.keys(content)[0];
                const schema = firstCT ? content[firstCT]?.schema : r?.schema;
                const ex = schema ? makeExample(schema, doc.components) : (r?.description || '');
                return (
                  <>
                    <div className="rounded-md border border-gray-200 p-3">
                      <div className="flex items-center gap-2 text-[11px] text-gray-600 mb-2">
                        <span className="status-pill">HTTP {code}</span>
                        {r?.description && <span>{r.description}</span>}
                      </div>
                      <CodeBlock text={formatMaybeJSON(ex)} />
                    </div>
                    {others.length > 0 && (
                      <button onClick={() => setShowAllResponses((v) => !v)} className="mt-2 text-xs text-gray-600 hover:text-gray-900">
                        {showAllResponses ? 'Hide other responses' : `Show ${others.length} more`}
                      </button>
                    )}
                    {showAllResponses && (
                      <div className="space-y-2 mt-2">
                        {others.map(([oc, or]: any) => {
                          const ocContent = (or?.content || {}) as any;
                          const ocCT = Object.keys(ocContent)[0];
                          const ocSchema = ocCT ? ocContent[ocCT]?.schema : or?.schema;
                          const ocEx = ocSchema ? makeExample(ocSchema, doc.components) : (or?.description || '');
                          return (
                            <div key={oc} className="rounded-md border border-gray-200 p-3">
                              <div className="flex items-center gap-2 text-[11px] text-gray-600 mb-2">
                                <span className="status-pill">HTTP {oc}</span>
                                {or?.description && <span>{or.description}</span>}
                              </div>
                              <CodeBlock text={formatMaybeJSON(ocEx)} />
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </>
                );
              })()}
            </div>
          )}

          <div className="mt-3">
            <h4 className="api-section-title mb-2">Examples</h4>
            <CodeTabs curl={curl} js={curlToFetch(curl)} />
          </div>
        </div>
      )}
    </article>
  );
}

function CodeBlock({ text, wrap }: { text: any; language?: string; wrap?: boolean }) {
  const value = typeof text === 'string' ? text : JSON.stringify(text, null, 2);
  const [copied, setCopied] = useState(false);
  return (
    <div className="relative">
      <pre className={`codeblock ${wrap ? 'codeblock-wrap' : ''}`}><code>{value}</code></pre>
      <button
        onClick={() => {
          navigator.clipboard.writeText(value).then(() => {
            setCopied(true);
            setTimeout(() => setCopied(false), 1200);
          });
        }}
        className="absolute top-2 right-2 inline-flex items-center gap-2 rounded-md border border-gray-300 bg-white text-gray-700 px-2 py-1 text-[11px] hover:bg-gray-50"
      >
        {copied ? 'Copied' : 'Copy'}
      </button>
    </div>
  );
}

function CodeTabs({ curl, js, wrap }: { curl: string; js: string; wrap?: boolean }) {
  const [tab, setTab] = useState<'curl' | 'js'>('curl');
  return (
    <div>
      <div className="mb-2 inline-flex rounded-md border border-gray-200 bg-white p-0.5 text-xs">
        <button onClick={() => setTab('curl')} className={`px-2 py-1 rounded ${tab === 'curl' ? 'bg-gray-100 text-gray-900' : 'text-gray-600'}`}>cURL</button>
        <button onClick={() => setTab('js')} className={`px-2 py-1 rounded ${tab === 'js' ? 'bg-gray-100 text-gray-900' : 'text-gray-600'}`}>JavaScript</button>
      </div>
      {tab === 'curl' ? <CodeBlock text={curl} wrap={wrap} /> : <CodeBlock text={js} wrap={wrap} />}
    </div>
  );
}

function MethodBadge({ method }: { method: string }) {
  const styles: Record<string, string> = {
    GET: 'bg-emerald-50 text-emerald-700 ring-emerald-200',
    POST: 'bg-blue-50 text-blue-700 ring-blue-200',
    PUT: 'bg-amber-50 text-amber-700 ring-amber-200',
    PATCH: 'bg-cyan-50 text-cyan-700 ring-cyan-200',
    DELETE: 'bg-rose-50 text-rose-700 ring-rose-200',
  };
  const cls = styles[method] || 'bg-gray-100 text-gray-700 ring-gray-200';
  return <span className={`pill ${cls} min-w-[54px] text-center`}>{method}</span>;
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
    <div className="space-y-3">
      {order.map((k) => groups[k] && groups[k].length ? (
        <div key={k}>
          <div className="text-[11px] font-medium text-gray-500 mb-1">{k.charAt(0).toUpperCase() + k.slice(1)} parameters</div>
          <ul className="space-y-1">
            {groups[k].map((p) => {
              const schema = p.schema || {};
              const type = schema.type || (schema.$ref ? refName(schema.$ref) : p.type || '');
              const bits: string[] = [];
              if (schema.default !== undefined || p.default !== undefined) bits.push(`default: ${schema.default ?? p.default}`);
              if (schema.enum) bits.push(`enum: ${(schema.enum || []).join(', ')}`);
              const desc = [p.description, ...bits].filter(Boolean).join(' — ');
              return (
                <li key={`${p.name}-${p.in}`} className="text-xs text-gray-700">
                  <span className="inline-flex items-center gap-1">
                    <code className="px-1 py-0.5 rounded bg-gray-100 border border-gray-200 text-gray-900">{p.name}</code>
                    <span className="text-gray-500">{type}</span>
                    {p.required && <span className="text-gray-500">• required</span>}
                  </span>
                  {desc && <span className="text-gray-600"> — {desc}</span>}
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
  const m = ref.match(/#\/components\/(schemas|requestBodies)\/(.+)$/);
  return m ? m[2] : ref;
}

function resolveRef(obj: any, components?: SwaggerDoc['components']): any {
  if (!obj || !obj.$ref) return obj;
  const name = refName(obj.$ref);
  return components?.schemas?.[name] || components?.requestBodies?.[name] || obj;
}

function resolveRequestBody(rb: any, doc: SwaggerDoc): any {
  if (!rb) return undefined;
  if (rb.$ref) return resolveRef(rb, doc.components);
  return rb;
}

// Swagger 2.0 fallback: detect body parameter and construct a pseudo requestBody
function legacyBodyFromSwagger2(meta: any) {
  if (!meta?.parameters) return undefined;
  const body = meta.parameters.find((p: any) => p.in === 'body');
  if (!body) return undefined;
  return {
    content: {
      'application/json': { schema: body.schema || {} },
    },
  };
}

function makeExample(
  schema: any,
  components?: SwaggerDoc['components'],
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
    return makeExample(resolveRef(schema, components), components, depth + 1, seen);
  }

  const type = schema.type;
  if (schema.examples && Array.isArray(schema.examples) && schema.examples.length) return schema.examples[0];

  if (type === 'object' || (schema.properties && typeof schema.properties === 'object')) {
    const obj: any = {};
    const props = schema.properties || {};
    let count = 0;
    for (const [k, v] of Object.entries<any>(props)) {
      obj[k] = makeExample(v, components, depth + 1, new Set(seen));
      if (++count > 15) break; // cap object size
    }
    return obj;
  }

  if (type === 'array') {
    return [makeExample(schema.items || {}, components, depth + 1, new Set(seen))];
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
    // Very lightweight parse for "curl -X METHOD 'URL' -H 'Header: v' -d '...'
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
  // Swagger2 apiKey type
  if (def.type === 'apiKey') {
    const name = def.name || (def.description?.includes('Authorization') ? 'Authorization' : 'X-API-Key');
    if (name.toLowerCase() === 'authorization') return ['Authorization: Bearer YOUR_JWT'];
    return [`${name}: YOUR_API_KEY`];
  }
  // HTTP bearer (OAS3)
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
    <span className="inline-flex items-center gap-1 rounded-full bg-gray-200 text-gray-800 px-2 py-0.5">
      {label}
    </span>
  );
}

function SchemaTable({ schema, doc }: { schema: any; doc: SwaggerDoc }) {
  const rows = schemaToRows(schema, doc.components);
  if (!rows.length) return <p className="text-sm text-gray-600">No fields.</p>;
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="text-left text-gray-500">
            <th className="py-1 pr-4">Field</th>
            <th className="py-1 pr-4">Type</th>
            <th className="py-1 pr-4">Required</th>
            <th className="py-1">Description</th>
          </tr>
        </thead>
        <tbody className="align-top">
          {rows.map((r) => (
            <tr key={r.name} className="border-t border-gray-100">
              <td className="py-2 pr-4 text-gray-900"><code>{r.name}</code></td>
              <td className="py-2 pr-4 text-gray-600">{r.type}</td>
              <td className="py-2 pr-4 text-gray-600">{r.required ? 'yes' : 'no'}</td>
              <td className="py-2 text-gray-600">{r.description}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function schemaToRows(schema: any, components?: SwaggerDoc['components'], prefix = '', parentRequired: string[] = []): any[] {
  const rows: any[] = [];
  const resolved = resolveRef(schema, components);
  const req: string[] = resolved.required || parentRequired || [];
  if (resolved.type === 'object' || resolved.properties) {
    const props = resolved.properties || {};
    for (const [k, v] of Object.entries<any>(props)) {
      const name = prefix ? `${prefix}.${k}` : k;
      const sub = resolveRef(v, components);
      const type = sub.type || (sub.$ref ? refName(sub.$ref) : (sub.items ? `${sub.type}[]` : 'object'));
      const description = sub.description || '';
      const required = req.includes(k);
      rows.push({ name, type, required, description });
      // nested
      if (sub.type === 'object' || sub.properties) {
        rows.push(...schemaToRows(sub, components, name, sub.required || []));
      } else if (sub.type === 'array' && sub.items) {
        const it = resolveRef(sub.items, components);
        if (it.type === 'object' || it.properties || it.$ref) {
          rows.push(...schemaToRows(it, components, `${name}[]`, it.required || []));
        }
      }
    }
  }
  return rows;
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

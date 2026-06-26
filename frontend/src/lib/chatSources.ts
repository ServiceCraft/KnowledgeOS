import type { ChatMessage, ChatSource } from '@/types';

const CITATION_RE = /\[([^\[\]\n]+)\]/g;
const USED_SOURCES_FOOTER_RE = /использованн/i;

function looksLikeSourceId(token: string): boolean {
  return (token.match(/:/g) ?? []).length >= 2;
}

export function buildSourceUrl(entityType: string, entityId: string): string | undefined {
  switch (entityType) {
    case 'qa':
      return `/kb/qa/${entityId}`;
    case 'article':
      return `/kb/articles/${entityId}`;
    case 'pricing':
      return '/kb/pricing';
    default:
      return undefined;
  }
}

export function sourcesById(sources: ChatSource[]): Map<string, ChatSource> {
  const map = new Map<string, ChatSource>();
  for (const source of sources) {
    map.set(source.source_id, source);
  }
  return map;
}

export function extractCitationIds(content: string): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const match of content.matchAll(CITATION_RE)) {
    const inner = match[1];
    for (const part of inner.split(',')) {
      const token = part.trim();
      if (!token || !looksLikeSourceId(token) || seen.has(token)) continue;
      seen.add(token);
      out.push(token);
    }
  }
  return out;
}

export function resolveCitedSourceIds(content: string, citedSourceIds?: string[]): string[] {
  if (citedSourceIds && citedSourceIds.length > 0) {
    return citedSourceIds;
  }
  return extractCitationIds(content);
}

export function resolveCitedSources(
  content: string,
  sources: ChatSource[],
  citedSourceIds?: string[]
): ChatSource[] {
  const byId = sourcesById(sources);
  const ids = resolveCitedSourceIds(content, citedSourceIds);
  const out: ChatSource[] = [];
  const seen = new Set<string>();
  for (const id of ids) {
    if (seen.has(id)) continue;
    const source = byId.get(id);
    if (source) {
      seen.add(id);
      out.push(source);
    }
  }
  return out;
}

function stripInlineCitations(content: string): string {
  return content.replace(CITATION_RE, (match, inner: string) => {
    const parts = inner.split(',').map((p) => p.trim()).filter(Boolean);
    if (parts.length === 0) return match;
    if (parts.every(looksLikeSourceId)) return '';
    return match;
  });
}

function isCitationOnlyLine(line: string): boolean {
  const trimmed = line.trim();
  if (!trimmed) return true;
  const withoutCitations = stripInlineCitations(trimmed).replace(/[.,;:\s]+/g, '');
  return withoutCitations.length === 0;
}

function stripCitationFooter(content: string): string {
  const lines = content.split('\n');
  while (lines.length > 0 && lines[lines.length - 1].trim() === '') {
    lines.pop();
  }
  while (lines.length > 0) {
    const last = lines[lines.length - 1].trim();
    if (!last) {
      lines.pop();
      continue;
    }
    if (USED_SOURCES_FOOTER_RE.test(last) || isCitationOnlyLine(last)) {
      lines.pop();
      continue;
    }
    break;
  }
  return lines.join('\n');
}

export function stripCitationMarkers(content: string): string {
  return stripCitationFooter(stripInlineCitations(content)).trim();
}

export function formatMessageBody(content: string): string {
  return stripCitationMarkers(content).replace(/\n{3,}/g, '\n\n');
}

export function sourcePreviewText(source: ChatSource): string {
  return source.snippet || source.content || '';
}

export const SOURCE_TYPE_LABELS: Record<string, string> = {
  qa: 'Q&A',
  article: 'Статья',
  pricing: 'Прайс',
};

export type ChatAnswerInput = Pick<ChatMessage, 'content' | 'sources' | 'cited_source_ids'>;

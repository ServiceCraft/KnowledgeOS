import remarkBreaks from 'remark-breaks';

/** Shared markdown preview options for viewer and editor. */
export const markdownPreviewOptions = {
  remarkPlugins: [remarkBreaks],
};

/** Strip common markdown syntax for plain-text excerpts. */
export function stripMarkdown(text: string): string {
  return text
    .replace(/^#{1,6}\s+/gm, '')
    .replace(/\*\*(.+?)\*\*/g, '$1')
    .replace(/__(.+?)__/g, '$1')
    .replace(/\*(.+?)\*/g, '$1')
    .replace(/_(.+?)_/g, '$1')
    .replace(/`(.+?)`/g, '$1')
    .replace(/^\s*[-*+]\s+/gm, '• ')
    .replace(/^\s*\d+\.\s+/gm, '');
}

/** Short plain-text preview for article cards, preserving line breaks. */
export function formatArticleExcerpt(body: string | undefined, maxLength = 160): string {
  if (!body?.trim()) return 'Пустая статья';

  const plain = stripMarkdown(body).trim();
  if (plain.length <= maxLength) return plain;

  const cut = plain.slice(0, maxLength);
  const lastNewline = cut.lastIndexOf('\n');
  if (lastNewline > maxLength * 0.4) {
    return `${cut.slice(0, lastNewline).trimEnd()}…`;
  }

  const lastSpace = cut.lastIndexOf(' ');
  if (lastSpace > maxLength * 0.6) {
    return `${cut.slice(0, lastSpace).trimEnd()}…`;
  }

  return `${cut.trimEnd()}…`;
}

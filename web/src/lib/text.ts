export const splitTags = (value: string): string[] => [
  ...new Set(
    value
      .split(/[,，]/)
      .map((tag) => tag.trim())
      .filter(Boolean),
  ),
];

export interface LinkSegment {
  type: "text" | "link";
  value: string;
  href?: string;
}

const URL_PATTERN = /https?:\/\/[^\s<>"']+/g;
const TRAILING = /[),.;!?，。；！？、）】》]+$/;

export function linkify(text: string): LinkSegment[] {
  const segments: LinkSegment[] = [];
  let lastIndex = 0;

  for (const match of text.matchAll(URL_PATTERN)) {
    const index = match.index ?? 0;

    if (index > lastIndex) {
      segments.push({ type: "text", value: text.slice(lastIndex, index) });
    }

    let value = match[0];
    const trailing = value.match(TRAILING)?.[0] ?? "";

    if (trailing) {
      value = value.slice(0, -trailing.length);
    }

    if (value.length > "https://".length) {
      segments.push({ type: "link", value, href: value });
    } else {
      segments.push({ type: "text", value });
    }

    if (trailing) {
      segments.push({ type: "text", value: trailing });
    }

    lastIndex = index + match[0].length;
  }

  if (lastIndex < text.length) {
    segments.push({ type: "text", value: text.slice(lastIndex) });
  }

  return segments;
}

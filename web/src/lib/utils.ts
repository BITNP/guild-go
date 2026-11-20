import type { Department, ProgressItem } from "./api";

export const progressKey = (item: ProgressItem): string =>
  item.kind === "comment"
    ? `comment:${item.comment.id}`
    : `description_change:${item.descriptionChange.id}`;

export function departmentLabel(
  department: Department | string | null | undefined,
): string {
  switch (department) {
    case "techdept": {
      return "技术部";
    }
    case "clinic": {
      return "电脑诊所";
    }
    case "digimedia": {
      return "数字媒体中心";
    }
    case "techsvc": {
      return "技术保障中心";
    }
    default: {
      return "";
    }
  }
}

export function formatDateTime(value: string | null | undefined): string {
  if (!value) {
    return "—";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "—";
  }
  const sameYear = date.getFullYear() === new Date().getFullYear();

  return new Intl.DateTimeFormat("zh-CN", {
    year: sameYear ? undefined : "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(date);
}

export function timeAgo(value: string | null | undefined): string {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const diffSeconds = Math.round((date.getTime() - Date.now()) / 1000);
  const abs = Math.abs(diffSeconds);
  if (abs < 45) {
    return "刚刚";
  }
  const formatter = new Intl.RelativeTimeFormat("zh-CN", { numeric: "auto" });
  const ranges: [Intl.RelativeTimeFormatUnit, number][] = [
    ["year", 31_536_000],
    ["month", 2_629_800],
    ["week", 604_800],
    ["day", 86_400],
    ["hour", 3600],
    ["minute", 60],
  ];
  for (const [unit, seconds] of ranges) {
    if (abs >= seconds) {
      return formatter.format(Math.round(diffSeconds / seconds), unit);
    }
  }

  return "刚刚";
}

export function isOverdue(
  dueAt: string | null,
  completedAt: string | null,
): boolean {
  if (!dueAt || completedAt) {
    return false;
  }
  const due = new Date(dueAt).getTime();

  return !Number.isNaN(due) && due < Date.now();
}

export const splitTags = (value: string): string[] => [
  ...new Set(
    value
      .split(/[,，]/)
      .map((tag) => tag.trim())
      .filter(Boolean),
  ),
];

export function errorMessage(
  error: unknown,
  fallback = "操作失败，请稍后重试",
): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return fallback;
}

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

export const reactionEmoji: Record<string, { symbol: string; label: string }> =
  {
    thumbs_up: { symbol: "👍", label: "赞" },
    thumbs_down: { symbol: "👎", label: "踩" },
    laugh: { symbol: "😄", label: "笑" },
    hooray: { symbol: "🎉", label: "庆祝" },
    confused: { symbol: "😕", label: "疑惑" },
    heart: { symbol: "❤️", label: "喜欢" },
    rocket: { symbol: "🚀", label: "推进" },
    eyes: { symbol: "👀", label: "关注" },
  };

export const reactionKeys = Object.keys(reactionEmoji);

export function safeReturnTo(value: unknown): string {
  if (typeof value !== "string") {
    return "/";
  }
  if (!value.startsWith("/") || value.startsWith("//")) {
    return "/";
  }
  if (/[\\\s]/.test(value) || hasAsciiControlChar(value)) {
    return "/";
  }
  try {
    const url = new URL(value, "https://guild.invalid");
    if (url.origin !== "https://guild.invalid") {
      return "/";
    }
    if (url.pathname === "/login" || url.pathname.startsWith("/api/")) {
      return "/";
    }

    return `${url.pathname}${url.search}${url.hash}`;
  } catch {
    return "/";
  }
}

function hasAsciiControlChar(value: string): boolean {
  for (let i = 0; i < value.length; i++) {
    const code = value.charCodeAt(i);
    if (code <= 0x1f || code === 0x7f) {
      return true;
    }
  }

  return false;
}

import * as THREE from "three";

import type { ProgressItem } from "@/api";
import type { ThemeMode } from "@/composables/useTheme";

export const CARD_CANVAS_WIDTH = 1024;
export const CARD_CANVAS_HEIGHT = 400;
export const REACTION_ROW_TOP = 300;
export const REACTION_ROW_BOTTOM = 374;
export const REACTION_START_X = 34;
export const REACTION_CELL_WIDTH = 110;
export const REACTION_GAP = 10;

const CARD_PALETTES = {
  dark: {
    surface: "rgba(8, 14, 17, 0.9)",
    author: "#f3c35f",
    meta: "rgba(178, 201, 208, 0.68)",
    body: "rgba(240, 244, 238, 0.94)",
    accentSoft: "rgba(239, 180, 66, 0.92)",
    cyan: "rgba(102, 219, 186, 0.82)",
    divider: "rgba(139, 166, 171, 0.2)",
    empty: "rgba(232, 185, 85, 0.86)",
  },
  light: {
    surface: "rgba(255, 255, 255, 0.94)",
    author: "#a56812",
    meta: "rgba(69, 87, 101, 0.74)",
    body: "rgba(23, 32, 43, 0.96)",
    accentSoft: "rgba(181, 120, 24, 0.9)",
    cyan: "rgba(21, 124, 89, 0.9)",
    divider: "rgba(82, 105, 119, 0.18)",
    empty: "rgba(133, 83, 12, 0.86)",
  },
} as const;

export function createProgressTexture(
  item: ProgressItem,
  theme: ThemeMode,
): THREE.CanvasTexture {
  const canvas = document.createElement("canvas");
  canvas.width = CARD_CANVAS_WIDTH;
  canvas.height = CARD_CANVAS_HEIGHT;
  const context = canvas.getContext("2d");
  if (!context) {
    throw new Error("Canvas 2D context is unavailable");
  }
  const palette = CARD_PALETTES[theme];

  context.clearRect(0, 0, canvas.width, canvas.height);
  context.fillStyle = palette.surface;
  context.fillRect(0, 0, canvas.width, canvas.height);
  context.fillStyle = palette.accentSoft;
  context.fillRect(0, 0, 6, canvas.height);
  context.fillStyle = palette.cyan;
  context.fillRect(canvas.width - 4, 0, 4, canvas.height);

  context.font = '600 46px "Noto Sans SC", "PingFang SC", sans-serif';
  context.fillStyle = palette.author;
  context.textBaseline = "middle";
  const author =
    item.kind === "comment"
      ? item.comment.author
      : item.descriptionChange.author;
  const createdAt =
    item.kind === "comment"
      ? item.comment.createdAt
      : item.descriptionChange.createdAt;
  const authorLabel =
    item.kind === "description_change"
      ? `${author.userName} · 描述更新`
      : author.userName;
  context.fillText(trimToWidth(context, authorLabel, 590, 46), 34, 42);
  context.font = '34px "Noto Sans SC", "PingFang SC", sans-serif';
  context.fillStyle = palette.meta;
  const time = new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(new Date(createdAt));
  context.textAlign = "right";
  context.fillText(`${time} · 展开`, 990, 42);
  context.textAlign = "left";

  context.font = '44px "Noto Sans SC", "PingFang SC", sans-serif';
  context.fillStyle = palette.body;
  const comment = item.kind === "comment" ? item.comment : null;
  const change =
    item.kind === "description_change" ? item.descriptionChange : null;
  const rawBody = comment?.body ?? change?.after ?? "";
  const body = rawBody.trim()
    ? rawBody
    : comment
      ? comment.attachments.length > 0
        ? "图片进展"
        : "无正文"
      : "清空了节点描述";
  const wrapped = wrapTextPreserving(context, body, 940, 4);
  for (const [index, line] of wrapped.lines.entries()) {
    context.fillText(line, 34, 96 + index * 52);
  }
  if (comment && comment.attachments.length > 0) {
    context.font = '26px "Noto Sans SC", "PingFang SC", sans-serif';
    context.fillStyle = palette.cyan;
    context.textAlign = "right";
    context.fillText(`附 ${comment.attachments.length} 张图片`, 990, 276);
    context.textAlign = "left";
  }
  context.strokeStyle = palette.accentSoft;
  context.lineWidth = 1;
  context.beginPath();
  context.moveTo(34, 76);
  context.lineTo(990, 76);
  context.stroke();
  context.strokeStyle = palette.divider;
  context.beginPath();
  context.moveTo(34, 288);
  context.lineTo(990, 288);
  context.stroke();

  if (comment) {
    const counts = new Map(
      comment.reactions
        .filter((reaction) => reaction.count > 0)
        .map((reaction) => [reaction.emoji, reaction]),
    );
    for (const [index, emoji] of reactionKeys.entries()) {
      const x = REACTION_START_X + index * (REACTION_CELL_WIDTH + REACTION_GAP);
      const reaction = counts.get(emoji);
      context.fillStyle = reaction?.reactedByMe
        ? "rgba(224, 174, 66, 0.27)"
        : reaction
          ? "rgba(75, 201, 174, 0.15)"
          : "rgba(91, 119, 124, 0.1)";
      context.fillRect(x, REACTION_ROW_TOP, REACTION_CELL_WIDTH, 72);
      context.strokeStyle = reaction?.reactedByMe
        ? "rgba(241, 192, 87, 0.78)"
        : reaction
          ? "rgba(98, 218, 186, 0.48)"
          : "rgba(103, 137, 142, 0.24)";
      context.strokeRect(
        x + 0.5,
        REACTION_ROW_TOP + 0.5,
        REACTION_CELL_WIDTH - 1,
        71,
      );

      context.font = '32px "Apple Color Emoji", "Segoe UI Emoji", sans-serif';
      context.textAlign = "center";
      context.textBaseline = "middle";
      context.fillStyle = reaction
        ? "rgba(244, 247, 239, 0.98)"
        : "rgba(176, 196, 198, 0.48)";
      context.fillText(
        reactionEmoji[emoji]?.symbol ?? emoji,
        x + (reaction ? 40 : REACTION_CELL_WIDTH / 2),
        REACTION_ROW_TOP + 36,
      );
      if (reaction) {
        context.font = '600 25px "Noto Sans SC", "PingFang SC", sans-serif';
        context.fillStyle = reaction.reactedByMe
          ? "rgba(255, 220, 141, 0.96)"
          : "rgba(133, 226, 203, 0.9)";
        context.textAlign = "left";
        context.fillText(String(reaction.count), x + 66, REACTION_ROW_TOP + 39);
      }
    }
  } else {
    context.fillStyle = "rgba(75, 201, 174, 0.1)";
    context.fillRect(34, REACTION_ROW_TOP, 956, 72);
    context.strokeStyle = "rgba(98, 218, 186, 0.35)";
    context.strokeRect(34.5, REACTION_ROW_TOP + 0.5, 955, 71);
    context.font = '600 27px "Noto Sans SC", "PingFang SC", sans-serif';
    context.fillStyle = "rgba(116, 224, 198, 0.88)";
    context.textAlign = "center";
    context.textBaseline = "middle";
    context.fillText("点击展开完整描述变更", 512, REACTION_ROW_TOP + 37);
  }
  context.textAlign = "left";
  context.textBaseline = "middle";

  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  texture.minFilter = THREE.LinearMipmapLinearFilter;
  texture.magFilter = THREE.LinearFilter;

  return texture;
}

export function createEmptyTexture(theme: ThemeMode): THREE.CanvasTexture {
  const canvas = document.createElement("canvas");
  canvas.width = 1024;
  canvas.height = 160;
  const context = canvas.getContext("2d");
  if (!context) {
    throw new Error("Canvas 2D context is unavailable");
  }
  context.clearRect(0, 0, canvas.width, canvas.height);
  context.font = '600 52px "Noto Sans SC", "PingFang SC", sans-serif';
  context.fillStyle = CARD_PALETTES[theme].empty;
  context.textAlign = "center";
  context.textBaseline = "middle";
  context.fillText("该节点还没有发布进展", canvas.width / 2, canvas.height / 2);
  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  texture.minFilter = THREE.LinearMipmapLinearFilter;
  texture.magFilter = THREE.LinearFilter;

  return texture;
}

function wrapTextPreserving(
  context: CanvasRenderingContext2D,
  text: string,
  maxWidth: number,
  maxLines: number,
): { lines: string[]; truncated: boolean } {
  const lines: string[] = [];
  const logicalLines = text
    .replace(/\r\n?/g, "\n")
    .replace(/\t/g, "    ")
    .split("\n");
  let truncated = false;

  for (let lineIndex = 0; lineIndex < logicalLines.length; lineIndex++) {
    const logicalLine = logicalLines[lineIndex] ?? "";
    if (!logicalLine) {
      lines.push("");
      if (lines.length >= maxLines) {
        truncated = lineIndex < logicalLines.length - 1;

        break;
      }

      continue;
    }

    let current = "";
    let reachedLimit = false;
    for (const character of logicalLine) {
      const candidate = current + character;
      if (!current || context.measureText(candidate).width <= maxWidth) {
        current = candidate;

        continue;
      }
      lines.push(current);
      current = character;
      if (lines.length >= maxLines) {
        truncated = true;
        reachedLimit = true;

        break;
      }
    }
    if (reachedLimit) {
      break;
    }
    if (current) {
      lines.push(current);
    }
    if (lines.length >= maxLines) {
      truncated = lineIndex < logicalLines.length - 1;

      break;
    }
  }

  if (lines.length > maxLines) {
    lines.length = maxLines;
  }
  if (truncated && lines.length > 0) {
    let last = lines[maxLines - 1] ?? "";
    while (last && context.measureText(`${last}…`).width > maxWidth) {
      last = [...last].slice(0, -1).join("");
    }
    lines[maxLines - 1] = `${last}…`;
  }

  return { lines, truncated };
}

function trimToWidth(
  context: CanvasRenderingContext2D,
  text: string,
  maxWidth: number,
  fontSize: number,
): string {
  context.font = `600 ${fontSize}px "Noto Sans SC", "PingFang SC", sans-serif`;
  if (context.measureText(text).width <= maxWidth) {
    return text;
  }
  let value = text;
  while (value && context.measureText(`${value}…`).width > maxWidth) {
    value = [...value].slice(0, -1).join("");
  }

  return `${value}…`;
}

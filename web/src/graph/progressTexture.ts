import * as THREE from "three";

import type { ThemeMode } from "@/composables/useTheme";
import type { ProgressItem } from "@/lib/api";
import { progressKey, reactionEmoji, reactionKeys } from "@/lib/utils";

import { CARD_PALETTES } from "./palette";
import { ellipsizeText } from "./visuals";

export const COMMENT_CARD_WIDTH = 6.2;
export const COMMENT_CARD_HEIGHT = 2.36;

const CARD_CANVAS_WIDTH = 1024;
const CARD_CANVAS_HEIGHT = 400;
const REACTION_ROW_TOP = 300;
const REACTION_ROW_BOTTOM = 374;
const REACTION_START_X = 34;
const REACTION_CELL_WIDTH = 110;
const REACTION_GAP = 10;

export function pickReactionEmojiAtUV(uv: THREE.Vector2): string | null {
  const x = uv.x * CARD_CANVAS_WIDTH;
  const y = (1 - uv.y) * CARD_CANVAS_HEIGHT;
  if (y < REACTION_ROW_TOP || y > REACTION_ROW_BOTTOM) {
    return null;
  }
  const slotWidth = REACTION_CELL_WIDTH + REACTION_GAP;
  const index = Math.floor((x - REACTION_START_X) / slotWidth);
  const withinSlot = (x - REACTION_START_X) % slotWidth <= REACTION_CELL_WIDTH;
  if (index < 0 || index >= reactionKeys.length || !withinSlot) {
    return null;
  }

  return reactionKeys[index] ?? null;
}

export function progressTextureKey(item: ProgressItem): string {
  if (item.kind === "description_change") {
    const change = item.descriptionChange;

    return [
      progressKey(item),
      change.author.userName,
      change.createdAt,
      change.before,
      change.after,
    ].join("|");
  }
  const comment = item.comment;
  const reactions = comment.reactions
    .map(
      (reaction) =>
        `${reaction.emoji}:${reaction.count}:${reaction.reactedByMe ? 1 : 0}`,
    )
    .join(",");

  return [
    progressKey(item),
    comment.author.userName,
    comment.createdAt,
    comment.body,
    comment.attachments.length,
    reactions,
  ].join("|");
}

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
  context.fillText(ellipsizeText(context, authorLabel, 590), 34, 42);
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
  const lines = wrapTextPreserving(context, body, 940, 4);
  for (const [index, line] of lines.entries()) {
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
        ? palette.reactionActiveBg
        : reaction
          ? palette.reactionBg
          : palette.reactionIdleBg;
      context.fillRect(x, REACTION_ROW_TOP, REACTION_CELL_WIDTH, 72);
      context.strokeStyle = reaction?.reactedByMe
        ? palette.reactionActiveBorder
        : reaction
          ? palette.reactionBorder
          : palette.reactionIdleBorder;
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
        ? palette.reactionEmojiActive
        : palette.reactionEmojiIdle;
      context.fillText(
        reactionEmoji[emoji]?.symbol ?? emoji,
        x + (reaction ? 40 : REACTION_CELL_WIDTH / 2),
        REACTION_ROW_TOP + 36,
      );
      if (reaction) {
        context.font = '600 25px "Noto Sans SC", "PingFang SC", sans-serif';
        context.fillStyle = reaction.reactedByMe
          ? palette.reactionCountActive
          : palette.reactionCount;
        context.textAlign = "left";
        context.fillText(String(reaction.count), x + 66, REACTION_ROW_TOP + 39);
      }
    }
  } else {
    context.fillStyle = palette.changePromptBg;
    context.fillRect(34, REACTION_ROW_TOP, 956, 72);
    context.strokeStyle = palette.changePromptBorder;
    context.strokeRect(34.5, REACTION_ROW_TOP + 0.5, 955, 71);
    context.font = '600 27px "Noto Sans SC", "PingFang SC", sans-serif';
    context.fillStyle = palette.changePromptText;
    context.textAlign = "center";
    context.textBaseline = "middle";
    context.fillText("点击展开完整描述变更", 512, REACTION_ROW_TOP + 37);
  }
  context.textAlign = "left";
  context.textBaseline = "middle";

  return createCanvasTexture(canvas);
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

  return createCanvasTexture(canvas);
}

function createCanvasTexture(canvas: HTMLCanvasElement): THREE.CanvasTexture {
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
): string[] {
  const lines: string[] = [];
  const logicalLines = text
    .replace(/\r\n?/g, "\n")
    .replace(/\t/g, "    ")
    .split("\n");

  for (let lineIndex = 0; lineIndex < logicalLines.length; lineIndex++) {
    const logicalLine = logicalLines[lineIndex] ?? "";
    const hasMoreLogicalLines = lineIndex < logicalLines.length - 1;
    if (!logicalLine) {
      lines.push("");
      if (lines.length >= maxLines) {
        return finalizeWrappedLines(
          lines,
          context,
          maxWidth,
          maxLines,
          hasMoreLogicalLines,
        );
      }

      continue;
    }

    let current = "";
    for (const character of logicalLine) {
      const candidate = current + character;
      if (!current || context.measureText(candidate).width <= maxWidth) {
        current = candidate;

        continue;
      }
      lines.push(current);
      current = character;
      if (lines.length >= maxLines) {
        return finalizeWrappedLines(lines, context, maxWidth, maxLines, true);
      }
    }
    if (current) {
      lines.push(current);
    }
    if (lines.length >= maxLines) {
      return finalizeWrappedLines(
        lines,
        context,
        maxWidth,
        maxLines,
        hasMoreLogicalLines,
      );
    }
  }

  return lines;
}

function finalizeWrappedLines(
  lines: string[],
  context: CanvasRenderingContext2D,
  maxWidth: number,
  maxLines: number,
  truncated: boolean,
): string[] {
  if (lines.length > maxLines) {
    lines.length = maxLines;
  }
  if (truncated && lines.length > 0) {
    lines[maxLines - 1] = ellipsizeText(
      context,
      lines[maxLines - 1] ?? "",
      maxWidth,
    );
  }

  return lines;
}

import type { ProgressItem } from "@/api";

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

export const progressKey = (item: ProgressItem): string =>
  item.kind === "comment"
    ? `comment:${item.comment.id}`
    : `description_change:${item.descriptionChange.id}`;

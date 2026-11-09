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

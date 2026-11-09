import type { UserSummary } from "./auth";

export interface Reaction {
  emoji: string;
  count: number;
  reactedByMe: boolean;
}

export interface Attachment {
  id: number;
  url: string;
  contentType: string;
  sizeBytes: number;
}

export interface Comment {
  id: number;
  nodeId: number;
  author: UserSummary;
  body: string;
  createdAt: string;
  attachments: Attachment[];
  reactions: Reaction[];
  canDelete: boolean;
}

export interface CommentPreview {
  id: number;
  author: UserSummary;
  bodyPreview: string;
  createdAt: string;
  reactionSummary: Reaction[];
  attachmentCount: number;
}

export interface CommentList {
  items: Comment[];
  hasMore: boolean;
  nextBefore: number | null;
}

export interface DescriptionChange {
  id: number;
  nodeId: number;
  author: UserSummary;
  before: string;
  after: string;
  createdAt: string;
}

export type ProgressItem =
  | { kind: "comment"; comment: Comment }
  | { kind: "description_change"; descriptionChange: DescriptionChange };

export interface ProgressCursor {
  createdAt: string;
  kind: ProgressItem["kind"];
  id: number;
}

export interface ProgressList {
  items: ProgressItem[];
  hasMore: boolean;
  nextBefore: ProgressCursor | null;
}

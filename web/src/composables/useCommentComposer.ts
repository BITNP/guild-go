import type { Comment } from "@/api";

import {
  filesFromClipboard,
  filesFromDataTransfer,
  hasFileDrag,
  useCommentImageDrafts,
} from "./useCommentImageDrafts";

interface CommentComposerOptions {
  emptyMessage: string;
  submitError: string;
}

export function useCommentComposer(
  nodeId: MaybeRefOrGetter<number>,
  options: CommentComposerOptions,
) {
  const body = ref("");
  const submitting = ref(false);
  const error = ref<string | null>(null);
  const dragging = ref(false);
  const drafts = useCommentImageDrafts(nodeId);

  let dragDepth = 0;

  const canSubmit = computed(
    () =>
      !submitting.value &&
      !drafts.pending.value &&
      (body.value.trim().length > 0 || drafts.images.value.length > 0),
  );

  function acceptFiles(files: File[]) {
    error.value = null;
    error.value = drafts.addFiles(files);
  }

  function selectFiles(event: Event) {
    const input = event.target as HTMLInputElement;
    const selected = [...(input.files ?? [])];
    input.value = "";
    acceptFiles(selected);
  }

  function pasteImages(event: ClipboardEvent) {
    const files = filesFromClipboard(event.clipboardData);

    if (files.length === 0) {
      return;
    }

    event.preventDefault();
    acceptFiles(files);
  }

  function dragEnter(event: DragEvent) {
    if (submitting.value || !hasFileDrag(event.dataTransfer)) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();

    dragDepth += 1;
    dragging.value = true;
  }

  function dragOver(event: DragEvent) {
    if (submitting.value || !hasFileDrag(event.dataTransfer)) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();

    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = "copy";
    }

    dragging.value = true;
  }

  function dragLeave(event: DragEvent) {
    if (!dragging.value) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();

    dragDepth = Math.max(0, dragDepth - 1);

    if (dragDepth === 0) {
      dragging.value = false;
    }
  }

  function dropFiles(event: DragEvent) {
    const files = filesFromDataTransfer(event.dataTransfer);

    if (files.length === 0) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();

    dragDepth = 0;
    dragging.value = false;

    if (!submitting.value) {
      acceptFiles(files);
    }
  }

  async function submit(): Promise<Comment | null> {
    if (submitting.value) {
      return null;
    }

    if (!canSubmit.value) {
      if (!body.value.trim() && drafts.images.value.length === 0) {
        error.value = options.emptyMessage;
      }

      return null;
    }

    error.value = null;
    submitting.value = true;

    try {
      const attachmentIds = drafts.images.value.flatMap((image) =>
        image.status.state === "uploaded" ? [image.status.id] : [],
      );

      const comment = await createComment(
        toValue(nodeId),
        body.value,
        attachmentIds,
      );

      body.value = "";
      drafts.clear(false);

      return comment;
    } catch (err) {
      error.value = errorMessage(err, options.submitError);

      return null;
    } finally {
      submitting.value = false;
    }
  }

  function dispose() {
    drafts.dispose(!submitting.value);
  }

  return {
    body,
    submitting,
    error,
    dragging,
    canSubmit,
    images: drafts.images,
    pending: drafts.pending,
    retry: drafts.retry,
    remove: drafts.remove,
    selectFiles,
    pasteImages,
    dragEnter,
    dragOver,
    dragLeave,
    dropFiles,
    submit,
    dispose,
  };
}

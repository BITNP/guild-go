import type { MaybeRefOrGetter } from "vue";
import { computed, onBeforeUnmount, ref, toValue, watch } from "vue";

import {
  filesFromClipboard,
  filesFromDataTransfer,
  hasFileDrag,
  useCommentImageDrafts,
} from "@/composables/useCommentImageDrafts";
import type { Comment } from "@/lib/api";
import { createComment } from "@/lib/api";
import { errorMessage } from "@/lib/utils";

interface CommentComposerOptions {
  emptyMessage: string;
  submitErrorFallback?: string;
  onCreated: (comment: Comment) => void;
}

export function useCommentComposer(
  nodeId: MaybeRefOrGetter<number>,
  options: CommentComposerOptions,
) {
  const body = ref("");
  const submitting = ref(false);
  const error = ref<string | null>(null);
  const dragging = ref(false);
  const { images, pending, addFiles, retry, remove, clear, dispose } =
    useCommentImageDrafts(nodeId);
  const canSubmit = computed(
    () =>
      !submitting.value &&
      !pending.value &&
      (body.value.trim().length > 0 || images.value.length > 0),
  );
  let dragDepth = 0;

  function acceptFiles(files: File[]) {
    error.value = addFiles(files);
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

  async function submit() {
    if (submitting.value || pending.value) {
      return;
    }
    error.value = null;
    if (!body.value.trim() && images.value.length === 0) {
      error.value = options.emptyMessage;

      return;
    }
    submitting.value = true;
    try {
      const attachmentIds = images.value.flatMap((image) =>
        image.status.state === "uploaded" ? [image.status.id] : [],
      );
      const comment = await createComment(
        toValue(nodeId),
        body.value,
        attachmentIds,
      );
      body.value = "";
      clear(false);
      options.onCreated(comment);
    } catch (err) {
      error.value = errorMessage(err, options.submitErrorFallback);
    } finally {
      submitting.value = false;
    }
  }

  watch(body, () => {
    if (error.value && body.value.trim()) {
      error.value = null;
    }
  });

  onBeforeUnmount(() => dispose(!submitting.value));

  return {
    body,
    submitting,
    error,
    dragging,
    images,
    pending,
    canSubmit,
    retry,
    remove,
    selectFiles,
    pasteImages,
    dragEnter,
    dragOver,
    dragLeave,
    dropFiles,
    submit,
  };
}

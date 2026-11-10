export const COMMENT_IMAGE_LIMIT = 4;
export const COMMENT_IMAGE_ACCEPT = "image/jpeg,image/png,image/webp,image/gif";

const MAX_IMAGE_BYTES = 5 * 1024 * 1024;
const ACCEPTED_IMAGE_TYPES = new Set([
  "image/gif",
  "image/jpeg",
  "image/png",
  "image/webp",
]);

export interface CommentImageDraft {
  file: File;
  src: string;
  status:
    | { state: "uploading" }
    | { state: "uploaded"; id: number }
    | { state: "failed"; message: string };
}

export function filesFromClipboard(data: DataTransfer | null): File[] {
  if (!data) {
    return [];
  }
  const files = [...data.items]
    .filter((item) => item.kind === "file")
    .map((item) => item.getAsFile())
    .filter((file): file is File => file !== null);

  return files.length > 0 ? files : [...data.files];
}

export const filesFromDataTransfer = (data: DataTransfer | null): File[] =>
  data ? [...data.files] : [];

export const hasFileDrag = (data: DataTransfer | null): boolean =>
  data ? [...data.types].includes("Files") : false;

export function useCommentImageDrafts(nodeId: MaybeRefOrGetter<number>) {
  const images = ref<CommentImageDraft[]>([]);
  const pending = computed(() =>
    images.value.some((image) => image.status.state !== "uploaded"),
  );
  let active = true;

  function discard(id: number) {
    void deleteCommentUpload(id).catch(() => {
      // Unpublished uploads expire automatically if cleanup cannot finish now.
    });
  }

  async function upload(image: CommentImageDraft) {
    image.status = { state: "uploading" };
    try {
      const { id } = await uploadCommentImage(toValue(nodeId), image.file);
      if (!active || !images.value.includes(image)) {
        discard(id);

        return;
      }
      image.status = { state: "uploaded", id };
    } catch (err) {
      if (active && images.value.includes(image)) {
        image.status = {
          state: "failed",
          message: errorMessage(err, "图片上传失败"),
        };
      }
    }
  }

  function addFiles(files: Iterable<File>): string | null {
    let message: string | null = null;
    for (const file of files) {
      if (images.value.length >= COMMENT_IMAGE_LIMIT) {
        message = `每条进展最多 ${COMMENT_IMAGE_LIMIT} 张图片`;

        break;
      }
      if (!ACCEPTED_IMAGE_TYPES.has(file.type)) {
        message = "只支持 JPEG、PNG、WebP、GIF 图片";

        continue;
      }
      if (file.size > MAX_IMAGE_BYTES) {
        message = "单张图片不能超过 5 MB";

        continue;
      }
      const image = reactive<CommentImageDraft>({
        file,
        src: URL.createObjectURL(file),
        status: { state: "uploading" },
      });
      images.value.push(image);
      void upload(image);
    }

    return message;
  }

  function retry(image: CommentImageDraft) {
    if (active && image.status.state === "failed") {
      void upload(image);
    }
  }

  function remove(image: CommentImageDraft) {
    if (image.status.state === "uploading") {
      return;
    }
    images.value = images.value.filter((item) => item !== image);
    URL.revokeObjectURL(image.src);
    if (image.status.state === "uploaded") {
      discard(image.status.id);
    }
  }

  function clear(abandon: boolean) {
    const current = images.value;
    images.value = [];
    for (const image of current) {
      URL.revokeObjectURL(image.src);
      if (abandon && image.status.state === "uploaded") {
        discard(image.status.id);
      }
    }
  }

  function dispose(abandon: boolean) {
    active = false;
    clear(abandon);
  }

  return {
    images,
    pending,
    addFiles,
    retry,
    remove,
    clear,
    dispose,
  };
}

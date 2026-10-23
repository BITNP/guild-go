<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, useId, watch } from 'vue'
import { X } from '@lucide/vue'

const props = withDefaults(defineProps<{
  open: boolean
  title: string
  wide?: boolean
  busy?: boolean
  side?: 'left' | 'right'
}>(), {
  wide: false,
  busy: false,
})

const emit = defineEmits<{
  close: []
}>()

const panel = ref<HTMLDialogElement | null>(null)
const titleId = useId()

function close() {
  if (!props.busy) emit('close')
}

function syncOpen() {
  if (props.open && !panel.value?.open) panel.value?.showModal()
  else if (!props.open && panel.value?.open) panel.value.close()
}

watch(() => props.open, syncOpen, { flush: 'post' })
onMounted(syncOpen)
onBeforeUnmount(() => panel.value?.close())
</script>

<template>
  <Teleport to="body">
    <dialog
      ref="panel"
      :aria-labelledby="titleId"
      :aria-busy="busy"
      class="m-auto max-h-[calc(100dvh-2rem)] w-[calc(100%-2rem)] flex-col overflow-hidden rounded-lg border border-line bg-surface p-0 text-ink shadow-lg open:flex"
      :class="side
        ? ['h-dvh max-h-dvh max-w-80 rounded-none', side === 'left' ? 'mr-auto ml-0' : 'mr-0 ml-auto']
        : wide ? 'max-w-3xl' : 'max-w-xl'"
      @cancel.prevent="close"
      @click.self="close"
    >
      <template v-if="open">
        <header class="flex shrink-0 items-center justify-between gap-3 border-b border-line px-4 py-3">
          <h2 :id="titleId" class="min-w-0 text-sm font-semibold">{{ title }}</h2>
          <button type="button" class="icon-btn" :disabled="busy" aria-label="关闭" @click="close">
            <X class="h-4 w-4" />
          </button>
        </header>
        <div class="min-h-0 overflow-y-auto" :class="side ? 'flex-1' : 'px-4 py-4'">
          <slot />
        </div>
        <footer v-if="$slots.footer" class="flex shrink-0 flex-wrap items-center justify-end gap-2 border-t border-line px-4 py-3">
          <slot name="footer" />
        </footer>
      </template>
    </dialog>
  </Teleport>
</template>

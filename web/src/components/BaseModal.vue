<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { X } from '@lucide/vue'

const props = withDefaults(defineProps<{
  open: boolean
  title: string
  wide?: boolean
  zClass?: string
}>(), {
  wide: false,
  zClass: 'z-50',
})

const emit = defineEmits<{
  close: []
}>()

const panel = ref<HTMLElement | null>(null)

function close() {
  emit('close')
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') close()
}

watch(() => props.open, async (open) => {
  if (open) {
    document.body.classList.add('overflow-hidden')
    window.addEventListener('keydown', onKeydown)
    await nextTick()
    panel.value?.focus()
  } else {
    document.body.classList.remove('overflow-hidden')
    window.removeEventListener('keydown', onKeydown)
  }
}, { immediate: true })

onBeforeUnmount(() => {
  document.body.classList.remove('overflow-hidden')
  window.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 flex items-start justify-center overflow-y-auto bg-ink/35 p-4 backdrop-blur-[1px] sm:p-8"
      :class="zClass"
      @click.self="close"
    >
      <section
        ref="panel"
        tabindex="-1"
        role="dialog"
        aria-modal="true"
        :aria-label="title"
        class="my-auto flex w-full flex-col rounded-lg border border-line bg-surface shadow-xl outline-none"
        :class="wide ? 'max-w-3xl' : 'max-w-xl'"
      >
        <header class="flex items-center justify-between gap-4 border-b border-line px-4 py-3">
          <h2 class="text-sm font-semibold text-ink">{{ title }}</h2>
          <button type="button" class="icon-btn" aria-label="关闭" @click="close">
            <X class="h-4 w-4" />
          </button>
        </header>
        <div class="max-h-[calc(100vh-10rem)] overflow-y-auto px-4 py-4">
          <slot />
        </div>
        <footer v-if="$slots.footer" class="flex items-center justify-end gap-2 border-t border-line bg-surface-2 px-4 py-3">
          <slot name="footer" />
        </footer>
      </section>
    </div>
  </Teleport>
</template>
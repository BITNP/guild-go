<script setup lang="ts">
import { NButton, NModal } from 'naive-ui'
import { X } from '@lucide/vue'
withDefaults(
  defineProps<{
    open: boolean
    title: string
    busy?: boolean
    wide?: boolean
  }>(),
  { busy: false, wide: false },
)
const emit = defineEmits<{ close: [] }>()
</script>
<template>
  <NModal
    :show="open"
    preset="card"
    :title="title"
    :aria-label="title"
    :bordered="false"
    :closable="false"
    :mask-closable="!busy"
    :close-on-esc="!busy"
    :style="{ width: wide ? '760px' : '540px', maxWidth: 'calc(100vw - 32px)' }"
    header-style="padding:16px 24px;border-bottom:1px solid var(--guild-border)"
    content-style="max-height:calc(100dvh - 200px);overflow:auto;padding-top:20px"
    @update:show="(value) => !value && emit('close')"
  >
    <template #header-extra
      ><NButton
        quaternary
        circle
        :disabled="busy"
        aria-label="关闭"
        @click="emit('close')"
        ><X :size="18" /></NButton
    ></template>
    <slot />
    <template v-if="$slots.footer" #footer
      ><div class="modal-actions"><slot name="footer" /></div
    ></template>
  </NModal>
</template>
<style scoped>
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}
</style>

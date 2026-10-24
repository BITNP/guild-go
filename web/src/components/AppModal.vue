<script setup lang="ts">
import { NModal } from 'naive-ui'
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
    :closable="!busy"
    :mask-closable="!busy"
    :close-on-esc="!busy"
    :style="{ width: wide ? '760px' : '540px', maxWidth: 'calc(100vw - 32px)' }"
    @update:show="(value) => !value && emit('close')"
  >
    <slot />
  </NModal>
</template>

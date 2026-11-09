<script setup lang="ts">
const emit = defineEmits<{ close: [] }>();
withDefaults(
  defineProps<{
    open: boolean;
    title: string;
    busy?: boolean;
    wide?: boolean;
  }>(),
  { busy: false, wide: false },
);
</script>

<template>
  <NModal
    :aria-label="title"
    :closable="!busy"
    :close-on-esc="!busy"
    :mask-closable="!busy"
    preset="card"
    :show="open"
    :style="{ width: wide ? '760px' : '540px', maxWidth: 'calc(100vw - 32px)' }"
    :title
    @update:show="(value) => !value && emit('close')"
  >
    <slot />
  </NModal>
</template>

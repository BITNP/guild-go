<script setup lang="ts">
const emit = defineEmits<{ confirm: []; close: [] }>();
withDefaults(
  defineProps<{
    open: boolean;
    title: string;
    confirmText: string;
    busy: boolean;
    error: string | null;
  }>(),
  {},
);
</script>

<template>
  <NModal
    :aria-label="title"
    :closable="!busy"
    :close-on-esc="!busy"
    :mask-closable="!busy"
    :negative-button-props="{ disabled: busy }"
    negative-text="取消"
    :positive-button-props="{ type: 'error', loading: busy, disabled: busy }"
    :positive-text="confirmText"
    preset="dialog"
    :show="open"
    :title
    type="warning"
    @negative-click="emit('close')"
    @positive-click="
      () => {
        emit('confirm');
        return false;
      }
    "
    @update:show="(value) => !value && emit('close')"
  >
    <slot />
    <NAlert v-if="error" class="confirm-error" role="alert" type="error">
      {{ error }}
    </NAlert>
  </NModal>
</template>

<style scoped>
.confirm-error {
  margin-top: 16px;
}
</style>

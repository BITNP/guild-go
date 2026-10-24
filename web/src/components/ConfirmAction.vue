<script setup lang="ts">
import { NAlert, NModal } from 'naive-ui'
withDefaults(
  defineProps<{
    open: boolean
    title: string
    confirmText: string
    busy: boolean
    error: string | null
  }>(),
  {},
)
const emit = defineEmits<{ confirm: []; close: [] }>()
</script>
<template>
  <NModal
    :show="open"
    preset="dialog"
    type="warning"
    :title="title"
    :aria-label="title"
    :closable="!busy"
    :mask-closable="!busy"
    :close-on-esc="!busy"
    :positive-text="confirmText"
    negative-text="取消"
    :positive-button-props="{ type: 'error', loading: busy, disabled: busy }"
    :negative-button-props="{ disabled: busy }"
    @positive-click="
      () => {
        emit('confirm')
        return false
      }
    "
    @negative-click="emit('close')"
    @update:show="(value) => !value && emit('close')"
    ><slot /><NAlert
      v-if="error"
      type="error"
      role="alert"
      class="confirm-error"
      >{{ error }}</NAlert
    ></NModal
  >
</template>
<style scoped>
.confirm-error {
  margin-top: 16px;
}
</style>
